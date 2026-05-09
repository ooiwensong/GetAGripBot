package pass

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/ooiwensong/GetAGripBot/services/gym"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Pass struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	OwnerID       int64              `bson:"owner_id,omitempty"`
	Owner         string             `bson:"owner,omitempty"`
	GymID         primitive.ObjectID `bson:"gym_id,omitempty"`
	GymName       string             `bson:"gym_name,omitempty"`
	Cost          float64            `bson:"cost,omitempty"`
	Qty           int8               `bson:"qty,omitempty"`
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

func (s *PassService) UsePassHandlerFunc() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		collection := db.Collection("passes")
		pipeline := mongo.Pipeline{
			{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gym_id"},
			}}},

			{{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "gyms"},
				{Key: "localField", Value: "_id"},
				{Key: "foreignField", Value: "_id"},
				{Key: "pipeline", Value: mongo.Pipeline{
					{{Key: "$project", Value: bson.D{
						{Key: "_id", Value: 0},
						{Key: "name_short", Value: "$name_short"},
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
					}},
				}},
			}}},

			{{Key: "$unset", Value: bson.A{"gym_info"}}},

			{{Key: "$sort", Value: bson.D{{Key: "gym_name", Value: 1}}}},
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

		var results []gym.Gym
		if err := cursor.All(ctx, &results); err != nil {
			log.Print(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		var kbButtons [][]models.InlineKeyboardButton
		const numItemsPerRow = 3
		i := 0
		for i < len(results) {
			var row []models.InlineKeyboardButton
			for j := range numItemsPerRow {
				if i+j < len(results) {
					gym := results[i+j]
					row = append(row, models.InlineKeyboardButton{
						Text:         fmt.Sprintf("%s ($%0.2f)", gym.NameShort, gym.Cost),
						CallbackData: fmt.Sprintf("use_gym_%s_%s", gym.NameShort, gym.ID.Hex()),
					})
				} else {
					break
				}
			}
			i += numItemsPerRow
			kbButtons = append(kbButtons, row)
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Select one gym with available passes:",
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: kbButtons,
			},
		})
	}
}

func (s *PassService) BuyPassHandlerFunc() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var results []gym.Gym
		collection := db.Collection("gyms")
		cursor, err := collection.Find(ctx, bson.D{})
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

		var kbButtons [][]models.InlineKeyboardButton
		const numItemsPerRow = 3
		i := 0
		for i < len(results) {
			var row []models.InlineKeyboardButton
			for j := range numItemsPerRow {
				if i+j < len(results) {
					gym := results[i+j]
					row = append(row, models.InlineKeyboardButton{
						Text:         fmt.Sprintf("%s (%d passes)", gym.NameShort, gym.Typ),
						CallbackData: fmt.Sprintf("buy_pass_%s", gym.ID.Hex()),
					})
				} else {
					break
				}
			}
			i += numItemsPerRow
			kbButtons = append(kbButtons, row)
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Select a gym to buy passes for:",
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: kbButtons,
			},
		})
	}
}

func (s *PassService) UseGymCallbackHandlerFunc() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})

		selectedGym := update.CallbackQuery.Data
		// Expected format: "use_gym_{gymName}_{gymID}"
		parts := strings.Split(selectedGym, "_")
		if len(parts) != 4 {
			log.Printf("Invalid callback query data: %s", selectedGym)
			return
		}

		gymName := parts[2]
		gymID := parts[3]

		gymObjID, err := primitive.ObjectIDFromHex(gymID)
		if err != nil {
			log.Printf("Invalid gym ID: %s", gymID)
			return
		}

		collection := db.Collection("passes")
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.D{
				{Key: "gym_id", Value: gymObjID},
			}}},

			{{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "users"},
				{Key: "localField", Value: "owner_id"},
				{Key: "foreignField", Value: "_id"},
				{Key: "pipeline", Value: mongo.Pipeline{
					{{Key: "$project", Value: bson.D{
						{Key: "_id", Value: 0},
						{Key: "owner", Value: "$username"},
					}}}},
				},
				{Key: "as", Value: "owner_info"},
			}}},

			{{Key: "$replaceRoot", Value: bson.D{
				{Key: "newRoot", Value: bson.D{
					{Key: "$mergeObjects", Value: bson.A{
						"$$ROOT",
						bson.D{{Key: "$arrayElemAt", Value: bson.A{"$owner_info", 0}}},
					}},
				}},
			}}},

			{{Key: "$unset", Value: bson.A{"owner_info"}}},
		}

		cursor, err := collection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Print(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		var results []Pass
		if err := cursor.All(ctx, &results); err != nil {
			log.Print(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		var kbButtons [][]models.InlineKeyboardButton
		numItemsPerRow := 2
		i := 0
		for i < len(results) {
			var row []models.InlineKeyboardButton
			for j := range numItemsPerRow {
				if i+j < len(results) {
					pass := results[i+j]
					row = append(row, models.InlineKeyboardButton{
						Text:         fmt.Sprintf("%s (%d left)", pass.Owner, pass.Qty),
						CallbackData: fmt.Sprintf("use_pass_%s", pass.ID.Hex()),
					})
				} else {
					break
				}
			}
			i += numItemsPerRow
			kbButtons = append(kbButtons, row)
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   fmt.Sprintf("Whose %s passes do you want to mooch:", gymName),
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: kbButtons,
			},
		})
	}
}

