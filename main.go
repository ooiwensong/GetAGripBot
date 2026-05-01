package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client, err := connectDB(ctx)
	if err != nil {
		panic(err)
	}

	bot, err := newBot()
	if err != nil {
		panic(err)
	}

	app := &App{
		dbClient: client,
		bot:      bot,
	}

	app.init()
	app.start(ctx)
	defer app.Stop(ctx)
}

// func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
// 	b.SendMessage(ctx, &bot.SendMessageParams{
// 		ChatID:    update.Message.Chat.ID,
// 		Text:      fmt.Sprintf("%s sent: %s", update.Message.From.Username, update.Message.Text),
// 		ParseMode: models.ParseModeHTML,
// 	})
// }
