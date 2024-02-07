package conections

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoConn *mongo.Database

func init() {
	db := "mongodb+srv"
	dbHost := ""
	dbName := ""
	userName := ""
	password := ""
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	uri := db + "://" + userName + ":" + password + "@" + dbHost + "/?retryWrites=true&w=majority"
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		os.Exit(1)
	}
	if err = client.Ping(context.TODO(), nil); err != nil {
		os.Exit(1)
	}
	mongoConn = client.Database(dbName)
	log.Println("Mongo connected successfully!")
}

func VFind(pipeline bson.A, collectionName string, ctx context.Context) (*mongo.Cursor, error) {
	return mongoConn.Collection(collectionName).Aggregate(ctx, pipeline)
}
