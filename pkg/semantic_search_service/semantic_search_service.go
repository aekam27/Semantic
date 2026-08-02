// Package semantic_search_service serves vector similarity queries.
package semantic_search_service

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/aekam27/Semantic/pkg/dao"
	util "github.com/aekam27/Semantic/pkg/utils"
)

// maxBodyBytes caps the search payload we are willing to read.
const maxBodyBytes = 1 << 16 // 64 KiB

// Handler serves the vector search endpoint.
type Handler struct {
	dao          dao.MongoDAO
	embeddingURL string
	client       *http.Client
}

// NewHandler builds the search handler.
func NewHandler(mongoDAO dao.MongoDAO, embeddingURL string, client *http.Client) *Handler {
	return &Handler{dao: mongoDAO, embeddingURL: embeddingURL, client: client}
}

// MongoVector is the request payload for a vector search.
type MongoVector struct {
	SearchIndex string `json:"searchIndex,omitempty"`
	Path        string `json:"path,omitempty"`
	Text        string `json:"text,omitempty"`
	Project     string `json:"project,omitempty"`
	Limit       int64  `json:"limit,omitempty"`
}

// MongoVectorSearch embeds the query text and returns the nearest documents.
func (h *Handler) MongoVectorSearch(w http.ResponseWriter, r *http.Request) {
	details, err := decodePayload(r)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(details.Text) == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	embeddingResp, err := util.PostJSON(ctx, h.client, h.embeddingURL, map[string]string{"name": details.Text})
	if err != nil {
		log.Printf("search: embedding service: %v", err)
		http.Error(w, "Server error", http.StatusBadGateway)
		return
	}

	var vectorData map[string]bson.A
	if err := json.Unmarshal(embeddingResp, &vectorData); err != nil {
		log.Printf("search: decode embedding response: %v", err)
		http.Error(w, "Server error", http.StatusBadGateway)
		return
	}

	result, err := h.dao.VectorFind(ctx, dao.VectorSearch{
		SearchIndex: details.SearchIndex,
		Path:        details.Path,
		Project:     details.Project,
		Limit:       details.Limit,
		QueryVector: vectorData["vector_embedding"],
	})
	if err != nil {
		log.Printf("search: vector find: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	jsonString, err := json.Marshal(bson.M{"data": result})
	if err != nil {
		log.Printf("search: marshal results: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	util.GZIPResp(w, jsonString)
}

func decodePayload(r *http.Request) (MongoVector, error) {
	var payload MongoVector
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return MongoVector{}, err
	}
	return payload, nil
}
