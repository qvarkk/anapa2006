package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/store"
	"time"

	"github.com/go-telegram/bot/models"
)

const (
	DodgeSleepTimeMs = 1200
	// A post claimed for sending that is still not resolved this long after its
	// scheduled time is considered a failed send (crash, restart, silent error).
	StaleSendGrace = 5 * time.Minute
)

type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string) (*models.Message, error)
	SendMediaWithCaption(ctx context.Context, chatID int64, media []db.DraftMedium, caption string) (*models.Message, error)
}

type Scheduler struct {
	store  *store.Store
	sender Sender
}

func NewScheduler(store *store.Store, sender Sender) *Scheduler {
	return &Scheduler{
		store:  store,
		sender: sender,
	}
}

func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	slog.LogAttrs(ctx, slog.LevelInfo, "started scheduler")
	s.Tick(ctx)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Tick(ctx)
		}
	}
}

func (s *Scheduler) Tick(ctx context.Context) {
	if err := s.store.MarkStaleSending(ctx, time.Now().Add(-StaleSendGrace)); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"mark stale scheduled posts as send errors",
			slog.String("error", err.Error()),
		)
	}

	due, err := s.store.ListDuePending(ctx, time.Now())
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"list due pending drafts",
			slog.String("error", err.Error()),
		)
		return
	}

	for i, sched := range due {
		if i > 0 {
			// rate limiting dodge
			time.Sleep(DodgeSleepTimeMs * time.Millisecond)
		}
		s.publish(ctx, sched)
	}
}

func (s *Scheduler) publish(ctx context.Context, sched db.ListDuePendingRow) {
	claimed, err := s.store.ClaimScheduledPost(ctx, sched.PostID)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"claim schedule",
			slog.Int64("post_id", sched.PostID),
			slog.String("error", err.Error()),
		)
		return
	}

	draft, err := s.store.GetPostDraftByID(ctx, claimed.ID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get post draft",
			slog.Int64("post_id", sched.PostID),
			slog.String("error", err.Error()),
		)
		return
	}

	media, err := s.store.ListDraftMedia(ctx, draft.PostID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"list draft media",
			slog.Int64("post_id", draft.PostID),
			slog.String("error", err.Error()),
		)
		return
	}

	var sent *models.Message
	var sendErr error
	if len(media) == 0 {
		sent, sendErr = s.sender.SendMessage(ctx, sched.ChatID, draft.FinalText)
	} else {
		sent, sendErr = s.sender.SendMediaWithCaption(ctx, sched.ChatID, media, draft.FinalText)
	}

	if sendErr != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"send message error",
			slog.Int64("post_id", draft.PostID),
			slog.String("error", sendErr.Error()),
		)

		if err := s.store.MarkPostSendError(ctx, claimed.ID); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"mark post send error",
				slog.Int64("post_id", claimed.ID),
				slog.String("error", err.Error()),
			)
		}
		if err := s.store.MarkScheduleSendError(ctx, db.MarkScheduleSendErrorParams{
			PostID:    claimed.ID,
			LastError: sql.NullString{String: sendErr.Error(), Valid: true},
		}); err != nil {
			slog.LogAttrs(
				ctx, slog.LevelWarn,
				"mark schedule send error",
				slog.Int64("post_id", claimed.ID),
				slog.String("error", err.Error()),
			)
		}
		return
	}

	var messageID sql.NullInt64
	if sent != nil {
		messageID = sql.NullInt64{Int64: int64(sent.ID), Valid: true}
	}
	if err := s.store.MarkScheduleSent(ctx, db.MarkScheduleSentParams{
		PostID: claimed.ID, SentMessageID: messageID,
	}); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"mark schedule sent",
			slog.Int64("post_id", claimed.ID),
			slog.String("error", err.Error()),
		)
	}

	if err := s.store.MarkPostSent(ctx, claimed.ID); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"mark post sent",
			slog.Int64("post_id", claimed.ID),
			slog.String("error", err.Error()),
		)
	}

	slog.LogAttrs(
		ctx, slog.LevelInfo,
		"published scheduled post",
		slog.Int64("post_id", draft.PostID),
		slog.Int64("chat_id", sched.ChatID),
	)
}
