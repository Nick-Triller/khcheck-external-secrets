package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/kuberhealthy/kuberhealthy/v2/pkg/checks/external/checkclient"
	log "github.com/sirupsen/logrus"
)

// CheckConfig contains check configuration
type CheckConfig struct {
	externalSecretName string
	manifestPath       string
	namespace          string
	deadline           time.Time
}

var checkConfig = getCheckConfig()

// getCheckConfig returns an application CheckConfig
func getCheckConfig() CheckConfig {
	c := CheckConfig{}
	manifestPath := os.Getenv("KH_CHECK_EXTERNAL_SECRETS_MANIFEST_PATH")
	if manifestPath == "" {
		defaultManifestPath := "./external-secret-manifest.yml"
		log.Infof("Env var KHCHECK_EXTERNAL_SECRETS_MANIFEST_PATH not set, defaulting manifest path to %s",
			defaultManifestPath)
		manifestPath = defaultManifestPath
	}
	c.manifestPath = manifestPath
	// Check manifest file exists
	_, err := os.Stat(c.manifestPath)
	if os.IsNotExist(err) {
		msg := fmt.Sprintf("File %s does not exist", c.manifestPath)
		reportFailure(msg, false)
		log.Fatalf(msg)
	}

	namespaceFilePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	ns, err := ioutil.ReadFile(namespaceFilePath)
	if err == nil {
		log.Infof("Using namespace %s from %s", ns, namespaceFilePath)
		c.namespace = string(ns)
	}
	if c.namespace == "" {
		defaultNs := "kuberhealthy"
		log.Infof("Using default namespace %s", defaultNs)
		c.namespace = defaultNs
	}

	deadline, err := checkclient.GetDeadline()
	if err != nil {
		defaultDeadlineDuration := 120 * time.Second
		log.Infof("Using default deadline Now + %s", defaultDeadlineDuration)
		deadline = time.Now().Add(defaultDeadlineDuration)
	}
	c.deadline = deadline

	return c
}
