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
