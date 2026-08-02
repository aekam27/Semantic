package dao

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// These cases are all rejected before the database is touched, so a nil
// connection is fine.
func TestVectorFindRejectsIncompleteSearches(t *testing.T) {
	rep := &mongoDB{collection: "vectormap"}

	cases := []struct {
		name   string
		search VectorSearch
	}{
		{
			name:   "missing search index",
			search: VectorSearch{Path: "embedding", QueryVector: bson.A{0.1}},
		},
		{
			name:   "missing path",
			search: VectorSearch{SearchIndex: "idx", QueryVector: bson.A{0.1}},
		},
		{
			name:   "empty query vector",
			search: VectorSearch{SearchIndex: "idx", Path: "embedding"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := rep.VectorFind(context.Background(), tc.search); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
