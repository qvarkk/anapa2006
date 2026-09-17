package telegram

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"qq/anapa2006/internal/domain"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const HttpClientTimeout = 15 * time.Second

type MediaCacher struct {
	bot         *bot.Bot
	http        *http.Client
	cacheChatID int64
}

func NewMediaCacher(b *bot.Bot, cacheChatID int64) *MediaCacher {
	return &MediaCacher{
		bot:         b,
		http:        &http.Client{Timeout: HttpClientTimeout},
		cacheChatID: cacheChatID,
	}
}

func (c *MediaCacher) CacheMedia(ctx context.Context, kind, url string) (string, error) {
	data, err := c.downloadWithReferer(ctx, url)
	if err != nil {
		return "", fmt.Errorf("cache %s: %w", kind, err)
	}

	switch kind {
	case string(domain.MediaKindPhoto):
		msg, err := c.bot.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID: c.cacheChatID,
			Photo:  &models.InputFileUpload{Filename: "media.jpg", Data: bytes.NewReader(data)},
		})
		if err != nil {
			return "", fmt.Errorf("cache photo: %w", err)
		}
		if len(msg.Photo) == 0 {
			return "", fmt.Errorf("cache photo: response had no photo sizes")
		}
		// msg has several resolutions. the last is the largest
		return msg.Photo[len(msg.Photo)-1].FileID, nil

	case "video":
		msg, err := c.bot.SendVideo(ctx, &bot.SendVideoParams{
			ChatID: c.cacheChatID,
			Video:  &models.InputFileUpload{Filename: "media.mp4", Data: bytes.NewReader(data)},
		})
		if err != nil {
			return "", fmt.Errorf("cache video: %w", err)
		}
		if msg.Video == nil {
			return "", fmt.Errorf("cache video: response had no video")
		}
		return msg.Video.FileID, nil

	default:
		return "", fmt.Errorf("cache media: unsupported kind %q", kind)
	}
}

func (c *MediaCacher) downloadWithReferer(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://t.me/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: unexpected status %d", resp.StatusCode)
	}

	// 50 mb upload limits for bots
	const maxCacheSize = 45 * 1024 * 1024
	limited := io.LimitReader(resp.Body, maxCacheSize+1)
	data, err := io.ReadAll(limited)
	if len(data) > maxCacheSize {
		return nil, fmt.Errorf("download: file exceeds %d bytes, refusing to cache", maxCacheSize)
	}

	return data, err
}
