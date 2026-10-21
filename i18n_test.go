package main

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestRegionalLanguageSelection(t *testing.T) {
	f := defaultFeatures()
	for _, tc := range []struct{ header, want string }{
		{"be-BY,be;q=0.9,en;q=0.8", "ru"}, {"en-BY,en;q=0.9", "ru"}, {"ru-BY", "ru"}, {"uk-UA,ru;q=0.9", "uk"}, {"ru-UA", "uk"}, {"en-US", "en"}, {"fr-FR,en;q=0.8", "en"}, {"uk;q=0.2,ru;q=0.8", "ru"}, {"ru;q=0,en;q=0.5", "en"}, {"be", "ru"}, {"", "ru"},
	} {
		r := httptest.NewRequest("GET", "http://localhost/", nil)
		r.Header.Set("Accept-Language", tc.header)
		if got := f.automaticLanguage(r); got != tc.want {
			t.Errorf("%q: %s, want %s", tc.header, got, tc.want)
		}
	}
	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.Header.Set("Accept-Language", "be-BY")
	r.AddCookie(&http.Cookie{Name: "qoncept_language", Value: "be"})
	if got := f.automaticLanguage(r); got != "be" {
		t.Fatal("manual Belarusian does not override automatic Russian")
	}
	r = httptest.NewRequest("GET", "http://localhost/", nil)
	r.RemoteAddr = "127.0.0.1:8000"
	r.Header.Set("Accept-Language", "en-US")
	r.Header.Set("CF-IPCountry", "BY")
	if f.automaticLanguage(r) != "en" {
		t.Fatal("unconfigured country header was trusted")
	}
	f.trustedProxyHeaders = true
	if f.automaticLanguage(r) != "ru" {
		t.Fatal("trusted Belarus country was ignored")
	}
	r.RemoteAddr = "192.0.2.4:8000"
	if f.automaticLanguage(r) != "en" {
		t.Fatal("remote country header was trusted")
	}
	r.AddCookie(&http.Cookie{Name: "qoncept_region", Value: "BY"})
	if f.automaticLanguage(r) != "ru" {
		t.Fatal("Belarus timezone hint did not select Russian")
	}
	r.AddCookie(&http.Cookie{Name: "qoncept_language", Value: "be"})
	if f.automaticLanguage(r) != "be" {
		t.Fatal("timezone hint overrode manual Belarusian")
	}
}

func TestLocalizedPagesAndSEO(t *testing.T) {
	handler := testHandler(t, "https://qoncept.example")
	for _, lang := range languages {
		for _, page := range pages() {
			path := localizedURL(lang, page.ActivePath)
			w := getPage(t, handler, "https://qoncept.example"+path)
			body := w.Body.String()
			for _, want := range []string{`lang="` + lang + `"`, `rel="canonical" href="https://qoncept.example` + path + `"`, `hreflang="uk"`, `class="language-switcher"`, html.EscapeString(translate(lang, page.MetaTitle)), `data-page="` + page.ActivePath + `"`} {
				if !strings.Contains(body, want) {
					t.Errorf("%s missing %q", path, want)
				}
			}
			if w.Header().Get("Content-Language") != lang {
				t.Errorf("%s wrong Content-Language", path)
			}
			if lang != "ru" && !strings.Contains(body, `href="/`+lang+`/websites"`) {
				t.Errorf("%s lost language on navigation", path)
			}
			if strings.Contains(body, "ZgotmplZ") {
				t.Errorf("%s contains rejected template URL", path)
			}
			match := regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).FindStringSubmatch(body)
			var graph struct {
				Graph []map[string]any `json:"@graph"`
			}
			if len(match) != 2 || json.Unmarshal([]byte(match[1]), &graph) != nil {
				t.Fatalf("%s invalid structured data", path)
			}
			if graph.Graph[2]["inLanguage"] != lang {
				t.Errorf("%s wrong JSON-LD language", path)
			}
		}
	}
	for _, lang := range []string{"en", "uk", "be"} {
		r := httptest.NewRequest("GET", "https://qoncept.example/site/qr/login", nil)
		r.AddCookie(&http.Cookie{Name: "qoncept_language", Value: lang})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if !strings.Contains(w.Body.String(), translate(lang, "Пароль кабинета")) || !strings.Contains(w.Body.String(), `lang="`+lang+`"`) {
			t.Errorf("%s owner login is not translated", lang)
		}
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "https://qoncept.example/"+lang+"/missing", nil))
		if w.Code != 404 || !strings.Contains(w.Body.String(), translate(lang, "Страница не найдена.")) {
			t.Errorf("%s 404 is not translated", lang)
		}
	}
}

