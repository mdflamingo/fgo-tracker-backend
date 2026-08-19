package config

import (
	"flag"
	"os"
	"strings"
)

type Config struct {
	RunAddr     string
	LogLevel    string
	DataBaseDSN string
}

func ParseFlags() *Config {
	cfg := &Config{}

	RunAddr := flag.String("a", ":8080", "address and port to run server")
	logLevel := flag.String("l", "INFO", "log level")
	dataBaseDSN := flag.String("d", "", "connect to postgres")

	flag.Parse()

	cfg.RunAddr = getEnvOrDefault("RUN_ADDRESS", *RunAddr)
	cfg.LogLevel = strings.ToUpper(getEnvOrDefault("LOG_LEVEL", *logLevel))
	cfg.DataBaseDSN = getEnvOrDefault("DATABASE_DSN", *dataBaseDSN)

	return cfg
}

func getEnvOrDefault(envName, defaultValue string) string {
	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}
	return defaultValue
}
