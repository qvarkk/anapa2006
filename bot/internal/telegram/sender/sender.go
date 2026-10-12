package sender

import (
	"context"
	"fmt"
	"qq/anapa2006/internal/db"
	"qq/anapa2006/internal/domain"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Sender struct {
	b *bot.Bot
}

func New(b *bot.Bot) *Sender {
	return &Sender{b: b}
}

func (s *Sender) SendMessage(ctx context.Context, chatID int64, text string) (*models.Message, error) {
	return s.b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}

func (s *Sender) SendMediaWithCaption(
	ctx context.Context, chatID int64,
	media []db.DraftMedium, caption string,
) (*models.Message, error) {
	if len(media) == 1 {
		medium := media[0]
		switch medium.Kind {
		case string(domain.MediaKindPhoto):
			return s.b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:    chatID,
				Photo:     &models.InputFileString{Data: mediaSource(medium)},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})
		case string(domain.MediaKindVideo):
			return s.b.SendVideo(ctx, &bot.SendVideoParams{
				ChatID:    chatID,
				Video:     &models.InputFileString{Data: mediaSource(medium)},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})
		case string(domain.MediaKindDocument):
			return s.b.SendDocument(ctx, &bot.SendDocumentParams{
				ChatID:    chatID,
				Document:  &models.InputFileString{Data: mediaSource(medium)},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})
		default:
			return nil, fmt.Errorf("unknown media kind in scheduled message")
		}
	}

	if len(media) < 2 || len(media) > 10 {
		return nil, fmt.Errorf("expected media count is between 1 and 10, got %d", len(media))
	}

	var tgMedia []models.InputMedia
	for i, medium := range media {
		var currentCaption string
		if i == 0 {
			currentCaption = caption
		}

		switch medium.Kind {
		case string(domain.MediaKindPhoto):
			tgMedia = append(tgMedia, &models.InputMediaPhoto{
				Media:     mediaSource(medium),
				Caption:   currentCaption,
				ParseMode: models.ParseModeHTML,
			})
		case string(domain.MediaKindVideo):
			tgMedia = append(tgMedia, &models.InputMediaVideo{
				Media:     mediaSource(medium),
				Caption:   currentCaption,
				ParseMode: models.ParseModeHTML,
			})
		case string(domain.MediaKindDocument):
			tgMedia = append(tgMedia, &models.InputMediaDocument{
				Media:     mediaSource(medium),
				Caption:   currentCaption,
				ParseMode: models.ParseModeHTML,
			})
		default:
			return nil, fmt.Errorf("unknown media kind in scheduled message")
		}
	}

	msgs, err := s.b.SendMediaGroup(ctx, &bot.SendMediaGroupParams{
		ChatID: chatID,
		Media:  tgMedia,
	})
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("media group returned no messages")
	}
	return msgs[0], nil
}

func mediaSource(m db.DraftMedium) string {
	if m.FileID.Valid && m.FileID.String != "" {
		return m.FileID.String
	}
	return m.Url.String
}
