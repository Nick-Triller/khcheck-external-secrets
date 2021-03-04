# khcheck-external-secrets

![Version: 1.0.0](https://img.shields.io/badge/Version-1.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.0.0](https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square)

An external Kuberhealthy check that checks External Secrets Operator health

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| externalSecretTemplate | string | `"apiVersion: kubernetes-client.io/v1\nkind: ExternalSecret\nmetadata:\n  name: my-test-secret\nspec:\n  backendType: vault\n  data:\n    - name: user\n      key: secrets/data/khcheck-external-secrets\n      property: user\n    - name: pass\n      key: secrets/data/khcheck-external-secrets\n      property: password\n"` |  |
| externalSecretTemplatePath | string | `"/external-secret-manifest.yml"` | Mount location for the ExternalSecret template file |
| extraEnvs | list | `[]` | Additional environment variables to pass to the check pod |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"Always"` |  |
| image.repository | string | `"docker.io/nicktriller/khcheck-external-secrets"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| podAnnotations | object | `{}` | Check pod annotations |
| podSecurityContext | object | `{}` | Check pod security context |
| reportDelay | string | `"6s"` |  |
| reportFailure | string | `"false"` |  |
| resources | object | `{"limits":{"cpu":"100m","memory":"64Mi"},"requests":{"cpu":"100m","memory":"64Mi"}}` | Check pod resource limits |
| runInterval | string | `"2m"` | The interval that Kuberhealthy will run your check on |
| securityContext | object | `{"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true,"runAsNonRoot":true,"runAsUser":1000}` | Check pod container security context |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `"kuberhealthy-external-secrets-sa"` |  |
| timeout | string | `"3m"` | After this much time, Kuberhealthy will kill your check and consider it "failed" |

