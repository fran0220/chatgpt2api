package api

import "net/http"

func handleModels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list",
			"data": []map[string]any{
				{
					"id":       "gpt-image-1",
					"object":   "model",
					"created":  1700000000,
					"owned_by": "openai",
				},
			},
		})
	}
}
