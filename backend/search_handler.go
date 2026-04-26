package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"x/search"
)

type amazonSearchResponse struct {
	Items []search.Item `json:"items"`
}

func amazon_search_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("expected GET request method, found %s", r.Method),
		})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("q"))
	}
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("k"))
	}
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "missing required search query",
		})
		return
	}

	limit := 1
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "limit must be a valid integer",
			})
			return
		}
		limit = parsedLimit
	}

	service, err := search.NewServiceFromEnv()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	items, err := service.SearchAmazon(r.Context(), query, limit)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	for i := range items {
		items[i].ImageURL = buildImageProxyURL(r, items[i].ImageURL)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(amazonSearchResponse{Items: items})
}
