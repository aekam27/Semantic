package elastic_service

import (
	"encoding/json"
	util "goverse/pkg/utils"
	"net/http"
)

func GetProductsList(w http.ResponseWriter, r *http.Request) {
	util.GenerateToken("1", "email", "Token A", "active")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("true")
}
