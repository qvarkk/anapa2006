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
	"qq/anapa2006/internal/domain"
	"qq/anapa2006/internal/store"
	"strings"
	"time"
)

const (
	HttpClientTimeout = 60 * time.Second
	DodgeSleepTimeout = 1200 * time.Millisecond
	MaxRetryCount     = 5
)

type MediaCacher interface {
	CacheMedia(ctx context.Context, kind, url string) (fileID string, err error)
}

type NewPostCallbackFn func(ctx context.Context, post db.Post)

type Fetcher struct {
	store       *store.Store
	rsshubUrl   *url.URL
	http        *http.Client
	onNewPost   NewPostCallbackFn
	mediaCacher MediaCacher

	pull func(ctx context.Context, url string) (*RSSResponse, error)
}

func NewFetcher(
	store *store.Store,
	rsshubUrl *url.URL,
	mediaCacher MediaCacher,
	onNewPost NewPostCallbackFn,
) *Fetcher {
	f := &Fetcher{
		store:       store,
		rsshubUrl:   rsshubUrl,
		mediaCacher: mediaCacher,
		http:        &http.Client{Timeout: HttpClientTimeout},
		onNewPost:   onNewPost,
	}
	f.pull = f.pullChannelFeed
	return f
}

func (f *Fetcher) Run(ctx context.Context, interval time.Duration) {
	slog.LogAttrs(ctx, slog.LevelInfo, "started fetcher")
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
		}
	}

	retrieable, err := f.store.GetRetrieablePosts(ctx, MaxRetryCount)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get retrieable posts",
			slog.String("error", err.Error()),
		)
		return
	}

	for _, post := range retrieable {
		if err := f.retryCachingPostMedia(ctx, post.ID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"failed to retry cache post media",
				slog.Int64("post_id", post.ID),
				slog.String("error", err.Error()),
			)

			if err = f.store.IncrementRetryCount(ctx, post.ID); err != nil {
				slog.LogAttrs(
					ctx, slog.LevelWarn,
					"failed to increment retry count",
					slog.Int64("post_id", post.ID),
					slog.String("error", err.Error()),
				)
			}
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
				"failed to parse description",
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

		f.storeAndCacheMedia(ctx, post.ID, media)

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

func (f *Fetcher) retryCachingPostMedia(ctx context.Context, postID int64) error {
	media, err := f.store.ListUncachedMediaByPostID(ctx, postID)
	if err != nil {
		return fmt.Errorf("list uncached media: %w", err)
	}

	var errs []error
	for _, m := range media {
		if err := f.cacheMedia(
			ctx,
			MediaItem{
				URL:       m.Url,
				MediaType: domain.MediaKind(m.Kind),
			},
			postID, m.ID,
		); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
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

func (f *Fetcher) storeAndCacheMedia(ctx context.Context, postID int64, media []MediaItem) {
	for i, m := range media {
		if m.URL == "" {
			continue
		}

		row, err := f.store.AddPostMedia(ctx, db.AddPostMediaParams{
			PostID: postID, Kind: string(m.MediaType), Url: m.URL, Position: int64(i),
		})
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelError,
				"add post media",
				slog.Int64("post_id", postID),
				slog.String("error", err.Error()),
			)
			continue
		}

		if f.mediaCacher == nil {
			continue
		}

		if i > 0 {
			time.Sleep(DodgeSleepTimeout)
		}

		err = f.cacheMedia(ctx, m, postID, row.ID)
		if err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"cache media failed",
				slog.Int64("post_id", postID),
				slog.Int64("media_id", row.ID),
				slog.String("kind", string(m.MediaType)),
				slog.String("error", err.Error()),
			)
			continue
		}
	}
}

func (f *Fetcher) cacheMedia(ctx context.Context, m MediaItem, postID, mediaID int64) error {
	fileID, err := f.mediaCacher.CacheMedia(ctx, string(m.MediaType), m.URL)
	if err != nil {
		if dbErr := f.store.MarkPostFailed(ctx, postID); dbErr != nil {
			return fmt.Errorf("post not marked as failed and is uncached. scheduling it will cause problems: %w: %w", dbErr, err)
		}
		return fmt.Errorf("post caching failed, it is marked as failed and will retry later if possible: %w", err)
	}

	if err := f.store.SetPostMediaFileID(ctx, db.SetPostMediaFileIDParams{
		ID: mediaID, FileID: sql.NullString{String: fileID, Valid: true},
	}); err != nil {
		return fmt.Errorf("set post media file_id: %w", err)
	}

	return nil
}

func resolveCaption(item RSSItem, parsed string, media []MediaItem) string {
	if parsed != "" {
		return parsed
	}
	for _, m := range media {
		if m.MediaType == domain.MediaKindDocument {
			return item.Title
		}
	}
	return ""
}

func isSystemMessage(item RSSItem) bool {
	return strings.HasPrefix(strings.TrimSpace(item.Title), "🔧")
}
