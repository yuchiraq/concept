package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func validContact() contactSubmission {
	return contactSubmission{Name: "Тестовый клиент", ContactMethod: "telegram", Contact: "@test_client", Service: "software", Message: "Автоматизация заявок", Page: "/websites", Source: "Обсудить проект", UTMSource: "qr", UTMMedium: "offline", UTMCampaign: "vizitka"}
}
func contactRequest(t *testing.T, f *siteFeatures, s contactSubmission) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "http://localhost/api/contact", bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "127.0.0.1:9000"
	w := httptest.NewRecorder()
	f.contactHandler(w, r)
	return w
}
func TestContactDeliveryAndAttribution(t *testing.T) {
	var received atomic.Int32
	telegram := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/bottest-token/sendMessage" {
			t.Errorf("unexpected Telegram request %s %s", r.Method, r.URL.Path)
		}
		r.ParseForm()
		if r.FormValue("chat_id") != "private-chat" {
			t.Error("wrong recipient")
		}
		for _, part := range []string{"Тестовый клиент", "@test_client", "Сервис и автоматизация", "Автоматизация заявок", "Кампания: vizitka", "Страница: /websites"} {
			if !strings.Contains(r.FormValue("text"), part) {
				t.Errorf("notification lacks %q", part)
			}
		}
		received.Add(1)
		io.WriteString(w, `{"ok":true,"result":{"message_id":1}}`)
	}))
	defer telegram.Close()
	f := defaultFeatures()
	f.telegram = telegramDelivery{Token: "test-token", Chats: []string{"private-chat"}, BaseURL: telegram.URL, Client: telegram.Client()}
	w := contactRequest(t, f, validContact())
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"success":true`) {
		t.Fatalf("submission %d %s", w.Code, w.Body.String())
	}
	if received.Load() != 1 {
		t.Fatal("request was not delivered exactly once")
	}
	stats := f.tracker.snapshot()
	if stats.Leads != 1 || stats.ByCampaign["vizitka"].Leads != 1 {
		t.Fatalf("QR lead attribution lost: %+v", stats)
	}
	if strings.Contains(w.Body.String(), "test-token") || strings.Contains(w.Body.String(), "private-chat") {
		t.Fatal("configuration leaked to browser")
	}
}
func TestContactNeverReportsUndeliveredAsSuccess(t *testing.T) {
	f := defaultFeatures()
	if w := contactRequest(t, f, validContact()); w.Code != 503 {
		t.Fatalf("unconfigured status %d", w.Code)
	}
	for _, response := range []struct {
		status int
		body   string
	}{{200, `{"ok":false}`}, {502, `{"ok":true}`}, {200, `not-json`}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(response.status)
			io.WriteString(w, response.body)
		}))
		f := defaultFeatures()
		f.telegram = telegramDelivery{Token: "test", Chats: []string{"test-chat"}, BaseURL: server.URL, Client: server.Client()}
		w := contactRequest(t, f, validContact())
		server.Close()
		if w.Code != 502 {
			t.Errorf("rejected delivery produced %d", w.Code)
		}
		if f.tracker.snapshot().Leads != 0 {
			t.Error("failed delivery counted as lead")
		}
	}
}
func TestContactValidationAndRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"ok":true}`) }))
	defer server.Close()
	f := defaultFeatures()
	f.telegram = telegramDelivery{Token: "test", Chats: []string{"chat"}, BaseURL: server.URL, Client: server.Client()}
	invalid := []contactSubmission{validContact(), validContact(), validContact(), validContact(), validContact()}
	invalid[0].Name = ""
	invalid[1].Contact = "bad"
	invalid[2].Service = "arbitrary"
	invalid[3].Page = "https://other.test"
	invalid[4].Message = strings.Repeat("a", 1201)
	for _, s := range invalid {
		if w := contactRequest(t, f, s); w.Code != 422 {
			t.Errorf("invalid fields status %d", w.Code)
		}
	}
	for i := 0; i < 5; i++ {
		if w := contactRequest(t, f, validContact()); w.Code != 200 {
			t.Fatalf("allowed request %d status %d", i, w.Code)
		}
	}
	if w := contactRequest(t, f, validContact()); w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal("rate limit missing")
	}
	data, _ := json.Marshal(validContact())
	r := httptest.NewRequest(http.MethodPost, "http://localhost/api/contact", bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://foreign.test")
	w := httptest.NewRecorder()
	f.contactHandler(w, r)
	if w.Code != 403 {
		t.Fatalf("foreign origin status %d", w.Code)
	}
}
func TestContactChannelValidation(t *testing.T) {
	for _, tc := range []struct {
		method, contact string
		valid           bool
	}{{"phone", "+375 (29) 123-45-67", true}, {"phone", "123", false}, {"phone", "++375291234567", false}, {"email", "hello@example.com", true}, {"email", "not-mail", false}, {"telegram", "https://t.me/test_client", true}, {"telegram", "+375 (29) 123-45-67", true}, {"telegram", "375291234567", true}, {"telegram", "123", false}, {"telegram", "123456", false}, {"telegram", "@bad handle", false}} {
		s := validContact()
		s.ContactMethod = tc.method
		s.Contact = tc.contact
		if got := s.validate(); got != tc.valid {
			t.Errorf("%s %q: %v", tc.method, tc.contact, got)
		}
	}
}
func TestQRRedirectPersistenceAndConcurrentCounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	f := defaultFeatures()
	f.tracker = newQRTracker(path)
	if err := f.tracker.load(); err != nil {
		t.Fatal(err)
	}
	f.tracker.now = func() time.Time { return time.Date(2026, 10, 20, 22, 30, 0, 0, time.UTC) }
	handler, err := newHandler("https://qoncept.example", f)
	if err != nil {
		t.Fatal(err)
	}
	var pending sync.WaitGroup
	for i := 0; i < 20; i++ {
		pending.Add(1)
		go func() {
			defer pending.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://qoncept.example/qr?campaign=Street%20Banner", nil))
			if w.Code != 302 {
				t.Errorf("QR status %d", w.Code)
			}
			if !strings.Contains(w.Header().Get("Location"), "utm_campaign=street-banner") || !strings.Contains(w.Header().Get("Location"), "utm_source=qr") {
				t.Error("QR attribution missing")
			}
			if w.Header().Get("Cache-Control") == "" {
				t.Error("redirect may be cached")
			}
		}()
	}
	pending.Wait()
	restored := newQRTracker(path)
	if err := restored.load(); err != nil {
		t.Fatal(err)
	}
	stats := restored.snapshot()
	if stats.Total != 20 || stats.ByDate["2026-10-21"] != 20 || stats.ByCampaign["street-banner"].Scans != 20 {
		t.Fatalf("wrong persisted stats %+v", stats)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodHead, "https://qoncept.example/qr", nil))
	if f.tracker.snapshot().Total != 20 {
		t.Error("HEAD inflated counts")
	}
	bytes, err := os.ReadFile(path)
	if err != nil || !json.Valid(bytes) {
		t.Fatal("persistent statistics corrupted")
	}
}
func TestQRDashboardAccessAndPNG(t *testing.T) {
	f := defaultFeatures()
	handler, err := newHandler("https://qoncept.example", f)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://qoncept.example/site/qr", nil))
	if w.Code != http.StatusSeeOther {
		t.Fatal("unconfigured public admin page is exposed")
	}
	f.adminPassword = "private-password"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://qoncept.example/site/qr", nil))
	if w.Code != http.StatusSeeOther || w.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("owner authentication is not required")
	}
	cookie, _ := loginOwnerForTest(t, handler, "private-password")
	for _, path := range []string{"/site/qr?campaign=vizitka", "/site/qr/code.png?campaign=vizitka"} {
		r := httptest.NewRequest(http.MethodGet, "https://qoncept.example"+path, nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("owner route %s: %d %s", path, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Header().Get("X-Robots-Tag"), "noindex") {
			t.Error("owner route is indexable")
		}
		if strings.Contains(path, "code.png") {
			image, err := png.DecodeConfig(w.Body)
			if err != nil || image.Width != 768 || image.Height != 768 {
				t.Fatalf("invalid QR PNG %+v %v", image, err)
			}
		} else if !strings.Contains(w.Body.String(), "https://qoncept.example/qr?campaign=vizitka") {
			t.Error("QR tracks wrong site")
		}
	}
}
func TestContactDialogExistsAndIsWired(t *testing.T) {
	handler := testHandler(t, "")
	for _, path := range []string{"/", "/websites", "/design"} {
		body := getPage(t, handler, "http://localhost"+path).Body.String()
		for _, want := range []string{`id="lead-dialog"`, `id="lead-form"`, `data-open-contact`, `/static/contact.js`, `Мои контакты`} {
			if !strings.Contains(body, want) {
				t.Errorf("%s lacks %s", path, want)
			}
		}
	}
}