func (s *PassService) UsePassCallbackHandlerFunc() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})
		selectedPass := update.CallbackQuery.Data
		passID, ok := strings.CutPrefix(selectedPass, "use_pass_")
		if !ok {
			log.Printf("Invalid callback query data: %s", selectedPass)
			return
		}

		passObjID, err := primitive.ObjectIDFromHex(passID)
		if err != nil {
			log.Printf("Invalid pass ID: %s", passID)
			return
		}

		collection := db.Collection("passes")
		result := collection.FindOneAndUpdate(
			ctx,
			bson.M{"_id": passObjID},
			bson.M{"$inc": bson.M{"qty": -1}},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		)
		if result.Err() != nil {
			log.Printf("Error: %v", result.Err())
			if result.Err() == mongo.ErrNoDocuments {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.CallbackQuery.Message.Message.Chat.ID,
					Text:   "This pass is no longer available.",
				})
				return
			} else {
				log.Printf("Error updating pass quantity: %v", result.Err())
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.CallbackQuery.Message.Message.Chat.ID,
					Text:   "Something went wrong, please try again later.",
				})
				return
			}
		}

		var updatedPass Pass
		if err := result.Decode(&updatedPass); err != nil {
			log.Printf("Error decoding updated pass: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		if updatedPass.Qty == 0 {
			collection.DeleteOne(ctx, bson.M{"_id": updatedPass.ID})
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   fmt.Sprintf("1 pass used. %d remaining.", updatedPass.Qty),
		})
	}
}

func (s *PassService) BuyPassCallbackHandlerFunc() bot.HandlerFunc {
	// db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})

		selectedGym := update.CallbackQuery.Data
		gymID, ok := strings.CutPrefix(selectedGym, "buy_pass_")
		if !ok {
			log.Printf("Invalid callback query data: %s", selectedGym)
			return
		}

		gymObjID, err := primitive.ObjectIDFromHex(gymID)
		if err != nil {
			log.Printf("Invalid gym ID: %s", gymID)
			return
		}

		var gym gym.Gym
		collection := s.db.Collection("gyms")
		err = collection.FindOne(ctx, bson.M{"_id": gymObjID}).Decode(&gym)
		if err != nil {
			log.Printf("Error finding gym: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		ownerID := update.CallbackQuery.From.ID
		today := today()
		newPass := Pass{
			OwnerID:       ownerID,
			GymID:         gymObjID,
			Qty:           gym.Typ,
			DatePurchased: today,
			DateExpiry:    today.AddDate(0, int(gym.PassValidity), 0),
		}
		passCollection := s.db.Collection("passes")
		_, err = passCollection.InsertOne(ctx, newPass)
		if err != nil {
			log.Printf("Error inserting pass: %v", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "Something went wrong, please try again later.",
			})
			return
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   "Done!",
		})
	}
}

func expiryStatus(expiryDate time.Time) string {
	today := today()
	threeMonthsFromNow := today.AddDate(0, 3, 0)
	if threeMonthsFromNow.After(expiryDate) {
		return "⚠️ "
	}
	if today.After(expiryDate) {
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

func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
