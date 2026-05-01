package pass

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Pass struct {
	Owner         uint64    `bson:"owner,omitempty"`
	Gym           string    `bson:"gym,omitempty"`
	Cost          float64   `bson:"cost,omitempty"`
	Qty           uint8     `bson:"qty,omitempty"`
	DatePurchased time.Time `bson:"date_purchased,omitempty"`
	DateExpiry    time.Time `bson:"date_expiry,omitempty"`
}

type PassesByOwner struct {
	Owner  string `bson:"owner"`
	Passes []Pass `bson:"passes"`
}

type PassesByGym struct {
	Gym    string `bson:"gym"`
	Passes []Pass `bson:"passes"`
}

type PassService struct {
	db *mongo.Database
}

func NewPassService(db *mongo.Database) *PassService {
	return &PassService{db: db}
}

func (s *PassService) GetPassesByOwnerHandler() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var results []PassesByOwner
		collection := db.Collection("users")

		pipeline := mongo.Pipeline{
			{{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "passes"},
				{Key: "localField", Value: "username"},
				{Key: "foreignField", Value: "owner"},
				{Key: "as", Value: "passes"},
			}}},
			{{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "owner", Value: "$username"},
				{Key: "passes", Value: bson.D{
					{Key: "$sortArray", Value: bson.D{
						{Key: "input", Value: bson.D{
							{Key: "$map", Value: bson.D{
								{Key: "input", Value: "$passes"},
								{Key: "as", Value: "pass"},
								{Key: "in", Value: bson.D{
									{Key: "gym", Value: "$$pass.gym"},
									{Key: "cost", Value: "$$pass.cost"},
									{Key: "qty", Value: "$$pass.qty"},
									{Key: "date_expiry", Value: "$$pass.date_expiry"},
								}},
							}},
						}},
						{Key: "sortBy", Value: bson.D{
							{Key: "gym", Value: 1},
						}},
					}},
				}},
			}}},
		}
		cursor, err := collection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Print(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		if err := cursor.All(ctx, &results); err != nil {
			log.Print(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		var response strings.Builder
		fmt.Fprint(&response, "<b>Passes (sorted by owner):</b>\n")
		for _, doc := range results {
			fmt.Fprintf(&response, "\n%s:\n", doc.Owner)
			if len(doc.Passes) == 0 {
				fmt.Fprint(&response, "    <i>No passes. Buy more, bitch.</i>\n")
				continue
			}
			for _, pass := range doc.Passes {
				fmt.Fprintf(&response, "    - %s ($%.2f): %d left\n", pass.Gym, pass.Cost, pass.Qty)
			}
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      response.String(),
			ParseMode: models.ParseModeHTML,
		})

	}
}
