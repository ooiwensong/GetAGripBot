package transaction

import (
	"context"
	"time"

	"github.com/ooiwensong/GetAGripBot/internal/passmodel"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Transaction struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Action    Action             `bson:"action,omitempty"`
	CreatedAt time.Time          `bson:"created_at,omitempty"`
	Payload   passmodel.Pass     `bson:"payload,omitempty"`
}

type Action string

const (
	Buy    Action = "buy"
	Use    Action = "use"
	Delete Action = "delete"
)

func NewTransaction(action Action, payload passmodel.Pass) Transaction {
	return Transaction{
		Action:    action,
		CreatedAt: time.Now(),
		Payload:   payload,
	}
}

func (t *Transaction) Save(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("transactions")
	_, err := collection.InsertOne(ctx, bson.M{
		"action":     t.Action,
		"created_at": t.CreatedAt,
		"payload":    t.Payload,
	})
	return err
}

func (t *Transaction) Delete(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("transactions")
	_, err := collection.DeleteOne(ctx, bson.M{"_id": t.ID})
	return err
}
