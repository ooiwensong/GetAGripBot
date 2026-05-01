package gym

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Gym struct {
	Name         string  `bson:"name"`
	NameShort    string  `bson:"name_short"`
	Typ          string  `bson:"type,omitempty"`
	PassValidity int32   `bson:"pass_validity"`
	Cost         float64 `bson:"cost"`
}

type GymService struct {
	db *mongo.Database
}

func NewGymService(db *mongo.Database) *GymService {
	return &GymService{db: db}
}

func (s *GymService) GetGymsHandler() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var results []Gym
		collection := db.Collection("gyms")
		cursor, err := collection.Find(ctx, bson.D{})
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		if err := cursor.All(ctx, &results); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		var response strings.Builder
		fmt.Fprint(&response, "<b>Gyms:</b>\n")
		for _, gym := range results {
			fmt.Fprintf(&response, "\n%s\n\n    Validity:%d months\n", gym.NameShort, gym.PassValidity)
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      response.String(),
			ParseMode: models.ParseModeHTML,
		})
	}

}
