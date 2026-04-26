package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	serpAPIEndpoint     = "https://serpapi.com/search.json"
	defaultSearchEngine = "amazon"
	defaultSearchLimit  = 5
	maxSearchLimit      = 20
)

type Item struct {
	Rating     float64 `json:"rating"`
	Price      string  `json:"price"`
	Title      string  `json:"title"`
	AmazonLink string  `json:"amazon_link"`
	ImageURL   string  `json:"image_url"`
}

type Service struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

type serpAPIResponse struct {
	Error          string           `json:"error"`
	OrganicResults []map[string]any `json:"organic_results"`
}

func NewServiceFromEnv() (*Service, error) {
	apiKey := firstEnvValue("SERPAPI_API_KEY", "SERPAPI_KEY", "SERP_API_KEY")
	if apiKey == "" {
		return nil, errors.New("SERPAPI_API_KEY (or SERP_API_KEY) is not set")
	}

	return &Service{
		apiKey:   apiKey,
		endpoint: serpAPIEndpoint,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}, nil
}

func firstEnvValue(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func (s *Service) SearchAmazon(ctx context.Context, query string, limit int) ([]Item, error) {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return nil, errors.New("query cannot be empty")
	}

	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	params := url.Values{}
	params.Set("engine", defaultSearchEngine)
	params.Set("api_key", s.apiKey)
	params.Set("amazon_domain", "amazon.com")
	params.Set("k", trimmedQuery)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build SerpApi request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SerpApi request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("serpapi returned non-200 response: %s", resp.Status)
	}

	var payload serpAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode SerpApi response: %w", err)
	}
	if payload.Error != "" {
		return nil, fmt.Errorf("serpapi error: %s", payload.Error)
	}

	results := make([]Item, 0, limit)
	for _, raw := range payload.OrganicResults {
		item := normalizeOrganicResult(raw)
		if item.Title == "" || item.AmazonLink == "" {
			continue
		}
		results = append(results, item)
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

func SearchAmazonToolCall(args map[string]any) (string, error) {
	query := getStringArg(args, "query")
	if query == "" {
		query = getStringArg(args, "k")
	}
	if query == "" {
		return "", errors.New("missing required argument: query")
	}

	limit := getIntArg(args, "limit")

	service, err := NewServiceFromEnv()
	if err != nil {
		return "", err
	}

	items, err := service.SearchAmazon(context.Background(), query, limit)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(items)
	if err != nil {
		return "", fmt.Errorf("failed to serialize search results: %w", err)
	}

	return string(data), nil
}

func normalizeOrganicResult(raw map[string]any) Item {
	amazonLink := getString(raw, "link")
	asin := getString(raw, "asin")
	if amazonLink == "" && asin != "" {
		amazonLink = "https://www.amazon.com/dp/" + asin
	}

	imageURL := getString(raw, "thumbnail")
	if imageURL == "" {
		imageURL = getString(raw, "image")
	}

	return Item{
		Rating:     getFloat(raw, "rating"),
		Price:      extractPrice(raw),
		Title:      getString(raw, "title"),
		AmazonLink: amazonLink,
		ImageURL:   imageURL,
	}
}

func extractPrice(raw map[string]any) string {
	if priceRaw := extractPriceFromMap(raw, "price"); priceRaw != "" {
		return priceRaw
	}

	pricesAny, ok := raw["prices"].([]any)
	if ok {
		for _, candidate := range pricesAny {
			if priceMap, isMap := candidate.(map[string]any); isMap {
				if priceRaw := extractPriceFromAnyMap(priceMap); priceRaw != "" {
					return priceRaw
				}
			}
		}
	}

	if value, ok := toFloat(raw["extracted_price"]); ok {
		return fmt.Sprintf("$%.2f", value)
	}

	if rawPrice := getString(raw, "price_string"); rawPrice != "" {
		return rawPrice
	}

	return ""
}

func extractPriceFromMap(raw map[string]any, key string) string {
	mapValue, ok := raw[key].(map[string]any)
	if !ok {
		return ""
	}

	return extractPriceFromAnyMap(mapValue)
}

func extractPriceFromAnyMap(priceMap map[string]any) string {
	if raw := getString(priceMap, "raw"); raw != "" {
		return raw
	}

	value, ok := toFloat(priceMap["value"])
	if !ok {
		return ""
	}

	symbol := getString(priceMap, "symbol")
	if symbol == "" {
		symbol = "$"
	}
	return fmt.Sprintf("%s%.2f", symbol, value)
}

func getStringArg(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func getIntArg(args map[string]any, key string) int {
	value, ok := args[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case string:
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func getString(data map[string]any, key string) string {
	raw, ok := data[key]
	if !ok {
		return ""
	}
	asString, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(asString)
}

func getFloat(data map[string]any, key string) float64 {
	raw, ok := data[key]
	if !ok {
		return 0
	}

	value, ok := toFloat(raw)
	if !ok {
		return 0
	}
	return value
}

func toFloat(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		value, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return value, true
	case string:
		value, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		return value, true
	default:
		return 0, false
	}
}
