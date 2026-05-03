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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Pass struct {
	OwnerID       uint64             `bson:"owner_id,omitempty"`
	Owner         string             `bson:"owner,omitempty"`
	GymID         primitive.ObjectID `bson:"gym_id,omitempty"`
	GymName       string             `bson:"gym_name,omitempty"`
	Cost          float64            `bson:"cost,omitempty"`
	Qty           uint8              `bson:"qty,omitempty"`
	DatePurchased time.Time          `bson:"date_purchased,omitempty"`
	DateExpiry    time.Time          `bson:"date_expiry,omitempty"`
}

type PassesByOwner struct {
	Owner  string `bson:"owner"`
	Passes []Pass `bson:"passes"`
}

type PassesByGym struct {
	Gym    string  `bson:"gym"`
	Cost   float64 `bson:"cost"`
	Passes []Pass  `bson:"passes"`
}

type PassService struct {
	db *mongo.Database
}

func NewPassService(db *mongo.Database) *PassService {
	return &PassService{db: db}
}

func (s *PassService) PassesByOwnerHandlerFunc() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		collection := db.Collection("passes")

		lookupPipeline := getLookupPipeline()

		pipeline := append(lookupPipeline, mongo.Pipeline{
			{{Key: "$sort", Value: bson.D{
				{Key: "username", Value: 1},
				{Key: "gym_name", Value: 1},
			}}},

			{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$username"},
				{Key: "passes", Value: bson.D{
					{Key: "$push", Value: bson.D{
						{Key: "gym_name", Value: "$gym_name"},
						{Key: "cost", Value: "$cost"},
						{Key: "qty", Value: "$qty"},
						{Key: "date_expiry", Value: "$date_expiry"},
					}},
				}},
			}}},

			{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},

			{{Key: "$project", Value: bson.D{
				{Key: "owner", Value: "$_id"},
				{Key: "passes", Value: 1},
			}}},
		}...,
		)

		cursor, err := collection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Print(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		var results []PassesByOwner
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
			for _, pass := range doc.Passes {
				emoji := expiryStatus(pass.DateExpiry)
				fmt.Fprintf(&response, "    - %s ($%.2f): %d left %s\n", pass.GymName, pass.Cost, pass.Qty, emoji)
			}
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      response.String(),
			ParseMode: models.ParseModeHTML,
		})

	}
}

func (s *PassService) PassByGymsHandlerFunc() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		collection := db.Collection("passes")

		lookupStage := getLookupPipeline()

		pipeline := append(lookupStage, mongo.Pipeline{
			{{Key: "$sort", Value: bson.D{
				{Key: "gym_name", Value: 1},
				{Key: "username", Value: 1},
			}}},

			{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gym_id"},
				{Key: "gym", Value: bson.D{{Key: "$first", Value: "$gym_name"}}},
				{Key: "cost", Value: bson.D{{Key: "$first", Value: "$cost"}}},
				{Key: "passes", Value: bson.D{
					{Key: "$push", Value: bson.D{
						{Key: "owner", Value: "$username"},
						{Key: "qty", Value: "$qty"},
						{Key: "date_expiry", Value: "$date_expiry"},
					}},
				}},
			}}},

			{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},

			{{Key: "$project", Value: bson.D{
				{Key: "gym", Value: "$gym"},
				{Key: "cost", Value: 1},
				{Key: "passes", Value: 1},
			}}},
		}...)

		cursor, err := collection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Print(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}
		var results []PassesByGym
		if err := cursor.All(ctx, &results); err != nil {
			log.Print(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}
		fmt.Printf("%v", results)

		var response strings.Builder
		fmt.Fprint(&response, "<b>Passes (sorted by gym):</b>\n")
		for _, doc := range results {
			fmt.Fprintf(&response, "\n%s ($%.2f):\n", doc.Gym, doc.Cost)
			for _, pass := range doc.Passes {
				emoji := expiryStatus(pass.DateExpiry)
				fmt.Fprintf(&response, "    - %s (%d left) %s\n", pass.Owner, pass.Qty, emoji)
			}
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      response.String(),
			ParseMode: models.ParseModeHTML,
		})
	}
}

func expiryStatus(expiryDate time.Time) string {
	threeMonthsFromNow := time.Now().AddDate(0, 3, 0)
	if threeMonthsFromNow.After(expiryDate) {
		return "⚠️ "
	}
	if time.Now().After(expiryDate) {
		return "☠️"
	}
	return ""
}

func getLookupPipeline() mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "owner_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "pipeline", Value: mongo.Pipeline{
				{{Key: "$project", Value: bson.D{
					{Key: "_id", Value: 0},
					{Key: "username", Value: "$username"},
				}}}},
			},
			{Key: "as", Value: "owner_info"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "gyms"},
			{Key: "localField", Value: "gym_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "pipeline", Value: mongo.Pipeline{
				{{Key: "$project", Value: bson.D{
					{Key: "_id", Value: 0},
					{Key: "gym_name", Value: "$name_short"},
					{Key: "cost", Value: "$cost"},
				}}}},
			},
			{Key: "as", Value: "gym_info"},
		}}},
		{{Key: "$replaceRoot", Value: bson.D{
			{Key: "newRoot", Value: bson.D{
				{Key: "$mergeObjects", Value: bson.A{
					"$$ROOT",
					bson.D{{Key: "$arrayElemAt", Value: bson.A{"$gym_info", 0}}},
					bson.D{{Key: "$arrayElemAt", Value: bson.A{"$owner_info", 0}}},
				}},
			}},
		}}},

		{{Key: "$unset", Value: bson.A{"gym_info", "owner_info"}}},
	}
}
