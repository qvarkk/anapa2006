package store

import (
	"context"
	"qq/anapa2006/internal/db"
)

// DeletePendingReplyByPrompt forgets the prompt that a message is answering.
func (s *Store) DeletePendingReplyByPrompt(ctx context.Context, chatID, promptMessageID int64) error {
	return s.DeletePendingReply(ctx, db.DeletePendingReplyParams{
		ChatID: chatID, PromptMessageID: promptMessageID,
	})
}
