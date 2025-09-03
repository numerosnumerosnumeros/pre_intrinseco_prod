package app

import (
	"encoding/json"
	"io"
	"net/http"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type URLReq struct {
	URL string `json:"url"`
}

const MaxResponseSize = 150 * 1024 * 1024 // 150MB

// STREAM THE RESPONSE!
func URL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Failed to read request body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var req URLReq
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Log.Error("Failed to unmarshal request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url := sanitize.Trim(req.URL, "")

	if !sanitize.URL(url) {
		logger.Log.Error("Invalid parameter", zap.String("url", url))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	httpClient := &http.Client{
		Timeout: 40 * time.Second,
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Log.Error("Failed to create request", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Add user agent
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; nodo.finance/1.0; contacto@nodo.finance)")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		logger.Log.Error("Failed to fetch URL", zap.Error(err), zap.String("url", url))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// check resp status code
	if resp.StatusCode != http.StatusOK {
		logger.Log.Error("Failed to fetch URL", zap.String("url", url), zap.Int("status_code", resp.StatusCode))
		w.WriteHeader(resp.StatusCode)
		w.Write([]byte("Failed to fetch URL"))
		return
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if length, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			if length > MaxResponseSize {
				logger.Log.Warn("Response too large",
					zap.String("url", url),
					zap.Int64("content_length", length),
					zap.Int64("max_size", MaxResponseSize))
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte("Response too large"))
				return
			}
		}
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	// Write the status code from original response
	w.WriteHeader(resp.StatusCode)

	// Stream response with size limit
	written, err := io.Copy(w, io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		logger.Log.Error("Error streaming response", zap.Error(err))
		return
	}

	// Check size limits
	if written >= MaxResponseSize {
		logger.Log.Warn("Response size limit reached",
			zap.String("url", url),
			zap.Int64("max_size", MaxResponseSize))
	}
}
