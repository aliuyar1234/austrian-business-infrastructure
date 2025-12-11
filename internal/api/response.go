package api

import (
	"encoding/json"
	"net/http"
)

// RespondJSON sends a JSON response with the given status code and data.
// This is a simpler version of JSONResponse for handlers that don't need
// structured error responses.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// RespondError sends a JSON error response with a simple error message.
// For structured error responses with codes, use JSONError instead.
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{"error": message})
}

// RespondErrorWithDetails sends a JSON error response with additional details.
// This is useful for error responses that need to include extra context.
func RespondErrorWithDetails(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error": message,
	}
	if err != nil {
		response["details"] = err.Error()
	}
	RespondJSON(w, status, response)
}
