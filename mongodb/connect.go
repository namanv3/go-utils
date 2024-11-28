package mongodb

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectToMongo(uri string) *mongo.Client {
	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to initiate client to connect to mongo")
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to ping mongo instance using initiated client")
	}
	return client
}
