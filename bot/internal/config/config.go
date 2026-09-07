package config

import (
	"net/url"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Debug bool `default:"true"`

	TelegramBotToken string `required:"true" envconfig:"TELEGRAM_BOT_TOKEN"`

	DBPath string `default:"/var/opt/anapa2006/anapa2006.db" envconfig:"DB_PATH"`

	RsshubBaseUrl *url.URL `default:"http://rsshub:1200" envconfig:"RSSHUB_BASE_URL"`

	FetchInterval time.Duration `default:"15m" envconfig:"FETCH_INTERVAL"`
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
