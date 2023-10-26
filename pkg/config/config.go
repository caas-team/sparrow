package config

type Config struct {
	Checks  map[string]any
	Updated chan bool
}

func NewConfig() *Config {
	// TODO read this from config file
	return &Config{
		Checks: map[string]any{},
	}
}
