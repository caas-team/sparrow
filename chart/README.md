# sparrow

![Version: 0.0.4](https://img.shields.io/badge/Version-0.0.4-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.5.0](https://img.shields.io/badge/AppVersion-v0.5.0-informational?style=flat-square)

A Helm chart to install Sparrow

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| eumel8 | <f.kloeker@telekom.de> | <https://www.telekom.com> |
| y-eight | <maximilian.schubert@telekom.de> | <https://www.telekom.com> |

## Source Code

* <https://github.com/caas-team/sparrow>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| checksConfig | object | `{}` | Check configuration of the Sparrow read on runtime see: https://github.com/caas-team/sparrow?tab=readme-ov-file#checks |
| env | object | `{}` |  |
| envFromSecrets | list | `[]` | extra environment variables Allows you to set environment variables through secrets you defined outside of the helm chart Useful for sensitive information like the http loader token |
| extraArgs | object | `{}` | Extra command line start parameters see: https://github.com/caas-team/sparrow/blob/main/docs/sparrow_run.md |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/caas-team/sparrow"` |  |
| image.tag | string | `""` | Overrides the image tag whose default is the chart appVersion. |
| imagePullSecrets | list | `[]` |  |
| ingress.annotations | object | `{}` |  |
| ingress.className | string | `""` |  |
| ingress.enabled | bool | `false` |  |
| ingress.hosts[0].host | string | `"chart-example.local"` |  |
| ingress.hosts[0].paths[0].path | string | `"/"` |  |
| ingress.hosts[0].paths[0].pathType | string | `"ImplementationSpecific"` |  |
| ingress.tls | list | `[]` |  |
| livenessProbe | object | `{"enabled":false,"failureThreshold":3,"initialDelaySeconds":30,"path":"/","periodSeconds":10,"successThreshold":1,"timeoutSeconds":1}` | Specifies the configuration for a liveness probe to check if the sparrow is still running. Ref: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/ |
| nameOverride | string | `""` |  |
| networkPolicies | object | `{"proxy":{"enabled":false}}` | define a network policy that will open egress traffic to a proxy |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| podSecurityContext.fsGroup | int | `1000` |  |
| podSecurityContext.supplementalGroups[0] | int | `1000` |  |
| readinessProbe | object | `{"enabled":true,"failureThreshold":3,"initialDelaySeconds":5,"path":"/","periodSeconds":10,"successThreshold":1,"timeoutSeconds":1}` | Specifies the configuration for a readiness probe to check if the sparrow is ready to serve traffic. Ref: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/ |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.privileged | bool | `false` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| securityContext.runAsGroup | int | `1000` |  |
| securityContext.runAsUser | int | `1000` |  |
| service.port | int | `8080` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.automount | bool | `true` | Automatically mount a ServiceAccount's API credentials? |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| serviceMonitor | object | `{"enabled":false,"interval":"30s","labels":{},"scrapeTimeout":"5s"}` | Configure a service monitor for prometheus-operator |
| serviceMonitor.enabled | bool | `false` | Enable the serviceMonitor |
| serviceMonitor.interval | string | `"30s"` | Sets the scrape interval |
| serviceMonitor.labels | object | `{}` | Additional label added to the service Monitor |
| serviceMonitor.scrapeTimeout | string | `"5s"` | Sets the scrape timeout |
| sparrowConfig | object | `{"loader":{"file":{"path":"/config/checks.yaml"},"interval":"30s","type":"file"},"name":"sparrow.com"}` | Sparrow configuration read on startup see: https://github.com/caas-team/sparrow/blob/main/docs/sparrow_run.md |
| startupProbe | object | `{"enabled":false,"failureThreshold":10,"initialDelaySeconds":10,"path":"/","periodSeconds":5,"successThreshold":1,"timeoutSeconds":1}` | Specifies the configuration for a startup probe to check if the sparrow application is started. Ref: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/ |
| tolerations | list | `[]` |  |

