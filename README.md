# Sparrow - Infrastructure Monitoring<!-- omit from toc -->

<p align="center">
    <a href="/../../commits/" title="Last Commit"><img src="https://img.shields.io/github/last-commit/caas-team/sparrow?style=flat"></a>
    <a href="/../../issues" title="Open Issues"><img src="https://img.shields.io/github/issues/caas-team/sparrow?style=flat"></a>
    <a href="./LICENSE" title="License"><img src="https://img.shields.io/badge/License-Apache%202.0-green.svg?style=flat"></a>
</p>

- [About this component](#about-this-component)
- [Installation](#installation)
    - [Binary](#binary)
    - [Container Image](#container-image)
    - [Helm](#helm)
- [Usage](#usage)
    - [Container Image](#container-image-1)
- [Configuration](#configuration)
    - [Startup](#startup)
        - [Loader](#loader)
    - [Runtime](#runtime)
    - [TargetManager](#targetmanager)
    - [Check: Health](#check-health)
    - [Check: Latency](#check-latency)
    - [API](#api)
- [Code of Conduct](#code-of-conduct)
- [Working Language](#working-language)
- [Support and Feedback](#support-and-feedback)
- [How to Contribute](#how-to-contribute)
- [Licensing](#licensing)

The `sparrow` is an infrastructure monitoring tool. The binary includes several checks (e.g. health check) that will be
executed periodically.

## About this component

The `sparrow` performs several checks to monitor the health of the infrastructure and network from its point of view.
The following checks are available:

1. Health check - `health`: The `sparrow` is able perform an http-based (HTTP/1.1) health check to provided endpoints.
   The `sparrow` will expose its own health check endpoint as well.

2. Latency check - `latency`: The `sparrow` is able to communicate with other `sparrow` instances to calculate the time
   a request takes to the target and back. The check is http (HTTP/1.1) based as well.

## Installation

The `sparrow` is provided as an small binary & a container image.

Please see the [release notes](https://github.com/caas-team/sparrow/releases) for to get the latest version.

### Binary

The binary is available for several distributions. Currently the binary needs to be installed from a provided bundle or
source.

```sh
curl https://github.com/caas-team/sparrow/releases/download/v${RELEASE_VERSION}/sparrow_${RELEASE_VERSION}_linux_amd64.tar.gz -Lo sparrow.tar.gz
curl https://github.com/caas-team/sparrow/releases/download/v${RELEASE_VERSION}/sparrow_${RELEASE_VERSION}_checksums.txt -Lo checksums.txt
```

For example release `v0.0.1`:

```sh
curl https://github.com/caas-team/sparrow/releases/download/v0.0.1/sparrow_0.0.1_linux_amd64.tar.gz -Lo sparrow.tar.gz
curl https://github.com/caas-team/sparrow/releases/download/v0.0.1/sparrow_0.0.1_checksums.txt -Lo checksums.txt
```

Extract the binary:

```sh
tar -xf sparrow.tar.gz
```

### Container Image

The [sparrow container images](https://github.com/caas-team/sparrow/pkgs/container/sparrow) for
dedicated [release](https://github.com/caas-team/sparrow/releases) can be found in the GitHub registry.

### Helm

Sparrow can be install via Helm Chart. The chart is provided in the GitHub registry:

```sh
helm -n sparrow upgrade -i sparrow oci://ghcr.io/caas-team/charts/sparrow --version 1.0.0 --create-namespace
```

The default settings are fine for a local running configuration. With the default Helm values the sparrow loader uses a
runtime configuration that is provided in a ConfigMap. The ConfigMap can be set by defining the `runtimeConfig` section.

To be able to load the configuration during the runtime dynamically, the sparrow loader needs to be set to type `http`.

Use the following configuration values to use a runtime configuration by the `http` loader:

```yaml
startupConfig:
  loaderType: http
  loaderHttpUrl: https://url-to-runtime-config.de/api/config%2Eyaml

runtimeConfig: { }
```

For all available value options see [Chart README](./chart/README.md).

Additionally check out the sparrow [configuration](#configuration) variants.

## Usage

Use `sparrow run` to execute the instance using the binary.

### Container Image

Run a `sparrow` container by using e.g. `docker run ghcr.io/caas-team/sparrow`.

Pass the available configuration arguments to the container e.g. `docker run ghcr.io/caas-team/sparrow --help`.

Start the instance using a mounted startup configuration file
e.g. `docker run -v /config:/config  ghcr.io/caas-team/sparrow --config /config/config.yaml`.

## Configuration

The configuration is divided into two parts. The startup configuration and the runtime configuration. The startup
configuration is a technical configuration to configure the `sparrow` instance itself. The runtime configuration will be
loaded by the `loader` from a remote endpoint. This configuration consist of the checks configuration.

### Startup

The available configuration options can found in the [CLI flag documentation](docs/sparrow.md).

The `sparrow` is able to get the startup configuration from different sources as follows.

Priority of configuration (high to low):

1. CLI flags
2. Environment variables
3. Defined configuration file
4. Default configuration file

#### Loader

The loader component of the `sparrow` will load the [Runtime](#runtime) configuration dynamically.

The loader can be selected by specifying the `loaderType` configuration parameter.

The default loader is an `http` loader that is able to get the runtime configuration from a remote endpoint.

Available loader:

- `http`: The default. Loads configuration from a remote endpoint. Token authentication is available. Additional
  configuration parameter have the prefix `loaderHttp`.
- `file` (experimental): Loads configuration once from a local file. Additional configuration parameter have the
  prefix `loaderFile`. This is just for development purposes.

### Runtime

Besides the technical startup configuration the configuration for the `sparrow` checks is loaded dynamically from an
http endpoint. The `loader` is able to load the configuration dynamically during the runtime. Checks can be enabled,
disabled and configured. The available loader confutation options for the startup configuration can be found
in [here](sparrow_run.md)

Example format of a runtime configuration:

```YAML
apiVersion: 0.0.1
kind: Config
checks:
  health:
    enabled: true
```

### Target Manager

The `sparrow` is able to manage the targets for the checks and register the `sparrow` as target on a (remote) backend.
This is done via a `TargetManager` interface, which can be configured on startup. The available configuration options
are listed below:

| Type                                 | Description                                                                          | Default              |
|--------------------------------------|--------------------------------------------------------------------------------------|----------------------|
| `targetManager.type`                 | The kind of target manager to use.                                                   | `gitlab`             |
| `targetManager.checkInterval`        | The interval in seconds to check for new targets.                                    | `300`                |
| `targetManager.unhealthyThreshold`   | The threshold in seconds to mark a target as unhealthy and remove it from the state. | `600`                |
| `targetManager.registrationInterval` | The interval in seconds to register the current sparrow at the targets backend.      | `300`                |
| `targetManager.gitlab.token`         | The token to authenticate against the gitlab instance.                               | `""`                 |
| `targetManager.gitlab.url`           | The base URL of the gitlab instance.                                                 | `https://gitlab.com` |
| `targetManager.gitlab.projectID`     | The project ID of the gitlab project to use as a remote state backend.               | `""`                 |

The Gitlab target manager uses a gitlab project as the remote state backend. The various `sparrow` instances will
register themselves as targets in the project. The `sparrow` instances will also check the project for new targets and
add them to the local state. The registration is done by committing a "state" file in the main branch of the repository,
which is named after the DNS name of the `sparrow`. The state file contains the following information:

```json
{
  "url": "https://<SPARROW_DNS_NAME",
  "lastSeen": "2021-09-30T12:00:00Z"
}
```

### Check: Health

Available configuration options:

- `checks.health.enabled` (boolean): Currently not used.
- `checks.health.targets` (list of strings): List of targets to send health probe. Needs to be a valid url. Can be
  another `sparrow` instance. Use health endpoint, e.g. `https://sparrow-dns.telekom.de/checks/health`. The
  remote `sparrow` instance needs the `healthEndpoint` enabled.
- `checks.health.healthEndpoint` (boolean): Needs to be activated when the `sparrow` should expose its own health
  endpoint. Mandatory if another `sparrow` instance wants perform a health check.

Example configuration:

```YAML
checks:
  health:
    enabled: true
    targets:
      - "https://gitlab.devops.telekom.de"
    healthEndpoint: false
```

### Check: Latency

Available configuration options:

- `checks`
    - `latency`
        - `enabled` (boolean): Currently not used.
        - `interval` (integer): Interval in seconds to perform the latency check.
        - `timeout` (integer): Timeout in seconds for the latency check.
        - `retry`
            - `count` (integer): Number of retries for the latency check.
            - `delay` (integer): Delay in seconds between retries for the latency check.
        - `targets` (list of strings): List of targets to send latency probe. Needs to be a valid url. Can be
          another `sparrow` instance. Use latency endpoint, e.g. `https://sparrow-dns.telekom.de/checks/latency`. The
          remote `sparrow` instance needs the `latencyEndpoint` enabled.
        - `latencyEndpoint` (boolean): Needs to be activated when the `sparrow` should expose its own latency endpoint.
          Mandatory if another `sparrow` instance wants perform a latency check.
          Example configuration:

```yaml
checks:
  latency:
    enabled: true
    interval: 1
    timeout: 3
    retry:
      count: 3
      delay: 1
    targets:
      - https://example.com/
      - https://google.com/
```

### API

The `sparrow` exposes an API that does provide access to the check results. Each check will register its own endpoint
at `/v1/metrics/{check-name}`. The API definition will be exposed at `/openapi`

## Code of Conduct

This project has adopted the [Contributor Covenant](https://www.contributor-covenant.org/) in version 2.1 as our code of
conduct. Please see the details in our [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). All contributors must abide by the code
of conduct.

## Working Language

We decided to apply *English* as the primary project language.

Consequently, all content will be made available primarily in English.
We also ask all interested people to use English as the preferred language to create issues,
in their code (comments, documentation, etc.) and when you send requests to us.
The application itself and all end-user facing content will be made available in other languages as needed.

## Support and Feedback

The following channels are available for discussions, feedback, and support requests:

| Type       | Channel                                                                                                                                                |
|------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Issues** | <a href="/../../issues/new/choose" title="General Discussion"><img src="https://img.shields.io/github/issues/caas-team/sparrow?style=flat-square"></a> |

## How to Contribute

Contribution and feedback is encouraged and always welcome. For more information about how to contribute, the project
structure, as well as additional contribution information, see our [Contribution Guidelines](./CONTRIBUTING.md). By
participating in this project, you agree to abide by its [Code of Conduct](./CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright (c) 2023 Deutsche Telekom IT GmbH.

Licensed under the **Apache License, Version 2.0** (the "License"); you may not use this file except in compliance with
the License.

You may obtain a copy of the License at <https://www.apache.org/licenses/LICENSE-2.0>.

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "
AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the [LICENSE](./LICENSE) for
the specific language governing permissions and limitations under the License.
