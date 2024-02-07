package auth_service

import (
	"encoding/json"
	util "goverse/pkg/utils"
	"io"
	"net/http"
)

func GenerateToken(w http.ResponseWriter, r *http.Request) {
	details, err := UnmarshaTokenPayload(r)
	if err != nil {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]string{"token": ""})
		return
	}
	token, err := util.GenerateToken("", details.Email, details.Name, "Active")
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

type Creds struct {
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

func UnmarshaTokenPayload(r *http.Request) (Creds, error) {
	var user Creds
	body, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(body, &user)
	if err != nil {
		return user, err
	}
	return user, err
}
