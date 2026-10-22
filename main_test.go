package main

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func TestSiteOrigin(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"https://qoncept.example/", "https://qoncept.example"},
		{" http://localhost:8089 ", "http://localhost:8089"},
		{"https://[::1]:8089/", "https://[::1]:8089"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := siteOrigin(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("siteOrigin(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
		})
	}
	for _, input := range []string{"", "example.com", "javascript:alert(1)", "ftp://example.com", "https://user:password@example.com", "https://example.com/subdir", "https://example.com/?q=1", "https://example.com/?", "https://example.com/#section", "https://example.com:bad", "https://example.com\r\nInjected: value"} {
		t.Run(input, func(t *testing.T) {
			if got, err := siteOrigin(input); err == nil {
				t.Fatalf("accepted invalid origin %q as %q", input, got)
			}
		})
	}
}

func TestRequestOrigin(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://localhost:8089/", nil)
	r.Header.Set("X-Forwarded-Host", "untrusted.example")
	r.Header.Set("X-Forwarded-Proto", "https")
	if got := requestOrigin(r); got != "http://localhost:8089" {
		t.Fatalf("origin = %q", got)
	}
	r.TLS = &tls.ConnectionState{}
	if got := requestOrigin(r); got != "https://localhost:8089" {
		t.Fatalf("TLS origin = %q", got)
	}
	r.Host = "attacker.example/path"
	if got := requestOrigin(r); got != "http://localhost:8089" {
		t.Fatalf("invalid host fallback = %q", got)
	}
}

func testHandler(t *testing.T, siteURL string) http.Handler {
	t.Helper()
	handler, err := newHandler(siteURL)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func getPage(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d: %s", target, w.Code, w.Body.String())
	}
	return w
}

