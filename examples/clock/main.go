package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/azzimoda/tgstatus"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal().Err(err).Msg("failed to load .env file")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatIDStr := os.Getenv("CHANNEL_CHAT_ID")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Fatal().Err(err).Str("CHANNEL_CHAT_ID", chatIDStr).Msg("failed to parse CHANNEL_CHAT_ID")
	}

	b, err := bot.New(token)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create bot")
	}

	statusManager := tgstatus.NewStatusManager(b, tgstatus.Config{
		ChatID:         chatID,
		SaveFile:       "status_state.json",
		StatusFunc:     statusFunc,
		StopStatusFunc: stopStatusFunc,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var wg sync.WaitGroup
	wg.Go(func() { statusManager.RunUpdater(ctx, 10*time.Second) })

	log.Info().Msg("Starting bot...")
	b.Start(ctx)
	log.Info().Msg("Bot stopped")

	wg.Wait()
	log.Info().Msg("Application stopped")
}

func statusFunc() tgstatus.StatusParams {
	return tgstatus.StatusParams{
		Text:                fmt.Sprintf("Curren time at the place where the server is existing: %s", time.Now()),
		DisableNotification: true,
	}
}

func stopStatusFunc() tgstatus.StatusParams {
	return tgstatus.StatusParams{
		Text:                fmt.Sprintf("Server stopped at %s", time.Now()),
		DisableNotification: true,
	}
}
