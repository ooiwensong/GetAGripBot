package transaction

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Transaction struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Action    Action             `bson:"action,omitempty"`
	CreatedAt time.Time          `bson:"created_at,omitempty"`
	Payload   primitive.ObjectID `bson:"payload,omitempty"` // Payload can be the PassID for both buy and use actions
}

type Action string

const (
	Buy Action = "buy"
	Use Action = "use"
)

func NewTransaction(action Action, payload primitive.ObjectID) Transaction {
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
