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
  - [Image](#image)
- [Configuration](#configuration)
  - [Startup](#startup)
    - [Example configuration](#example-configuration)
    - [Loader](#loader)
  - [Runtime](#runtime)
  - [Target Manager](#target-manager)
  - [Check: Health](#check-health)
    - [Health Metrics](#health-metrics)
  - [Check: Latency](#check-latency)
    - [Latency Metrics](#latency-metrics)
- [API](#api)
- [Metrics](#metrics)
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

1. Health check - `health`: The `sparrow` is able to perform an HTTP-based (HTTP/1.1) health check to the provided
   endpoints.
   The `sparrow` will expose its own health check endpoint as well.

2. Latency check - `latency`: The `sparrow` is able to communicate with other `sparrow` instances to calculate the time
   a request takes to the target and back. The check is http (HTTP/1.1) based as well.

## Installation

The `sparrow` is provided as a small binary & a container image.

Please see the [release notes](https://github.com/caas-team/sparrow/releases) for to get the latest version.

### Binary

The binary is available for several distributions. Currently, the binary needs to be installed from a provided bundle or
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

Sparrow can be installed via Helm Chart. The chart is provided in the GitHub registry:

```sh
helm -n sparrow upgrade -i sparrow oci://ghcr.io/caas-team/charts/sparrow --version 1.0.0 --create-namespace
```

The default settings are fine for a local running configuration. With the default Helm values, the sparrow loader uses a
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

Use `sparrow run` to execute the instance using the binary. A `sparrowName` (a valid DNS name) is required to be passed,
else the sparrow will not start:

```sh
sparrow run --sparrowName sparrow.telekom.de
```

### Image

Run a `sparrow` container by using e.g. `docker run ghcr.io/caas-team/sparrow`.

Pass the available configuration arguments to the container e.g. `docker run ghcr.io/caas-team/sparrow --help`.

Start the instance using a mounted startup configuration file
e.g. `docker run -v /config:/config  ghcr.io/caas-team/sparrow --config /config/config.yaml`.

## Configuration

The configuration is divided into two parts. The startup configuration and the runtime configuration. The startup
configuration is a technical configuration to configure the `sparrow` instance itself. The runtime configuration will be
loaded by the `loader` from a remote endpoint. This configuration consists of the checks' configuration.

### Startup

The available configuration options can be found in the [CLI flag documentation](docs/sparrow.md).

The `sparrow` is able to get the startup configuration from different sources as follows.

Priority of configuration (high to low):

1. CLI flags
2. Environment variables
3. Defined configuration file
4. Default configuration file

Every value in the config file can be set through environment variables.

You can set a token for the http loader:

```bash
export SPARROW_LOADER_HTTP_TOKEN="xxxxxx"
```

Or for any other config attribute:

```bash
export SPARROW_ANY_OTHER_OPTION="Some value"
```

Just write out the path to the attribute, delimited by `_`.

#### Example configuration

```yaml
# DNS sparrow is exposed on 
name: sparrow.example.com
# Selects and configures a loader for continuosly fetching the configuration at runtime
loader:
  # defines which loader to use. Options: "file | http" 
  type: http
  # the interval in which sparrow tries to fetch a new configuration
  interval: 30s
  # config specific to the http loader
  http:
    # The url where the config is located
    url: https://myconfig.example.com/config.yaml
    # This token is passed in the Authorization header, when refreshing the config
    token: xxxxxxx
    # A timeout for the config refresh
    timeout: 30s
    retry:
      # How long to wait in between retries
      delay: 10s
      # How many times to retry
      count: 3

  # config specific to the file loader
  # The file loader is not intended for production use and does 
  # not refresh the config after reading it the first time
  file:
    # where to read the runtime config from
    path: ./config.yaml

# Configures the api
api:
  # Which address to expose sparrows rest api on
  address: :8080

# Configures the targetmanager
targetManager:
  # time between checking for new targets
  checkInterval: 1m
  # how often the instance should register itself as a global target
  registrationInterval: 1m
  # the amount of time a target can be
  # unhealthy before it is removed from the global target list
  unhealthyThreshold: 3m
  # Configuration options for the gitlab target manager
  gitlab:
    # The url of your gitlab host
    baseUrl: https://gitlab.com
    # Your gitlab api token 
    # you can also set this value through the 
    # SPARROW_TARGETMANAGER_GITLAB_TOKEN environment variable
    token: glpat-xxxxxxxx
    # the id of your gitlab project. This is where sparrow will register itself
    # and grab the list of other sparrows from
    projectId: 18923
```

#### Loader

The loader component of the `sparrow` will load the [Runtime](#runtime) configuration dynamically.

The loader can be selected by specifying the `loaderType` configuration parameter.

The default loader is an `http` loader that is able to get the runtime configuration from a remote endpoint.

Available loader:

- `http`: The default. Loads configuration from a remote endpoint. Token authentication is available. Additional
  configuration parameters have the prefix `loaderHttp`.
- `file` (experimental): Loads configuration once from a local file. Additional configuration parameters have the
  prefix `loaderFile`. This is just for development purposes.

### Runtime

In addition to the technical startup configuration, the `sparrow` checks' configuration can be dynamically loaded from
an HTTP endpoint during runtime. The `loader` is capable of dynamically loading and configuring checks. You can enable,
disable, and configure checks as needed.

For detailed information on available loader configuration options, please refer
to [this documentation](docs/sparrow_run.md).

Example format of a runtime configuration:

```YAML
apiVersion: 0.0.1
kind: Config
checks:
  health:
    targets: [ ]
```

### Target Manager

The `sparrow` is able to manage the targets for the checks and register the `sparrow` as target on a (remote) backend.
This is done via a `TargetManager` interface, which can be configured on startup. The available configuration options
are listed below and can be set in the startup YAML configuration file, as shown in
the [example configuration](#example-configuration).

| Type                                 | Description                                                                   | Default              |
| ------------------------------------ | ----------------------------------------------------------------------------- | -------------------- |
| `targetManager.checkInterval`        | The interval in seconds to check for new targets.                             | `300s`               |
| `targetManager.unhealthyThreshold`   | The threshold in seconds to mark a target as unhealthy and remove it from the |
| state.                               | `600s`                                                                        |
| `targetManager.registrationInterval` | The interval in seconds to register the current sparrow at the targets        |
| backend.                             | `300s`                                                                        |
| `targetManager.gitlab.token`         | The token to authenticate against the gitlab instance.                        | `""`                 |
| `targetManager.gitlab.baseUrl`       | The base URL of the gitlab instance.                                          | `https://gitlab.com` |
| `targetManager.gitlab.projectId`     | The project ID of the gitlab project to use as a remote state                 |
| backend.                             | `""`                                                                          |

Currently, only one target manager exists: the Gitlab target manager. It uses a gitlab project as the remote state
backend. The various `sparrow` instances will
register themselves as targets in the project. The `sparrow` instances will also check the project for new targets and
add them to the local state. The registration is done by committing a "state" file in the main branch of the repository,
which is named after the DNS name of the `sparrow`. The state file contains the following information:

```json
{
  "url": "https://<SPARROW_DNS_NAME>",
  "lastSeen": "2021-09-30T12:00:00Z"
}
```

### Check: Health

Available configuration options:

- `checks`
  - `health`
    - `interval` (duration): Interval to perform the health check.
    - `timeout` (duration): Timeout for the health check.
    - `retry`
      - `count` (integer): Number of retries for the health check.
      - `delay` (duration): Delay between retries for the health check.
    - `targets` (list of strings): List of targets to send health probe. Needs to be a valid url. Can be
      another `sparrow` instance. Automatically used when target manager is activated otherwise use the health endpoint of
      the remote sparrow, e.g. `https://sparrow-dns.telekom.de/checks/health`.
      
Example configuration:
```yaml
checks:
  health:
    interval: 10s
    timeout: 30s
    retry:
      count: 3
      delay: 1s
    targets:
      - https://example.com/
      - https://google.com/
```

#### Health Metrics

- `sparrow_health_up`
  - Type: Gauge
  - Description: Health of targets
  - Labelled with `target`

### Check: Latency

Available configuration options:

- `checks`
  - `latency`
    - `interval` (duration): Interval to perform the latency check.
    - `timeout` (duration): Timeout for the latency check.
    - `retry`
      - `count` (integer): Number of retries for the latency check.
      - `delay` (duration): Delay between retries for the latency check.
    - `targets` (list of strings): List of targets to send latency probe. Needs to be a valid url. Can be
      another `sparrow` instance. Automatically used when the target manager is enabled otherwise
      use latency endpoint, e.g. `https://sparrow-dns.telekom.de/checks/latency`.
      
Example configuration:
```yaml
checks:
  latency:
    interval: 10s
    timeout: 30s
    retry:
      count: 3
      delay: 1s
    targets:
      - https://example.com/
      - https://google.com/
```

#### Latency Metrics

- `sparrow_latency_duration_seconds`
  - Type: Gauge
  - Description: Latency with status information of targets
  - Labelled with `target` and `status`

- `sparrow_latency_count`
  - Type: Counter
  - Description: Count of latency checks done
  - Labelled with `target`

- `sparrow_latency_duration`
  - Type: Histogram
  - Description: Latency of targets in seconds
  - Labelled with `target`

## API

The `sparrow` exposes an API that does provide access to the check results. Each check will register its own endpoint
at `/v1/metrics/{check-name}`. The API definition will be exposed at `/openapi`

## Metrics

The `sparrow` is providing a `/metrics` endpoint to expose application metrics. Besides metrics about runtime
information the sparrow is also provided `Check` specific metrics. See the Checks section for more information.

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
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
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
