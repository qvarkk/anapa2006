package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/store"
	"time"
)

const DodgeSleepTimeMs = 1200

type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendMediaWithCaption(ctx context.Context, chatID int64, media []db.DraftMedium, caption string) error
}

type AfterPostCallbackFn func(ctx context.Context, schedule db.Schedule)

type Scheduler struct {
	store  *store.Store
	sender Sender

	onAfterPost AfterPostCallbackFn
}

func NewScheduler(
	store *store.Store,
	sender Sender,
	onAfterPost AfterPostCallbackFn,
) *Scheduler {
	return &Scheduler{
		store:       store,
		sender:      sender,
		onAfterPost: onAfterPost,
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

func (s *Scheduler) publish(ctx context.Context, sched db.Schedule) {
	claimed, err := s.store.ClaimDueSchedule(ctx, sched.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"claim schedule",
			slog.Int64("schedule_id", sched.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	draft, err := s.store.GetDraftByID(ctx, claimed.DraftID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"get schedule draft",
			slog.Int64("schedule_id", sched.ID),
			slog.Int64("draft_id", sched.DraftID),
			slog.String("error", err.Error()),
		)
		return
	}

	media, err := s.store.ListDraftMedia(ctx, draft.ID)
	if err != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"list draft media",
			slog.Int64("draft_id", draft.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	var sendErr error
	if len(media) == 0 {
		sendErr = s.sender.SendMessage(ctx, claimed.TargetChatID, draft.FinalText)
	} else {
		sendErr = s.sender.SendMediaWithCaption(ctx, claimed.TargetChatID, media, draft.FinalText)
	}

	if sendErr != nil {
		slog.LogAttrs(
			ctx, slog.LevelError,
			"send message error",
			slog.Int64("schedule_id", claimed.ID),
			slog.Int64("draft_id", draft.ID),
			slog.String("error", sendErr.Error()),
		)
		return
	}

	if err := s.store.MarkScheduleSent(ctx, claimed.ID); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"mark schedule sent",
			slog.Int64("schedule_id", claimed.ID),
			slog.String("error", err.Error()),
		)
	}

	if err := s.store.MarkPostSent(ctx, draft.PostID); err != nil {
		slog.LogAttrs(
			ctx, slog.LevelWarn,
			"mark post sent",
			slog.Int64("post_id", draft.PostID),
			slog.String("error", err.Error()),
		)
	}

	if s.onAfterPost != nil {
		s.onAfterPost(ctx, claimed)
	}

	slog.LogAttrs(
		ctx, slog.LevelInfo,
		"published scheduled post",
		slog.Int64("schedule_id", claimed.ID),
		slog.Int64("draft_id", draft.ID),
		slog.Int64("chat_id", sched.TargetChatID),
	)
}
