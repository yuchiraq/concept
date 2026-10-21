package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func loginOwnerForTest(t *testing.T, handler http.Handler, password string) (*http.Cookie, *httptest.ResponseRecorder) {
	t.Helper()
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "https://qoncept.example/site/qr/login", nil))
	match := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(page.Body.String())
	if len(match) != 2 {
		t.Fatalf("login form missing: %d", page.Code)
	}
	form := url.Values{"csrf": {match[1]}, "password": {password}, "next": {"/site/qr?campaign=vizitka"}}
	request := httptest.NewRequest("POST", "https://qoncept.example/site/qr/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range page.Result().Cookies() {
		request.AddCookie(cookie)
	}
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	for _, cookie := range result.Result().Cookies() {
		if cookie.Name == ownerCookie {
			return cookie, result
		}
	}
	return nil, result
}

func TestOwnerPasswordSessions(t *testing.T) {
	f := defaultFeatures()
	f.adminPassword = "test-owner-password"
	now := time.Now()
	f.now = func() time.Time { return now }
	handler, err := newHandler("https://qoncept.example", f)
	if err != nil {
		t.Fatal(err)
	}
	cookie, result := loginOwnerForTest(t, handler, "wrong-password")
	if cookie != nil || result.Code != 401 || result.Header().Get("WWW-Authenticate") != "" || !strings.Contains(result.Body.String(), "Пароль не подошёл") {
		t.Fatal("wrong password did not render a form error")
	}
	cookie, result = loginOwnerForTest(t, handler, "test-owner-password")
	if cookie == nil || result.Code != 303 || result.Header().Get("Location") != "/site/qr?campaign=vizitka" {
		t.Fatal("password login failed")
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/site/qr" || cookie.MaxAge != 8*60*60 {
		t.Fatalf("unsafe session cookie: %+v", cookie)
	}
	request := httptest.NewRequest("GET", "https://qoncept.example/site/qr", nil)
	request.AddCookie(cookie)
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != 200 || !strings.Contains(result.Body.String(), "Выйти") {
		t.Fatal("session did not open dashboard")
	}
	match := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(result.Body.String())
	if len(match) != 2 {
		t.Fatal("logout CSRF missing")
	}
	logout := httptest.NewRequest("POST", "https://qoncept.example/site/qr/logout", strings.NewReader(url.Values{"csrf": {match[1]}}.Encode()))
	logout.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	logout.AddCookie(cookie)
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, logout)
	if result.Code != 303 {
		t.Fatal("logout failed")
	}
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != 303 {
		t.Fatal("logged out token still works")
	}
	cookie, _ = loginOwnerForTest(t, handler, "test-owner-password")
	now = now.Add(9 * time.Hour)
	request = httptest.NewRequest("GET", "https://qoncept.example/site/qr/code.png", nil)
	request.AddCookie(cookie)
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != 303 {
		t.Fatal("expired token still works")
	}
}

func TestOwnerLoginCSRFAndThrottling(t *testing.T) {
	f := defaultFeatures()
	f.adminPassword = "test-owner-password"
	handler, err := newHandler("https://qoncept.example", f)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "https://qoncept.example/site/qr/login", strings.NewReader("password=test-owner-password"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != 403 {
		t.Fatal("login without CSRF was accepted")
	}
	for i := 0; i < 9; i++ {
		_, result = loginOwnerForTest(t, handler, "wrong-password")
	}
	if result.Code != 429 || result.Header().Get("Retry-After") == "" {
		t.Fatal("password guessing is not limited")
	}
	for _, value := range []string{"https://evil.example/", "//evil.example/", "/site/qr/code.png", "/site/qr\r\nLocation: evil"} {
		if ownerNext(value) != "/site/qr" {
			t.Errorf("unsafe next accepted: %q", value)
		}
	}
	request = httptest.NewRequest("GET", "https://qoncept.example/site/qr", nil)
	request.SetBasicAuth("owner", "test-owner-password")
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != 303 || result.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("old basic auth is still active")
	}
}

func TestTelegramPhoneRemainsPhone(t *testing.T) {
	submission := validContact()
	submission.ContactMethod = "telegram"
	submission.Contact = "+375 (29) 123-45-67"
	if !submission.validate() || submission.Contact != "+375 (29) 123-45-67" {
		t.Fatal("Telegram phone was rejected or turned into a username")
	}
}

func TestNotFoundIsBrandedAndNotIndexed(t *testing.T) {
	handler := testHandler(t, "https://qoncept.example")
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, httptest.NewRequest("GET", "https://qoncept.example/missing-page", nil))
	if result.Code != 404 || !strings.Contains(result.Header().Get("Content-Type"), "text/html") || !strings.Contains(result.Header().Get("X-Robots-Tag"), "noindex") {
		t.Fatal("404 status or metadata missing")
	}
	for _, text := range []string{"glitch-code", "Страница не найдена", "На главную", "/static/motion.js"} {
		if !strings.Contains(result.Body.String(), text) {
			t.Errorf("404 lacks %s", text)
		}
	}
}
