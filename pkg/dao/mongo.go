// Package dao holds the data access layer.
package dao

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/aekam27/Semantic/pkg/connections"
)

// defaultNumCandidates is how many nearest neighbours Atlas considers before
// trimming to the requested limit. Atlas recommends oversampling by 10-20x.
const defaultNumCandidates = 420

// maxLimit caps how many documents a single search may return, so a client
// cannot ask for the whole collection.
const maxLimit int64 = 100

// VectorSearch is a query against a vector index.
type VectorSearch struct {
	SearchIndex   string
	Path          string
	Project       string
	Limit         int64
	NumCandidates int
	QueryVector   bson.A
}

// MongoDAO reads documents by vector similarity.
type MongoDAO interface {
	VectorFind(ctx context.Context, search VectorSearch) ([]bson.M, error)
}

type mongoDB struct {
	conn       *connections.Mongo
	collection string
}

// InitializeMongoDAO builds a MongoDAO backed by conn.
func InitializeMongoDAO(conn *connections.Mongo, collection string) MongoDAO {
	return &mongoDB{conn: conn, collection: collection}
}

func (rep *mongoDB) VectorFind(ctx context.Context, search VectorSearch) ([]bson.M, error) {
	if search.SearchIndex == "" || search.Path == "" {
		return nil, fmt.Errorf("searchIndex and path are required")
	}
	if len(search.QueryVector) == 0 {
		return nil, fmt.Errorf("query vector is empty")
	}

	limit := search.Limit
	if limit <= 0 || limit > maxLimit {
		limit = maxLimit
	}
	numCandidates := search.NumCandidates
	if numCandidates <= 0 {
		numCandidates = defaultNumCandidates
	}

	pipeline := bson.A{
		bson.D{{Key: "$vectorSearch", Value: bson.D{
			{Key: "queryVector", Value: search.QueryVector},
			{Key: "path", Value: search.Path},
			{Key: "numCandidates", Value: numCandidates},
			{Key: "limit", Value: limit},
			{Key: "index", Value: search.SearchIndex},
		}}},
	}
	if search.Project != "" {
		pipeline = append(pipeline, bson.D{{Key: "$project", Value: bson.D{{Key: search.Project, Value: 1}}}})
	}

	cur, err := rep.conn.VFind(ctx, pipeline, rep.collection)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer cur.Close(ctx)

	var results []bson.M
	if err := cur.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("decode vector search results: %w", err)
	}
	return results, nil
}
