package config

type Config struct {
	Checks map[string]any
}

func NewConfig() *Config {
	// TODO read this from config file
	return &Config{
		Checks: map[string]any{},
	}
}
