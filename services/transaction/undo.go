package transaction

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UndoService struct {
	db *mongo.Database
}

func NewUndoService(db *mongo.Database) *UndoService {
	return &UndoService{db: db}
}

func (s *UndoService) UndoHandlerFunc() bot.HandlerFunc {
	db := s.db
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		collection := db.Collection("transactions")
		var lastTransaction Transaction
		err := collection.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.M{"created_at": -1})).Decode(&lastTransaction)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Cannot undo liao",
				})
				return
			} else {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Something went wrong. Try again later.",
				})
			}
			return
		}

		switch lastTransaction.Action {
		case Buy:
			// Undo buy: delete the pass
			collection := db.Collection("passes")
			_, err = collection.DeleteOne(ctx, bson.M{"_id": lastTransaction.Payload})
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Failed to undo buy action. Try again later.",
				})
				return
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Buy action undone.",
			})
		case Use:
			// Undo use: increase the pass count back
			collection := db.Collection("passes")
			_, err = collection.UpdateOne(ctx, bson.M{"_id": lastTransaction.Payload}, bson.M{"$inc": bson.M{"qty": 1}})
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Failed to undo use action. Try again later.",
				})
				return
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Use pass action undone.",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Unknown action type. Cannot undo.",
			})
			return
		}

		lastTransaction.Delete(ctx, db)
	}
}
