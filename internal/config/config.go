package config

import "os"

type Config struct {
	HTTPAddr       string
	SocialDataFile string
}

func Load() Config {
	addr := os.Getenv("BOOSTLAB_CONTROL_HTTP_ADDR")
	if addr == "" {
		addr = ":8090"
	}
	dataFile := os.Getenv("BOOSTLAB_SOCIAL_DATA_FILE")
	if dataFile == "" {
		dataFile = "./data/social.json"
	}
	return Config{
		HTTPAddr:       addr,
		SocialDataFile: dataFile,
	}
}
