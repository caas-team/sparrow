# sparrow

![Version: 0.0.4](https://img.shields.io/badge/Version-0.0.4-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.3.0](https://img.shields.io/badge/AppVersion-v0.3.0-informational?style=flat-square)

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
| config | object | `{}` | startup configuration of the Sparrow see: https://github.com/caas-team/sparrow/blob/main/docs/sparrow_run.md |
| env | object | `{}` |  |
| extraArgs | object | `{}` | extra command line start parameters see: https://github.com/caas-team/sparrow/blob/main/docs/sparrow_run.md |
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
| nameOverride | string | `""` |  |
| networkPolicies | object | `{"proxy":{"enabled":false}}` | define a network policy that will open egress traffic to a proxy |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| podSecurityContext.fsGroup | int | `1000` |  |
| podSecurityContext.supplementalGroups[0] | int | `1000` |  |
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
| tolerations | list | `[]` |  |

