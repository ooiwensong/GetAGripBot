package main

import (
	"context"
	"os"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func newBot() (*bot.Bot, error) {
	apiKey := os.Getenv("KEY_API")

	opts := []bot.Option{
		bot.WithSkipGetMe(),
		bot.WithDebug(),
		bot.WithDefaultHandler(defaultHandler),
		bot.WithMessageTextHandler("help", bot.MatchTypeCommand, helpHandler),
	}

	bot, err := bot.New(apiKey, opts...)
	if err != nil {
		return nil, err
	}

	return bot, nil
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil && strings.HasPrefix(update.Message.Text, "/") {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Unknown command. Type /help for more help.",
		})
	}
}

func helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	helpText := "Available commands:\n\n" +
		"/gyms - List all gyms and their pass validity\n" +
		"/who - List all available passes sorted by owner\n" +
		"/where - List all available passes sorted by gym\n" +
		"/mooch - Mooch off a pass from someone\n" +
		"/register - Register yourself with the bot\n"

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   helpText,
	})
}
