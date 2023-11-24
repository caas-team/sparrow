package config

type RunFlagsNameMapping struct {
	ApiListeningAddress string

	LoaderType           string
	LoaderInterval       string
	LoaderHttpUrl        string
	LoaderHttpToken      string
	LoaderHttpTimeout    string
	LoaderHttpRetryCount string
	LoaderHttpRetryDelay string
}
