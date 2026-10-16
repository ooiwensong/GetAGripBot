package passmodel

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
