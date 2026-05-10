package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	// Keep-alive server to prevent Render deployment from sleeping
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	go func() {
		fmt.Printf("Starting keep-alive server on port %s...\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			fmt.Printf("Keep-alive server error: %v\n", err)
		}
	}()

	// Telegram bot
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
