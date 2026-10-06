package config

import "os"

type Config struct {
	HTTPAddr string
}

func Load() Config {
	addr := os.Getenv("BOOSTLAB_CONTROL_HTTP_ADDR")
	if addr == "" {
		addr = ":8090"
	}
	return Config{HTTPAddr: addr}
}
