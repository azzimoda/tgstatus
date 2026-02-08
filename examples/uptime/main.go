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
	chatID, err := strconv.ParseInt(os.Getenv("CHANNEL_CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse CHANNEL_CHAT_ID")
	}

	b, err := bot.New(token)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create bot")
	}

	startTime := time.Now()

	statusManager := tgstatus.NewStatusManager(b, tgstatus.Config{
		ChatID:         chatID,
		StatusFunc:     func() tgstatus.StatusParams { return statusFunc(startTime) },
		StopStatusFunc: func() tgstatus.StatusParams { return stopStatusFunc(startTime) },
		SaveFile:       "status_state.json",
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var wg sync.WaitGroup
	wg.Go(func() { statusManager.RunUpdater(ctx, tgstatus.MinimumUpdatePeriod) })

	log.Info().Msg("Starting bot...")
	b.Start(ctx)
	log.Info().Msg("Bot stopped")

	wg.Wait()
	log.Info().Msg("Application stopped")
}

func statusFunc(start time.Time) tgstatus.StatusParams {
	return tgstatus.StatusParams{
		Text:                fmt.Sprintf("Server is up for %s", time.Since(start)),
		DisableNotification: true,
	}
}

func stopStatusFunc(start time.Time) tgstatus.StatusParams {
	return tgstatus.StatusParams{
		Text:                fmt.Sprintf("Server stopped after %s", time.Since(start)),
		DisableNotification: true,
	}
}
