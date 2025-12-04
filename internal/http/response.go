package http

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response to the client
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// If encoding fails, try to write a simple error message
			w.Write([]byte(`{"error":"Failed to encode response"}`))
		}
	}
}
