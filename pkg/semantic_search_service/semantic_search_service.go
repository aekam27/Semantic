package semantic_search_service

import (
	"context"
	"encoding/json"
	"goverse/pkg/dao"
	util "goverse/pkg/utils"
	"io"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
)

func MongoVectorSearch(w http.ResponseWriter, r *http.Request) {
	details, err := UnmarshalMVPayload(r)
	if err != nil {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(bson.M{"rep": []string{}})
		return
	}

	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	mongoDAO := dao.InitializeMongoDAO("vectormap")
	ctx := context.TODO()
	embeddingResp, err := util.PostApi("http://localhost:8000/v1/vectorEmbeddings/", map[string]string{"name": details.Text})
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	var vectorData map[string]bson.A
	err = json.Unmarshal(embeddingResp, &vectorData)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	result, err := mongoDAO.VectorFind(details.SearchIndex, details.Path, details.Project, details.Limit, vectorData["vector_embedding"], ctx)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bson.M{"data": result})
}

type MongoVector struct {
	SearchIndex string `json:"searchIndex,omitempty"`
	Path        string `json:"path,omitempty"`
	Text        string `json:"text,omitempty"`
	Project     string `json:"project,omitempty"`
	Limit       int64  `json:"limit,omitempty"`
}

func UnmarshalMVPayload(r *http.Request) (MongoVector, error) {
	var user MongoVector
	body, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(body, &user)
	if err != nil {
		return user, err
	}
	return user, err
}
