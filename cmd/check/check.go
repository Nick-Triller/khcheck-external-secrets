package main

import (
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"time"

	"github.com/kuberhealthy/kuberhealthy/v2/pkg/checks/external/checkclient"
	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	secretsClient         v1.SecretInterface
	externalSecretsClient dynamic.ResourceInterface
	externalSecretObject  *unstructured.Unstructured
	backoffStrategy       backoff.BackOff
	rootCtx               context.Context
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	done := make(chan struct{})

	go check(done)

	select {
	case <-time.After(checkConfig.deadline.Sub(time.Now())):
		// Stop everything if deadline is reached. No cleanup will happen.
		log.Fatalf("Check deadline %s reached. Stopping everything.", checkConfig.deadline)
	case <-done:
		log.Infof("Everything succeeded, terminating program.")
	case sig := <-signalChan:
		log.Infof("Received signal %v. Exiting without cleanup.", sig)
		os.Exit(1)
	}
}

// getK8sConfig creates a client-go kubernetes rest checkConfig
func getK8sConfig() *rest.Config {
	// Try loading kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	k8sConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		nil).ClientConfig()
	if err == nil {
		return k8sConfig
	}
	// Try in-cluster config instead
	k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	return k8sConfig
}

// setup initializes variables including Kubernetes clients
func setup() error {
	// Configure backoff strategy
	backoffStrategy = backoff.NewConstantBackOff(3 * time.Second)

	// Root context. Cancel discarded as the check pod is short-lived
	rootCtx, _ = context.WithCancel(context.Background())

	// Read external secret
	dat, err := ioutil.ReadFile(checkConfig.manifestPath)
	if err != nil {
		return err
	}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	externalSecretObject = &unstructured.Unstructured{}
	_, gvk, err := dec.Decode(dat, nil, externalSecretObject)
	if err != nil {
		return err
	}
	checkConfig.externalSecretName = externalSecretObject.GetName()

	// Create k8s clients
	k8sConfig := getK8sConfig()
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return err
	}
	dynamicClient, err := dynamic.NewForConfig(k8sConfig)
	if err != nil {
		return err
	}
	externalSecretGVR := gvk.GroupVersion().WithResource("externalsecrets")
	externalSecretsClient = dynamicClient.Resource(externalSecretGVR).Namespace(checkConfig.namespace)
	secretsClient = clientset.CoreV1().Secrets(checkConfig.namespace)
	return nil
}

// check performs the check including cleanup
func check(done chan struct{}) {
	// Handle panics
	var r interface{}
	defer func() {
		r = recover()
		if r != nil {
			log.Infoln("Recovered panic:", r)
			reportFailure("Unexpected panic", true)
		}
	}()

	// Indicate program can be terminated
	defer func() {
		done <- struct{}{}
	}()

	err := setup()
	if err != nil {
		log.Errorf("Setup failed: %v", err)
		reportFailure("Failed setup", true)
	}

	// Cleanup before the check in case there are resources from previous checks.
	err = cleanup()
	if err != nil {
		log.Errorf("Cleanup before check failed: %v", err)
		reportFailure("Cleanup before check failed", false)
	}
	// Wait a few seconds for removal of resources
	time.Sleep(5 * time.Second)

	// Create ExternalSecret
	log.Infoln("Creating external secret...")
	_, err = externalSecretsClient.Create(timeout(), externalSecretObject, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("Creating ExternalSecret failed: %v", err)
		reportFailure("Could not create external secret", true)
	} else {
		log.Infof("Created ExternalSecret %q.\n", checkConfig.externalSecretName)
	}

	// Validate operator creates matching secret
	err = backoff.Retry(func() error {
		log.Infoln("Checking operator created matching secret...")
		_, err := secretsClient.Get(timeout(), checkConfig.externalSecretName, metav1.GetOptions{})
		if err != nil {
			log.Infof("Error while checking operator created matching secret: %v", err)
		}
		return err
	}, backoff.WithMaxRetries(backoffStrategy, 5))

	if err != nil {
		log.Errorf("Validating operator created matching secret failed: %v", err)
		reportFailure("Could not validate operator created matching secret", true)
	}

	log.Infof("Check finished. Starting cleanup.")
	err = cleanup()
	if err != nil {
		log.Errorf("Cleanup after successful check failed: %v", err)
		reportFailure("Cleanup after check failed", false)
	} else {
		reportSuccess()
	}
}

// reportFailure reports check failures to kuberhealthy and exits the program. Optionally, attempts cleanup.
func reportFailure(error string, attemptCleanup bool) {
	failures := []string{error}
	// Attempt cleanup
	if attemptCleanup {
		log.Info("Attempting cleanup before reporting kuberhealthy failures")
		err := cleanup()
		if err != nil {
			log.Warnf("Cleanup failed: %v", err)
			failures = append(failures, "Cleanup failed")
		}
	} else {
		log.Info("Not attempting cleanup before reporting kuberhealthy failures")
	}

	log.Warnf("Reporting failures to kuberhealthy: %v", failures)
	if err := checkclient.ReportFailure(failures); err != nil {
		log.Errorf("Error reporting failures to kuberhealthy: %v\n", err)
	}

	os.Exit(1)
}

// reportSuccess reports check succeess to kuberhealthy.
func reportSuccess() {
	log.Infoln("Reporting success to kuberhealthy")
	if err := checkclient.ReportSuccess(); err != nil {
		log.Errorf("Error reporting success to kuberhealthy: %v\n", err)
		os.Exit(1)
	}
}

// cleanup deletes Kubernetes resources that were created by this or previous check runs.
func cleanup() error {
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	handleErr := func(err error) error {
		if err != nil && errors.IsNotFound(err) {
			// fine, resource did not exist
			return nil
		}
		return err
	}
	// Delete ExternalSecret
	log.Infoln("Deleting ExternalSecret if it exists")
	err := externalSecretsClient.Delete(timeout(), externalSecretObject.GetName(), deleteOptions)
	err = handleErr(err)
	if err != nil {
		return err
	}
	// Delete secret in case it's not cleaned up automatically because something is broken
	log.Infoln("Deleting Secret if it exists")
	err = secretsClient.Delete(timeout(), externalSecretObject.GetName(), deleteOptions)
	err = handleErr(err)
	if err != nil {
		return err
	}
	return nil
}

// timeout creates a context with a default timeout of 6 seconds.
// Cancel function is discarded which is a potential resource leak.
func timeout() context.Context {
	// Discarding cancel is ok because the checker pod is short-lived.
	ctx, _ := context.WithTimeout(rootCtx, 6*time.Second)
	return ctx
}
