package main

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/ooiwensong/GetAGripBot/services/gym"
	"github.com/ooiwensong/GetAGripBot/services/pass"
	"github.com/ooiwensong/GetAGripBot/services/user"
	"go.mongodb.org/mongo-driver/mongo"
)

type App struct {
	dbClient *mongo.Client
	bot      *bot.Bot
}

func (a *App) init() {
	db := a.dbClient.Database("get_a_grip_bot")

	us := user.NewUserService(db)
	ps := pass.NewPassService(db)
	gs := gym.NewGymService(db)

	a.bot.RegisterHandler(bot.HandlerTypeMessageText, "gyms", bot.MatchTypeCommand, gs.GymsHandlerFunc())
	a.bot.RegisterHandler(bot.HandlerTypeMessageText, "who", bot.MatchTypeCommand, ps.PassesByOwnerHandlerFunc())
	a.bot.RegisterHandler(bot.HandlerTypeMessageText, "where", bot.MatchTypeCommand, ps.PassByGymsHandlerFunc())
	a.bot.RegisterHandler(bot.HandlerTypeMessageText, "register", bot.MatchTypeCommand, us.RegisterUserHandlerFunc())
}

func (a *App) start(ctx context.Context) {
	log.Print("Bot online.")
	a.bot.Start(ctx)
}

func (a *App) Stop(ctx context.Context) {
	log.Print("Bot shutting down...")
	if err := a.dbClient.Disconnect(ctx); err != nil {
		panic(err)
	}
}
