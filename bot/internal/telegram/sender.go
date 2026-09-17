package telegram

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

func NewSender(b *bot.Bot) *Sender {
	return &Sender{b: b}
}

func (s *Sender) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := s.b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	return err
}

func (s *Sender) SendMediaWithCaption(
	ctx context.Context, chatID int64,
	media []db.DraftMedium, caption string,
) error {
	if len(media) == 1 {
		var err error
		medium := media[0]
		switch medium.Kind {
		case string(domain.MediaKindPhoto):
			_, err = s.b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:    chatID,
				Photo:     &models.InputFileString{Data: mediaSource(medium)},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})
		case string(domain.MediaKindVideo):
			_, err = s.b.SendVideo(ctx, &bot.SendVideoParams{
				ChatID:    chatID,
				Video:     &models.InputFileString{Data: mediaSource(medium)},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})
		case string(domain.MediaKindDocument):
			_, err = s.b.SendDocument(ctx, &bot.SendDocumentParams{
				ChatID:    chatID,
				Document:  &models.InputFileString{Data: mediaSource(medium)},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})
		default:
			return fmt.Errorf("unknown media kind in scheduled message")
		}
		return fmt.Errorf("media url %s: %w", medium.Url.String, err)
	} else if len(media) >= 2 && len(media) <= 10 {
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
				return fmt.Errorf("unknown media kind in scheduled message")
			}
		}

		_, err := s.b.SendMediaGroup(ctx, &bot.SendMediaGroupParams{
			ChatID: chatID,
			Media:  tgMedia,
		})
		return err
	} else {
		return fmt.Errorf("expected media count is between 1 and 10, got %d", len(media))
	}
}

func mediaSource(m db.DraftMedium) string {
	if m.FileID.Valid && m.FileID.String != "" {
		return m.FileID.String
	}
	return m.Url.String
}
