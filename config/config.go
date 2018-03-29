package config

type Config struct {
	UseResolverMethods bool
}

func Default() *Config {
	return &Config{
		UseResolverMethods: true,
	}
}
