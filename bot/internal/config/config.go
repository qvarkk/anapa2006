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

	// Optional: seeded as the first target channel when target_channels is
	// empty. Target channels are otherwise managed from the bot UI.
	TelegramChannelChatID      int64 `envconfig:"TELEGRAM_CHANNEL_CHAT_ID"`
	TelegramCacheChannelChatID int64 `required:"true" envconfig:"TELEGRAM_CACHE_CHANNEL_CHAT_ID"`

	// Owner of the bot. Has full access and is the only one who may manage
	// users; it is seeded into allowed_users by the /bootstrap command.
	// 0 means "not configured".
	TelegramOwnerID int64 `envconfig:"TELEGRAM_OWNER_ID"`

	DBPath string `default:"/var/opt/anapa2006/anapa2006.db" envconfig:"DB_PATH"`

	RsshubBaseUrl *url.URL `default:"http://rsshub:1200" envconfig:"RSSHUB_BASE_URL"`

	FetchInterval time.Duration `default:"15m" envconfig:"FETCH_INTERVAL"`

	PostInterval time.Duration `default:"1m" envconfig:"POST_INTERVAL"`
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
