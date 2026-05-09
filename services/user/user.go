package user

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type User struct {
	ID       int64  `bson:"_id"`
	Username string `bson:"username"`
}

type UserService struct {
	db *mongo.Database
}

func NewUserService(db *mongo.Database) *UserService {
	return &UserService{db: db}
}

func (s *UserService) RegisterUserHandlerFunc() bot.HandlerFunc {

	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		collection := db.Collection("users")
		userID := update.Message.From.ID

		var result User
		err := collection.FindOne(ctx, bson.D{{Key: "_id", Value: userID}}).Decode(&result)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				username := update.Message.From.Username
				collection.InsertOne(ctx, bson.D{
					{Key: "_id", Value: userID},
					{Key: "username", Value: username},
				})

				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "You have been registered successfully!",
				})
				return
			} else {
				log.Print(err.Error())
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Something went wrong, please try again later.",
				})
				return
			}

		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "You are already registered.",
		})
	}
}
