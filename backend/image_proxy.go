package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var imageProxyClient = &http.Client{
	Timeout: 20 * time.Second,
}

func amazon_image_handler(w http.ResponseWriter, r *http.Request) {
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		http.Error(w, "missing image url", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Host == "" {
		http.Error(w, "invalid image url", http.StatusBadRequest)
		return
	}

	if !isAllowedAmazonImageHost(parsedURL.Hostname()) {
		http.Error(w, "image host is not allowed", http.StatusBadRequest)
		return
	}

	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		http.Error(w, "failed to build image request", http.StatusInternalServerError)
		return
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PreCog/1.0)")
	request.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")

	response, err := imageProxyClient.Do(request)
	if err != nil {
		http.Error(w, "failed to fetch remote image", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("remote image returned %s", response.Status), http.StatusBadGateway)
		return
	}

	if contentType := response.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if cacheControl := response.Header.Get("Cache-Control"); cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, response.Body)
}

func buildImageProxyURL(r *http.Request, rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return ""
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	}

	return fmt.Sprintf("%s://%s/amazon-image?url=%s", scheme, r.Host, url.QueryEscape(rawURL))
}

func isAllowedAmazonImageHost(host string) bool {
	normalizedHost := strings.ToLower(strings.TrimSpace(host))
	if normalizedHost == "" {
		return false
	}

	allowedSuffixes := []string{
		".media-amazon.com",
		".ssl-images-amazon.com",
	}
	allowedExactHosts := map[string]struct{}{
		"m.media-amazon.com":              {},
		"images-na.ssl-images-amazon.com": {},
	}

	if _, ok := allowedExactHosts[normalizedHost]; ok {
		return true
	}

	for _, suffix := range allowedSuffixes {
		if strings.HasSuffix(normalizedHost, suffix) {
			return true
		}
	}

	return false
}
