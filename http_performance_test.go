package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProductionOriginAndExplicitPreview(t *testing.T) {
	t.Setenv("LOCAL_PREVIEW", "")
	if got := publicSiteURL("", ":8089"); got != productionOrigin {
		t.Fatalf("default public URL = %q", got)
	}
	t.Setenv("LOCAL_PREVIEW", "1")
	for _, address := range []string{"127.0.0.1:8089", "[::1]:8089"} {
		if got := publicSiteURL("", address); got != "" {
			t.Errorf("local preview %s = %q", address, got)
		}
	}
	if got := publicSiteURL("", ":8089"); got != productionOrigin {
		t.Fatal("wildcard bind must not enable local preview")
	}
	if got := publicSiteURL("https://other.example", "127.0.0.1:8089"); got != "https://other.example" {
		t.Fatal("explicit origin must win")
	}
	handler := testHandler(t, publicSiteURL("", ":8089"))
	body := getPage(t, handler, "http://localhost:8089/software").Body.String()
	if !strings.Contains(body, `rel="canonical" href="https://qoncept.by/software"`) || strings.Contains(body, "http://localhost:8089") {
		t.Fatal("production reverse-proxy response leaks localhost")
	}
	for _, host := range []string{"qoncept.by", "www.qoncept.by", "QONCEPT.BY:80"} {
		if got := requestOrigin(httptest.NewRequest("GET", "http://"+host+"/", nil)); got != productionOrigin {
			t.Errorf("public host %q = %q", host, got)
		}
	}
}

func TestGzipRoundTripAndNegotiation(t *testing.T) {
	handler := testHandler(t, productionOrigin)
	plain := getPage(t, handler, "http://localhost/software").Body.String()
	for _, tc := range []struct {
		accept     string
		compressed bool
	}{{"gzip", true}, {"br, gzip;q=0.5", true}, {"GZIP; q = 1", true}, {"gzip;q=0", false}, {"gzip; q = 0.000", false}, {"gzip;q=no", false}, {"br", false}} {
		r := httptest.NewRequest("GET", "http://localhost/software", nil)
		r.Header.Set("Accept-Encoding", tc.accept)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if (w.Header().Get("Content-Encoding") == "gzip") != tc.compressed {
			t.Fatalf("%q gzip negotiation incorrect", tc.accept)
		}
		if !tc.compressed {
			continue
		}
		compressedSize := w.Body.Len()
		reader, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		reader.Close()
		if err != nil || string(body) != plain {
			t.Fatal("gzip response changed HTML")
		}
		if compressedSize >= len(plain) {
			t.Fatal("gzip did not reduce page size")
		}
	}
}

func TestVideoRangesAndVersionedAssets(t *testing.T) {
	handler := testHandler(t, productionOrigin)
	r := httptest.NewRequest("GET", "http://localhost/static/videos/previews/hero-ogbigtoy.mp4", nil)
	r.Header.Set("Range", "bytes=0-99")
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusPartialContent || w.Body.Len() != 100 || w.Header().Get("Content-Encoding") != "" || !strings.HasPrefix(w.Header().Get("Content-Range"), "bytes 0-99/") {
		t.Fatalf("video range broken: status=%d bytes=%d headers=%v", w.Code, w.Body.Len(), w.Header())
	}
	versioned := assetURL("/static/styles.css")
	if !strings.Contains(versioned, "?v=") {
		t.Fatal("CSS not versioned")
	}
	w = getPage(t, handler, "http://localhost"+versioned)
	if !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatal("versioned asset not cached")
	}
}
