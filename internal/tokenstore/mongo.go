package tokenstore

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoStore implements Store using MongoDB
type MongoStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewMongoStore creates a new MongoDB token store
func NewMongoStore(ctx context.Context, uri string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetRetryReads(true).SetRetryWrites(true))
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database("clipboard")
	collection := db.Collection("tokens")

	return &MongoStore{
		client:     client,
		collection: collection,
	}, nil
}

// Store saves a new token
func (m *MongoStore) Store(ctx context.Context, token Token) error {
	_, err := m.collection.InsertOne(ctx, token)
	return err
}

// Exists checks if a token exists
func (m *MongoStore) Exists(ctx context.Context, token string) (bool, error) {
	count, err := m.collection.CountDocuments(ctx, bson.M{"token": token})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserID returns the user ID for a token, or false if not found
func (m *MongoStore) GetUserID(ctx context.Context, token string) (string, bool, error) {
	var result struct {
		UserID string `bson:"user_id"`
	}
	err := m.collection.FindOne(ctx, bson.M{"token": token}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", false, nil
		}
		return "", false, err
	}
	return result.UserID, true, nil
}

// Delete removes a token
func (m *MongoStore) Delete(ctx context.Context, token string) error {
	_, err := m.collection.DeleteOne(ctx, bson.M{"token": token})
	return err
}

// Close disconnects from MongoDB
func (m *MongoStore) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
