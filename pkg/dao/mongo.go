package dao

import (
	"context"
	conections "goverse/pkg/connections"

	"go.mongodb.org/mongo-driver/bson"
)

type MongoDAO interface {
	VectorFind(searchIndex, path, project string, limit int64, searchQuery bson.A, ctx context.Context) ([]bson.M, error)
}

type mongoDB struct {
	collection string
}

func InitializeMongoDAO(collection string) MongoDAO {
	return &mongoDB{
		collection: collection,
	}
}

func (rep *mongoDB) VectorFind(searchIndex, path, project string, limit int64, searchQuery bson.A, ctx context.Context) ([]bson.M, error) {
	pipeline := bson.A{
		bson.D{
			{Key: "$vectorSearch",
				Value: bson.D{
					{
						Key:   "queryVector",
						Value: searchQuery,
					},
					{Key: "path", Value: path},
					{Key: "numCandidates", Value: 420},
					{Key: "limit", Value: limit},
					{Key: "index", Value: searchIndex},
				},
			},
		},
		bson.D{{Key: "$project", Value: bson.D{{Key: project, Value: 1}}}},
	}
	cur, err := conections.VFind(pipeline, rep.collection, ctx)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var results []bson.M
	if err = cur.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
