package tgstatus

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
)

// The maximum request rate for channels and groups is 20 requests/minute. So the minimum update period is 3 seconds,
// but this constant is set to 4 seconds to avoid occasional rate limiting.
const MinimumUpdatePeriod = (60/20 + 1) * time.Second

type Config struct {
	ChatID         int64
	StatusFunc     StatusFunc
	StopStatusFunc StatusFunc
	SaveFile       string
	// TODO: Not implemented yet.
	// 
	// How much time to wait before deleting and resending the status message again.
	// Useful when you want keep the message near to end of chat.
	//
	// When equal to zero, the message will not be deleted until it cannot be edited (48 hours).
	DeleteResendTimeout time.Duration
}

type StatusFunc = func() StatusParams

type StatusParams struct {
	Text                    string                          `json:"text"`
	ParseMode               models.ParseMode                `json:"parse_mode,omitempty"`
	Entities                []models.MessageEntity          `json:"entities,omitempty"`
	LinkPreviewOptions      *models.LinkPreviewOptions      `json:"link_preview_options,omitempty"`
	DisableNotification     bool                            `json:"disable_notification,omitempty"`
	ProtectContent          bool                            `json:"protect_content,omitempty"`
	AllowPaidBroadcast      bool                            `json:"allow_paid_broadcast,omitempty"`
	MessageEffectID         string                          `json:"message_effect_id,omitempty"`
	SuggestedPostParameters *models.SuggestedPostParameters `json:"suggested_post_parameters,omitempty"`
	ReplyParameters         *models.ReplyParameters         `json:"reply_parameters,omitempty"`
	ReplyMarkup             models.ReplyMarkup              `json:"reply_markup,omitempty"`
}

func (s StatusParams) ToSendMessageParams() bot.SendMessageParams {
	return bot.SendMessageParams{
		Text:                    s.Text,
		ParseMode:               s.ParseMode,
		Entities:                s.Entities,
		LinkPreviewOptions:      s.LinkPreviewOptions,
		DisableNotification:     s.DisableNotification,
		ProtectContent:          s.ProtectContent,
		AllowPaidBroadcast:      s.AllowPaidBroadcast,
		MessageEffectID:         s.MessageEffectID,
		SuggestedPostParameters: s.SuggestedPostParameters,
		ReplyParameters:         s.ReplyParameters,
		ReplyMarkup:             s.ReplyMarkup,
	}
}

func (s StatusParams) ToEditMessageTextParams() bot.EditMessageTextParams {
	return bot.EditMessageTextParams{
		Text:               s.Text,
		ParseMode:          s.ParseMode,
		Entities:           s.Entities,
		LinkPreviewOptions: s.LinkPreviewOptions,
		ReplyMarkup:        s.ReplyMarkup,
	}
}

func NewStatusManager(b *bot.Bot, config Config) StatusManager {
	sm := StatusManager{Config: config, bot: b}
	if config.SaveFile != "" {
		data, err := os.ReadFile(config.SaveFile)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read save file")
		}

		err = json.Unmarshal(data, &sm.messageID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal save file")
		}
	}
	return sm
}

type StatusManager struct {
	Config
	bot       *bot.Bot
	messageID int
}

func (s *StatusManager) setStatus(params StatusParams) error {
	if s.messageID == 0 {
		log.Warn().Msg("Message ID is not set; sending new message...")
		sendMessageParams := params.ToSendMessageParams()
		sendMessageParams.ChatID = s.ChatID
		msg, err := s.bot.SendMessage(context.Background(), &sendMessageParams)
		if err != nil {
			return err
		}
		s.messageID = msg.ID // TODO: Save messageID somewhere.
		return nil
	} else {
		log.Debug().Int64("chatID", s.ChatID).Msg("Editing status message...")

		editMessageTextParams := params.ToEditMessageTextParams()
		editMessageTextParams.ChatID = s.ChatID
		editMessageTextParams.MessageID = s.messageID
		_, err := s.bot.EditMessageText(context.Background(), &editMessageTextParams)

		if err != nil {
			log.Warn().Err(err).Msg("Failed to edit status message; trying to delete it and send new one...")
			if isDeleted, err := s.bot.DeleteMessage(context.Background(), &bot.DeleteMessageParams{
				ChatID:    s.ChatID,
				MessageID: s.messageID,
			}); err != nil {
				log.Error().Err(err).Msg("Failed to delete status message")
			} else if !isDeleted {
				log.Warn().Msg("The status message is not deleted")
			}
			s.messageID = 0
			return s.setStatus(params)
		}

		return err
	}
}

func (s *StatusManager) RunUpdater(ctx context.Context, period time.Duration) {
	log.Debug().Msg("Updater started")
	s.UpdateStatus(s.StatusFunc)
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Stopping updater...")
			s.UpdateStatus(s.StopStatusFunc)
			if err := s.saveFile(); err != nil {
				log.Error().Err(err).Msg("Failed to save message ID")
			} else {
				log.Info().Msg("Updater stopped")
			}
			return
		case <-time.After(period):
			s.UpdateStatus(s.StatusFunc)
		}
	}
}

func (s *StatusManager) UpdateStatus(statusFunc StatusFunc) {
	log.Debug().Msg("Updating status...")
	params := statusFunc()
	err := s.setStatus(params)
	if err != nil {
		log.Warn().Err(err).Any("params", params).Msg("Failed to update status")
	} else {
		log.Info().Msg("Status updated")
	}
}

func (s *StatusManager) saveFile() error {
	if s.messageID == 0 || s.SaveFile == "" {
		return nil
	}

	log.Debug().Int("messageID", s.messageID).Msg("Saving message ID...")
	data, err := json.Marshal(s.messageID)
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.SaveFile, data, 0644); err != nil {
		return err
	}
	return nil
}
