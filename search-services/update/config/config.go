package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type BrokerConfig struct {
	Address string `yaml:"address" env:"BROKER_ADDRESS" env-default:"nats://nats:4222"`
	Subject string `yaml:"topic" env:"BROKER_SUBJECT" env-default:"xkcd.db.updated"`
}

type XKCDConfig struct {
	URL         string        `yaml:"url" env:"XKCD_URL" env-default:"xkcd.com"`
	Concurrency int           `yaml:"concurrency" env:"XKCD_CONCURRENCY" env-default:"1"`
	Timeout     time.Duration `yaml:"timeout" env:"XKCD_TIMEOUT" env-default:"10s"`
	CheckPeriod time.Duration `yaml:"check_period" env:"XKCD_CHECK_PERIOD" env-default:"1h"`
}

type PromConfig struct {
	Address string        `yaml:"address" env:"PROMETHEUS_ADDRESS" env-default:":2115"`
	Timeout time.Duration `yaml:"timeout" env:"PROMETHEUS_TIMEOUT" env-default:"5s"`
}

type Config struct {
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`

	UpdateAddress string `yaml:"update_address" env:"UPDATE_ADDRESS" env-default:":80"`
	WordsAddress  string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:":81"`
	DBAddress     string `yaml:"db_address" env:"DB_ADDRESS" env-default:"postgres://postgres:password@postgres:5432/postgres?sslmode=disable"`

	XKCD   XKCDConfig   `yaml:"xkcd"`
	Broker BrokerConfig `yaml:"broker"`

	PromServer PromConfig `yaml:"prom_server"`
}

func MustLoad(configPath string, cfg *Config) {
	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
}
