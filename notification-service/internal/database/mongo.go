package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	connectClient = func(opts *options.ClientOptions) (*mongo.Client, error) {
		return mongo.Connect(opts)
	}
	pingClient = func(ctx context.Context, client *mongo.Client) error {
		return client.Ping(ctx, nil)
	}
)

// ConnectMongo ouvre la connexion et vérifie qu'elle répond (ping).
func ConnectMongo(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := connectClient(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := pingClient(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}
