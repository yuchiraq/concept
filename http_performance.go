package main

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func productionHost(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	return host == "qoncept.by" || host == "www.qoncept.by"
}

func publicSiteURL(configured, address string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	host, _, _ := net.SplitHostPort(address)
	if os.Getenv("LOCAL_PREVIEW") == "1" && net.ParseIP(host).IsLoopback() {
		return ""
	}
	return productionOrigin
}

var assetVersions sync.Map

// Content versions let browsers keep CSS and scripts cached across visits while
// a new build immediately requests the updated file.
func assetURL(path string) string {
	if value, ok := assetVersions.Load(path); ok {
		return value.(string)
	}
	if !strings.HasPrefix(path, "/static/") || strings.Contains(path, "..") {
		return path
	}
	data, err := os.ReadFile(filepath.FromSlash(strings.TrimPrefix(path, "/")))
	if err != nil {
		return path
	}
	digest := sha256.Sum256(data)
	result := fmt.Sprintf("%s?v=%x", path, digest[:6])
	assetVersions.Store(path, result)
	return result
}

func staticAssets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		next.ServeHTTP(w, r)
	})
}

type compressedWriter struct {
	http.ResponseWriter
	gzip  *gzip.Writer
	wrote bool
}

func (w *compressedWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	kind := w.Header().Get("Content-Type")
	compressible := strings.HasPrefix(kind, "text/") || strings.HasPrefix(kind, "application/javascript") || strings.HasPrefix(kind, "application/json") || strings.HasPrefix(kind, "application/xml") || strings.HasPrefix(kind, "image/svg+xml")
	if compressible && status >= 200 && status != 204 && status != 304 && status != 206 && w.Header().Get("Content-Encoding") == "" {
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
		w.gzip, _ = gzip.NewWriterLevel(w.ResponseWriter, gzip.BestSpeed)
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *compressedWriter) Write(data []byte) (int, error) {
	if !w.wrote {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(data))
		}
		w.WriteHeader(http.StatusOK)
	}
	if w.gzip != nil {
		return w.gzip.Write(data)
	}
	return w.ResponseWriter.Write(data)
}
func (w *compressedWriter) ReadFrom(reader io.Reader) (int64, error) {
	return io.Copy(struct{ io.Writer }{w}, reader)
}
func compressResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")
		accepts := false
		for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
			fields := strings.Split(strings.TrimSpace(part), ";")
			if !strings.EqualFold(strings.TrimSpace(fields[0]), "gzip") {
				continue
			}
			accepts = true
			for _, param := range fields[1:] {
				key, value, ok := strings.Cut(strings.TrimSpace(param), "=")
				if ok && strings.EqualFold(strings.TrimSpace(key), "q") {
					quality, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
					accepts = err == nil && quality > 0 && quality <= 1
				}
			}
		}
		if !accepts || r.Method == http.MethodHead || r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}
		wrapped := &compressedWriter{ResponseWriter: w}
		defer func() {
			if wrapped.gzip != nil {
				wrapped.gzip.Close()
			}
		}()
		next.ServeHTTP(wrapped, r)
	})
}
