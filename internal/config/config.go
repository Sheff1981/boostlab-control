package config

import "os"

type Config struct {
	HTTPAddr           string
	SocialDataFile     string
	ProvisioningSecret string
	DeviceAuthDataFile  string
	EnrollmentCode      string
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
		HTTPAddr:           addr,
		SocialDataFile:     dataFile,
		ProvisioningSecret: os.Getenv("BOOSTLAB_PROVISIONING_SECRET"),
		DeviceAuthDataFile: envOr("BOOSTLAB_DEVICE_AUTH_DATA_FILE", "./data/devices.json"),
		EnrollmentCode:     os.Getenv("BOOSTLAB_ENROLLMENT_CODE"),
	}
}


func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