func TestLanguageSwitchPersistenceAndSafeRedirects(t *testing.T) {
	handler := testHandler(t, "")
	r := httptest.NewRequest("GET", "http://localhost/language?lang=be&next=%2Fen%2Fwebsites", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 303 || w.Header().Get("Location") != "/be/websites" {
		t.Fatalf("bad switch: %d %s", w.Code, w.Header().Get("Location"))
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != "be" || cookies[0].MaxAge != 365*24*3600 {
		t.Fatal("preference not persisted")
	}
	r = httptest.NewRequest("GET", "http://localhost/websites?utm_source=qr", nil)
	r.Header.Set("Accept-Language", "be-BY")
	r.AddCookie(cookies[0])
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 302 || w.Header().Get("Location") != "/be/websites?utm_source=qr" {
		t.Fatal("manual choice lost to region or QR attribution lost")
	}
	r = httptest.NewRequest("GET", "http://localhost/en/", nil)
	r.AddCookie(&http.Cookie{Name: "qoncept_language", Value: "ru"})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("Content-Language") != "en" {
		t.Fatal("explicit shared locale URL was overridden")
	}
	for _, next := range []string{"https%3A%2F%2Fevil.example", "%2F%2Fevil.example", "%2F%252F%252Fevil.example", "%2F%255Cevil.example"} {
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/language?lang=ru&next="+next, nil))
		if w.Code != 303 || w.Header().Get("Location") != "/" {
			t.Errorf("unsafe language redirect: %s -> %s", next, w.Header().Get("Location"))
		}
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/language?lang=auto&next=%2Fbe%2Fwebsites", nil))
	if w.Code != 303 || w.Header().Get("Location") != "/websites" || len(w.Result().Cookies()) != 1 || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("automatic language mode was not restored")
	}
}

func TestTranslationCatalogCoverage(t *testing.T) {
	if err := loadTranslations(); err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	files, err := filepath.Glob("templates/*.html")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`\{\{t\s+("(?:[^"\\]|\\.)*")\s*\}\}`)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range pattern.FindAllStringSubmatch(string(data), -1) {
			key, _ := strconv.Unquote(m[1])
			keys[key] = true
		}
	}
	for _, key := range clientTranslationKeys {
		keys[key] = true
	}
	for _, page := range pages() {
		for _, key := range []string{page.MetaTitle, page.MetaDescription, page.HeroText, page.HeroTitle, page.Eyebrow} {
			if key != "" {
				keys[key] = true
			}
		}
	}
	for _, project := range projects() {
		for _, key := range append(append([]string{project.Name, project.Category, project.Description, project.CTALabel}, project.Tags...), project.Details...) {
			if strings.ContainsAny(key, "АБВГДЕЖЗИКЛМНОПРСТУФХЦЧШЩЭЮЯабвгдежзиклмнопрстуфхцчшщэюя") {
				keys[key] = true
			}
		}
	}
	for key := range keys {
		for _, lang := range []string{"en", "uk", "be"} {
			if translations[key][lang] == "" {
				t.Errorf("missing %s translation: %s", lang, key)
			}
		}
	}
}
