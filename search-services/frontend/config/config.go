package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type WebConfig struct {
	Address string        `yaml:"address" env:"FRONTEND_ADDRESS" env-default:"localhost:3000"`
	Timeout time.Duration `yaml:"timeout" env:"FRONTEND_TIMEOUT" env-default:"10s"`
}

type ApiConfig struct {
	ApiAddress string        `yaml:"address" env:"API_ADDRESS" env-default:"http://api:8080"`
	Timeout    time.Duration `yaml:"timeout" env:"API_TIMEOUT" env-default:"3m"`
}

type AuthConfig struct {
	AdminUser     string        `yaml:"admin_user" env:"ADMIN_USER" env-default:"admin"`
	AdminPassword string        `yaml:"admin_password" env:"ADMIN_PASSWORD" env-default:"password"`
	JwtSecret     string        `yaml:"jwt_secret" env:"ADMIN_JWT_KEY" env-default:"your-secret-key"`
	TokenTtl      time.Duration `yaml:"token_ttl" env:"TOKEN_TTL" env-default:"2m"`
}

type Config struct {
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`

	Web  WebConfig  `yaml:"web_server"`
	Api  ApiConfig  `yaml:"api"`
	Auth AuthConfig `yaml:"auth"`
}

func MustLoad(configPath string, cfg *Config) {
	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
}
