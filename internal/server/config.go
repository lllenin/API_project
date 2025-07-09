package server

type Config struct {
	Addr string
}

func NewConfig() *Config {
	return &Config{
		Addr: ":8080",
	}
}

var DefaultConfig = NewConfig()
