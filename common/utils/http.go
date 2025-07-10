package utils

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/LexiconIndonesia/crawler-http-service/common/models"
)

// WriteJSON writes a JSON response with the given status code and data
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	response := models.BaseResponse{
		Data: data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// WriteMessage writes a JSON response with the given status code and message
func WriteMessage(w http.ResponseWriter, statusCode int, message string) {
	response := models.BaseResponse{
		Data: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// WriteError writes a JSON response with the given status code and error message
func WriteError(w http.ResponseWriter, statusCode int, errorMessage string) {
	response := models.ErrorResponse{
		Error: http.StatusText(statusCode),
		Msg:   errorMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// WritePagination writes a JSON response with pagination metadata
func WritePagination(w http.ResponseWriter, statusCode int, data interface{}, currentPage, perPage int, total int64) {
	lastPage := int64(math.Ceil(float64(total) / float64(perPage)))

	meta := models.MetaResponse{
		CurrentPage: int64(currentPage),
		LastPage:    lastPage,
		PerPage:     int64(perPage),
		Total:       total,
	}

	response := models.BasePaginationResponse{
		Data: data,
		Meta: meta,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
