## sparrow run

Run sparrow

### Synopsis

Sparrow will be started with the provided configuration

```
sparrow run [flags]
```

### Options

```
      --apiAddress string          api: The address the server is listening on (default ":8080")
  -h, --help                       help for run
      --loaderFilePath string      file loader: The path to the file to read the runtime config from (default "config.yaml")
      --loaderHttpRetryCount int   http loader: Amount of retries trying to load the configuration (default 3)
      --loaderHttpRetryDelay int   http loader: The initial delay between retries in seconds (default 1)
      --loaderHttpTimeout int      http loader: The timeout for the http request in seconds (default 30)
      --loaderHttpToken string     http loader: Bearer token to authenticate the http endpoint
      --loaderHttpUrl string       http loader: The url where to get the remote configuration
      --loaderInterval int         defines the interval the loader reloads the configuration in seconds (default 300)
  -l, --loaderType string          defines the loader type that will load the checks configuration during the runtime. The fallback is the fileLoader (default "http")
      --sparrowName string         The DNS name of the sparrow
      --tmconfig string            target manager: The path to the file to read the target manager config from
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sparrow.yaml)
```

### SEE ALSO

* [sparrow](sparrow.md)	 - Sparrow, the infrastructure monitoring agent

