package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address  string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"80"`
}

func MustLoad(configPath string, cfg *Config) {
	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
}
