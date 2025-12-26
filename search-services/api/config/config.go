package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type ApiConfig struct {
	Address string        `yaml:"address" env:"API_ADDRESS" env-default:"localhost:80"`
	Timeout time.Duration `yaml:"timeout" env:"API_TIMEOUT" env-default:"5s"`
}

type AuthConfig struct {
	AdminUser     string        `yaml:"admin_user" env:"ADMIN_USER" env-default:"admin"`
	AdminPassword string        `yaml:"admin_password" env:"ADMIN_PASSWORD" env-default:"password"`
	JwtSecret     string        `yaml:"jwt_secret" env:"ADMIN_JWT_KEY" env-default:"your-secret-key"`
	TokenTtl      time.Duration `yaml:"token_ttl" env:"TOKEN_TTL" env-default:"2m"`
}

type Limits struct {
	SearchConcurrency int `yaml:"search_concurrency" env:"SEARCH_CONCURRENCY" env-default:"10"`
	SearchRate        int `yaml:"search_rate" env:"SEARCH_RATE" env-default:"100"`
}

type Config struct {
	LogLevel      string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	WordsAddress  string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"words:81"`
	UpdateAddress string `yaml:"update_address" env:"UPDATE_ADDRESS" env-default:"update:82"`
	SearchAddress string `yaml:"search_address" env:"SEARCH_ADDRESS" env-default:"search:83"`

	ApiConfig ApiConfig  `yaml:"api_server"`
	Auth      AuthConfig `yaml:"auth"`
	Limits    Limits     `yaml:"limits"`
}

func MustLoad(configPath string, cfg *Config) {
	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
}
