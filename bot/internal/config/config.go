package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Debug bool `default:"true"`

	TelegramBotToken string `required:"true" envconfig:"TELEGRAM_BOT_TOKEN"`

	DBPath string `default:"/var/opt/anapa2006/anapa2006.db" envconfig:"DB_PATH"`
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
