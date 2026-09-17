package fetcher

import (
	"context"
	"net/url"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/store"
	"testing"
	"time"
)

// TODO: clean AI slop

func TestTick_DedupsAcrossRuns(t *testing.T) {
	ctx := context.Background()

	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	src, err := st.CreateSource(ctx, db.CreateSourceParams{ChannelHandle: "testchannel", Active: 1})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	var notified []db.Post
	url, _ := url.Parse("http://fake.invalid")
	f := NewFetcher(st, url, nil, func(_ context.Context, p db.Post) {
		notified = append(notified, p)
	})

	f.pull = func(_ context.Context, _ string) (*RSSResponse, error) {
		return &RSSResponse{
			Items: []RSSItem{
				{GUID: "post-1", Description: "hello world", PublishedAt: RSSDateTime{time.Now()}},
			},
		}, nil
	}

	f.Tick(ctx)
	if len(notified) != 1 {
		t.Fatalf("expected 1 new post after first tick, got %d", len(notified))
	}
	if notified[0].SourceID != src.ID {
		t.Errorf("expected post linked to source %d, got %d", src.ID, notified[0].SourceID)
	}

	f.Tick(ctx)
	if len(notified) != 1 {
		t.Fatalf("expected dedup on second tick, total notifications = %d", len(notified))
	}
}

func TestTick_MultipleNewItemsInOneFeed(t *testing.T) {
	ctx := context.Background()
	st, _ := store.Open(":memory:")
	defer st.Close()
	st.CreateSource(ctx, db.CreateSourceParams{ChannelHandle: "testchannel", Active: 1})

	var notified []db.Post
	url, _ := url.Parse("http://fake.invalid")
	f := NewFetcher(st, url, nil, func(_ context.Context, p db.Post) {
		notified = append(notified, p)
	})
	f.pull = func(_ context.Context, _ string) (*RSSResponse, error) {
		return &RSSResponse{
			Items: []RSSItem{
				{GUID: "a", Description: "post a"},
				{GUID: "b", Description: "post b"},
			},
		}, nil
	}

	f.Tick(ctx)
	if len(notified) != 2 {
		t.Fatalf("expected 2 new posts, got %d", len(notified))
	}
}
