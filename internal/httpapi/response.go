package httpapi

import (
	"SuperChat/internal/model"
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, model.APIResponse{
		Success: false,
		Error:   message,
	})
}

func WriteSuccess(w http.ResponseWriter, data interface{}, message string) {
	WriteJSON(w, http.StatusOK, model.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}
