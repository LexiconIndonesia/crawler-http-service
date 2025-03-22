package middlewares

import (
	"encoding/json"
	"net/http"
)

func middlewareError(w http.ResponseWriter, s int, err string, m string) {
	var j struct {
		Error string `json:"error"`
		Msg   string `json:"message"`
	}

	j.Error = err
	j.Msg = m

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)

	if err := json.NewEncoder(w).Encode(j); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

}
