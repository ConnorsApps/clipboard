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

// NewMongoStore creates a new MongoDB token store. tokenExpirySecs is the TTL
// for tokens in seconds; nil means tokens never expire.
func NewMongoStore(ctx context.Context, uri string, tokenExpirySecs *int32) (*MongoStore, error) {
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

	if err := ensureTokenIndex(ctx, collection, tokenExpirySecs); err != nil {
		return nil, err
	}

	return &MongoStore{
		client:     client,
		collection: collection,
	}, nil
}

// Ping checks that the MongoDB connection is alive.
func (m *MongoStore) Ping(ctx context.Context) error {
	return m.client.Ping(ctx, nil)
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
			return "", false, ErrNotFound
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

func ensureTokenIndex(ctx context.Context, collection *mongo.Collection, tokenExpirySecs *int32) error {
	specs, err := collection.Indexes().ListSpecifications(ctx)
	if err != nil {
		return err
	}
	existing := make(map[string]bool, len(specs))
	for _, spec := range specs {
		existing[spec.Name] = true
	}

	if !existing["token_1"] {
		if _, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		}); err != nil {
			return err
		}
	}
	if tokenExpirySecs != nil {
		if !existing["created_at_1"] {
			if _, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
				Keys:    bson.D{{Key: "created_at", Value: 1}},
				Options: options.Index().SetExpireAfterSeconds(*tokenExpirySecs),
			}); err != nil {
				return err
			}
		}
	} else if existing["created_at_1"] {
		if err = collection.Indexes().DropOne(ctx, "created_at_1"); err != nil {
			return err
		}
	}
	return nil
}