func TestPagesRenderSEOAndProjectLinks(t *testing.T) {
	handler := testHandler(t, "https://qoncept.example/")
	jsonLDPattern := regexp.MustCompile(`(?s)<script\s+type="application/ld\+json">(.*?)</script>`)
	for _, page := range pages() {
		t.Run(page.ActivePath, func(t *testing.T) {
			w := getPage(t, handler, "http://localhost:8089"+page.ActivePath+"?utm_source=test")
			body := w.Body.String()
			for _, want := range []string{
				`<title>` + page.MetaTitle + `</title>`,
				`name="description" content="` + page.MetaDescription + `"`,
				`rel="canonical" href="https://qoncept.example` + page.ActivePath + `"`,
				`property="og:url" content="https://qoncept.example` + page.ActivePath + `"`,
				`property="og:image" content="https://qoncept.example/static/images/covers/cover.jpg"`,
				`name="twitter:card" content="summary_large_image"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("rendered page is missing %q", want)
				}
			}
			if strings.Contains(body, "appworks.avayusstroi.by") {
				t.Error("page exposes the WorkService external address")
			}
			if strings.Contains(body, "/static/images/optimized/") {
				t.Error("page still references removed cover directory")
			}
			if strings.Contains(body, "#ZgotmplZ") || strings.Contains(body, "ZgotmplZ") {
				t.Error("template rejected a URL or script value")
			}
			match := jsonLDPattern.FindStringSubmatch(body)
			if len(match) != 2 {
				t.Fatal("JSON-LD script not found")
			}
			var data struct {
				Context string           `json:"@context"`
				Graph   []map[string]any `json:"@graph"`
			}
			if err := json.Unmarshal([]byte(match[1]), &data); err != nil {
				t.Fatalf("JSON-LD is invalid: %v", err)
			}
			if data.Context != "https://schema.org" || len(data.Graph) < 3 {
				t.Fatalf("unexpected JSON-LD graph: %#v", data)
			}
			if data.Graph[2]["url"] != "https://qoncept.example"+page.ActivePath {
				t.Error("WebPage URL does not match canonical URL")
			}
			if page.ActivePath == "/" || page.ActivePath == "/websites" {
				for _, want := range []string{"WorkService", "Обсудить свой сервис", `href="https://avayusstroi.by/"`} {
					if !strings.Contains(body, want) {
						t.Errorf("project section missing %q", want)
					}
				}
			}
		})
	}
}

func TestCrawlerRoutes(t *testing.T) {
	handler := testHandler(t, "https://qoncept.example")
	robots := getPage(t, handler, "http://localhost:8089/robots.txt")
	if !strings.Contains(robots.Body.String(), "Sitemap: https://qoncept.example/sitemap.xml") || !strings.Contains(robots.Body.String(), "Allow: /") {
		t.Fatalf("unexpected robots.txt: %s", robots.Body.String())
	}
	w := getPage(t, handler, "http://localhost:8089/sitemap.xml")
	var document sitemapDocument
	if err := xml.Unmarshal(w.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.XMLName.Space != "http://www.sitemaps.org/schemas/sitemap/0.9" || len(document.URLs) != len(pages())*len(languages) {
		t.Fatalf("invalid sitemap: %#v", document)
	}
	for i, page := range pages() {
		for j, lang := range languages {
			if got := document.URLs[i*len(languages)+j].Location; got != "https://qoncept.example"+localizedURL(lang, page.ActivePath) {
				t.Errorf("sitemap location = %q", got)
			}
		}
	}
	if got := w.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
		t.Errorf("sitemap Content-Type = %q", got)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://localhost:8089/missing", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown page status = %d", w.Code)
	}
}

func TestLocalOriginDoesNotLeakAcrossRequests(t *testing.T) {
	handler := testHandler(t, "")
	var pending sync.WaitGroup
	for _, host := range []string{"localhost:8089", "preview.example", "second.example:9090"} {
		pending.Add(1)
		go func(host string) {
			defer pending.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://"+host+"/websites", nil))
			want := `rel="canonical" href="http://` + host + `/websites"`
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), want) {
				t.Errorf("host %s: status %d, canonical missing", host, w.Code)
			}
		}(host)
	}
	pending.Wait()
}

func TestStructuredDataEscapesScriptContent(t *testing.T) {
	page := PageData{SiteURL: "https://qoncept.example", CanonicalURL: "https://qoncept.example/", MetaTitle: `</script><script>alert("test")</script>`}
	encoded, err := structuredData(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "<") {
		t.Fatal("JSON-LD includes unescaped HTML")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("escaped JSON is invalid: %v", err)
	}
}

func TestArtworkAssetsExist(t *testing.T) {
	handler := testHandler(t, "")
	for _, art := range artworks() {
		w := getPage(t, handler, "http://localhost:8089/static/images/covers/"+art.Image)
		if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/") {
			t.Errorf("%s Content-Type = %q", art.Image, got)
		}
	}
}

func TestPresentationMediaAndIcons(t *testing.T) {
	handler := testHandler(t, "https://qoncept.example")
	videoPattern := regexp.MustCompile(`<video\b([^>]*)>`)
	controlsPattern := regexp.MustCompile(`(?:^|\s)controls(?:\s|=|$)`)
	for _, page := range pages() {
		body := getPage(t, handler, "https://qoncept.example"+page.ActivePath).Body.String()
		if strings.ContainsAny(body, "↗↖↘↙↑↓←→➡⬆⬇") {
			t.Errorf("%s contains font-dependent arrow glyphs", page.ActivePath)
		}
		if strings.Contains(body, `class="theme-toggle"`) {
			t.Errorf("%s still has a theme button", page.ActivePath)
		}
		if !strings.Contains(body, `class="icon-arrow"`) {
			t.Errorf("%s is missing drawn arrow icons", page.ActivePath)
		}
		videos := videoPattern.FindAllStringSubmatch(body, -1)
		if page.ActivePath == "/" && len(videos) != 4 {
			t.Errorf("home has %d videos; want hero and three media previews", len(videos))
		}
		for _, video := range videos {
			if controlsPattern.MatchString(video[1]) {
				t.Errorf("%s exposes native video controls", page.ActivePath)
			}
			for _, attribute := range []string{"data-motion-preview", "autoplay", "loop", "muted", "playsinline", "poster="} {
				if !strings.Contains(video[1], attribute) {
					t.Errorf("%s video lacks %s", page.ActivePath, attribute)
				}
			}
		}
	}
}
