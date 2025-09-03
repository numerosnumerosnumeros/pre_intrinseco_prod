package static

import (
	"bytes"
	"compress/gzip"
	"mime"
	"net/http"
	"nodofinance/utils/logger"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

var cache map[string]*fileContent

type fileContent struct {
	Compressed   []byte
	Uncompressed []byte
	ContentType  string
}

func CacheFrontend(baseDir string) error {
	cache = make(map[string]*fileContent)
	distDir := filepath.Join(baseDir, "./dist")

	return filepath.Walk(distDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil // Skip directories, continue walking
		}

		// Convert file path to URL path
		relPath, err := filepath.Rel(distDir, path)
		if err != nil {
			return err
		}
		urlPath := "/" + filepath.ToSlash(relPath)

		// Read and process file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var compressed bytes.Buffer
		gz, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
		if err != nil {
			logger.Log.Fatal("Failed to create gzip writer", zap.Error(err))
		}

		gz.Write(content)
		gz.Close()

		cache[urlPath] = &fileContent{
			Compressed:   compressed.Bytes(),
			Uncompressed: content,
			ContentType:  mime.TypeByExtension(filepath.Ext(path)),
		}

		return nil
	})
}

func Handler(devMode bool, allowedMethods []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if devMode {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
			w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, "+strings.Join(allowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Csrf-Token, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' "+
				"https://js.stripe.com https://www.googletagmanager.com https://player.vimeo.com; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob: https://www.googletagmanager.com "+
				"https://www.google-analytics.com https://*.vimeo.com https://i.vimeocdn.com; "+
				"connect-src 'self' https://region1.google-analytics.com "+
				"https://*.stripe.com https://*.vimeo.com https://vimeo.com "+
				"https://api.vimeo.com https://player.vimeo.com blob:; "+
				"font-src 'self'; "+
				"frame-src 'self' https://js.stripe.com https://player.vimeo.com; "+
				"frame-ancestors 'none'; "+
				"form-action 'self';")

		path := r.URL.Path

		file, exists := cache[path]

		if !exists {
			if strings.HasPrefix(path, "/api/") {
				http.NotFound(w, r)
				return
			}

			indexFile, indexExists := cache["/index.html"]
			if !indexExists {
				http.NotFound(w, r)
				return
			}

			file = indexFile
		}

		w.Header().Set("Content-Type", file.ContentType)

		if strings.HasSuffix(path, "index.html") || !strings.Contains(path, ".") {
			w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=60")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")
			w.Write(file.Compressed)
		} else {
			w.Write(file.Uncompressed)
		}
	}
}
