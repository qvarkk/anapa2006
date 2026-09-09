package fetcher

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/store"
	"strings"
	"time"
)

const HttpClientTimeout = 15 * time.Second

type NewPostCallbackFn func(ctx context.Context, post db.Post)

type Fetcher struct {
	store     *store.Store
	rsshubUrl *url.URL
	http      *http.Client
	onNewPost NewPostCallbackFn

	pull func(ctx context.Context, url string) (*RSSResponse, error)
}

func NewFetcher(store *store.Store, rsshubUrl *url.URL, onNewPost NewPostCallbackFn) *Fetcher {
	f := &Fetcher{
		store:     store,
		rsshubUrl: rsshubUrl,
		http:      &http.Client{Timeout: HttpClientTimeout},
		onNewPost: onNewPost,
	}
	f.pull = f.pullChannelFeed
	return f
}

func (f *Fetcher) Run(ctx context.Context, interval time.Duration) {
	f.Tick(ctx)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			f.Tick(ctx)
		}
	}
}

func (f *Fetcher) Tick(ctx context.Context) {
	sources, err := f.store.GetActiveSources(ctx)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get active sources",
			slog.String("error", err.Error()),
		)
		return
	}

	for _, src := range sources {
		err := f.fetchSource(ctx, src)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"failed to fetch channed feed",
				slog.String("channel", src.ChannelHandle),
				slog.String("error", err.Error()),
			)
			continue
		}
	}
}

func (f *Fetcher) fetchSource(ctx context.Context, src db.Source) error {
	url := f.getRsshubChannelFeedUrl(src.ChannelHandle)
	resp, err := f.pull(ctx, url)
	if err != nil {
		return fmt.Errorf("pull feed for %s: %w", src.ChannelHandle, err)
	}

	for _, item := range resp.Items {
		if isSystemMessage(item) {
			continue
		}

		caption, media, err := ParseDescription(item.Description)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"parse description",
				slog.String("channel", src.ChannelHandle),
				slog.String("guid", item.GUID),
				slog.String("error", err.Error()),
			)
		}
		caption = resolveCaption(item, caption, media)

		post, err := f.store.UpsertPost(ctx, db.UpsertPostParams{
			SourceID:   src.ID,
			ExternalID: item.GUID,
			RawText:    caption,
			PublishedAt: sql.NullTime{
				Time:  item.PublishedAt.Time,
				Valid: true,
			},
		})
		if errors.Is(err, sql.ErrNoRows) {
			// Post already exists
			continue
		}
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"upsert post",
				slog.String("channel", src.ChannelHandle),
				slog.String("guid", item.GUID),
				slog.String("error", err.Error()),
			)
			continue
		}

		for i, m := range media {
			if m.URL == "" {
				continue
			}
			if _, err := f.store.AddPostMedia(ctx, db.AddPostMediaParams{
				PostID:   post.ID,
				Kind:     string(m.MediaType),
				Url:      m.URL,
				Position: int64(i),
			}); err != nil {
				slog.LogAttrs(
					ctx, slog.LevelError,
					"add post media",
					slog.Int64("post_id", post.ID),
					slog.String("error", err.Error()),
				)
			}
		}

		slog.LogAttrs(
			ctx, slog.LevelInfo,
			"new post fetched",
			slog.String("channel", src.ChannelHandle),
			slog.Int64("post_id", post.ID),
		)
		if f.onNewPost != nil {
			f.onNewPost(ctx, post)
		}
	}
	return nil
}

func (f *Fetcher) pullChannelFeed(ctx context.Context, url string) (*RSSResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var feed RSSResponse
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func (f *Fetcher) getRsshubChannelFeedUrl(handle string) string {
	return f.rsshubUrl.JoinPath(RsshubTelegramPath, handle).String()
}

func resolveCaption(item RSSItem, parsed string, media []MediaItem) string {
	if parsed != "" {
		return parsed
	}
	for _, m := range media {
		if m.MediaType == MediaTypeDocument {
			return item.Title
		}
	}
	return ""
}

func isSystemMessage(item RSSItem) bool {
	return strings.HasPrefix(strings.TrimSpace(item.Title), "🔧")
}