func TestPrivateFilesAreNotServed(t *testing.T) {
	handler := testHandler(t, "")
	for _, path := range []string{"/settings.json", "/settings.example.json", "/data/qr_stats.json"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://localhost"+path, nil))
		if w.Code != 404 {
			t.Errorf("private path %s: %d", path, w.Code)
		}
	}
}

func TestRateLimitUsesOnlyExplicitlyTrustedLocalProxy(t *testing.T) {
	f := defaultFeatures()
	f.trustedProxyHeaders = true
	for i := 0; i < 5; i++ {
		r := httptest.NewRequest(http.MethodPost, "http://localhost/api/contact", nil)
		r.RemoteAddr = "127.0.0.1:8000"
		r.Header.Set("X-Real-IP", "192.0.2.1")
		if !f.allowContact(r) {
			t.Fatal("proxy client rejected early")
		}
	}
	r := httptest.NewRequest(http.MethodPost, "http://localhost/api/contact", nil)
	r.RemoteAddr = "127.0.0.1:8000"
	r.Header.Set("X-Real-IP", "192.0.2.2")
	if !f.allowContact(r) {
		t.Fatal("distinct visitors behind proxy share one limit")
	}
	f = defaultFeatures()
	f.trustedProxyHeaders = true
	for i := 0; i < 6; i++ {
		r := httptest.NewRequest(http.MethodPost, "http://localhost/api/contact", nil)
		r.RemoteAddr = "198.51.100.1:8000"
		r.Header.Set("X-Real-IP", fmt.Sprintf("203.0.113.%d", i+1))
		allowed := f.allowContact(r)
		if i == 5 && allowed {
			t.Fatal("untrusted remote headers bypass rate limit")
		}
	}
}

func TestConfiguredOriginWorksBehindLocalProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"ok":true}`) }))
	defer server.Close()
	f := defaultFeatures()
	f.telegram = telegramDelivery{Token: "test", Chats: []string{"chat"}, BaseURL: server.URL, Client: server.Client()}
	handler, err := newHandler("https://qoncept.example", f)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(validContact())
	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8090/api/contact", bytes.NewReader(data))
	r.Header.Set("Origin", "https://qoncept.example")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("same-site proxy request failed: %d %s", w.Code, w.Body.String())
	}
}
