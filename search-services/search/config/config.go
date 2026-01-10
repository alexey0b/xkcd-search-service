package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Broker struct {
	Address string `yaml:"address" env:"BROKER_ADDRESS" env-default:"nats://nats:4222"`
	Subject string `yaml:"topic" env:"BROKER_SUBJECT" env-default:"xkcd.db.updated"`
}

type PromConfig struct {
	Address string        `yaml:"address" env:"PROMETHEUS_ADDRESS" env-default:":2114"`
	Timeout time.Duration `yaml:"timeout" env:"PROMETHEUS_TIMEOUT" env-default:"5s"`
}

type Config struct {
	LogLevel string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	IndexTTL time.Duration `yaml:"index_ttl" env:"INDEX_TTL" env-default:"20s"`

	SearchAddress string `yaml:"search_address" env:"SEARCH_ADDRESS" env-default:":83"`
	WordsAddress  string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:":81"`
	DBAddress     string `yaml:"db_address" env:"DB_ADDRESS" env-default:"postgres://postgres:password@postgres:5432/postgres"`

	Broker     Broker     `yaml:"broker"`
	PromServer PromConfig `yaml:"prom_server"`
}

func MustLoad(configPath string, cfg *Config) {
	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
}
