package main

import (
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func qrEditForTest(t *testing.T, handler http.Handler, cookie *http.Cookie, campaign, destination, token string, addUTM bool) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"campaign": {campaign}, "destination": {destination}, "csrf": {token}}
	if addUTM {
		form.Set("add_utm", "1")
	}
	r := httptest.NewRequest(http.MethodPost, "https://qoncept.example/site/qr/save", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://qoncept.example")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func qrOwnerTokenForTest(t *testing.T, handler http.Handler, cookie *http.Cookie) string {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "https://qoncept.example/site/qr", nil)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	match := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
	if w.Code != http.StatusOK || len(match) != 2 {
		t.Fatalf("dashboard CSRF missing: %d", w.Code)
	}
	return match[1]
}

func TestQRDestinationsPersistAndKeepPrintedCodesAndCounts(t *testing.T) {
	f := defaultFeatures()
	f.adminPassword = "owner-test"
	f.tracker = newQRTracker(filepath.Join(t.TempDir(), "qr.json"))
	if err := f.tracker.record("vizitka", false); err != nil {
		t.Fatal(err)
	}
	if err := f.tracker.recordLead("vizitka"); err != nil {
		t.Fatal(err)
	}
	handler, err := newHandler("https://qoncept.example", f)
	if err != nil {
		t.Fatal(err)
	}
	cookie, _ := loginOwnerForTest(t, handler, "owner-test")
	token := qrOwnerTokenForTest(t, handler, cookie)
	destination := "https://external.example/some/path?sig=A%2Fb%2Bc&x=1&x=2#details"
	result := qrEditForTest(t, handler, cookie, "vizitka", destination, token, false)
	if result.Code != http.StatusSeeOther {
		t.Fatalf("save destination: %d %s", result.Code, result.Body.String())
	}
	stats := f.tracker.snapshot()
	if stats.Total != 1 || stats.Leads != 1 || stats.ByCampaign["vizitka"].Scans != 1 || stats.ByCampaign["vizitka"].Leads != 1 {
		t.Fatalf("editing changed counts: %+v", stats)
	}
	restored := newQRTracker(f.tracker.path)
	if err := restored.load(); err != nil {
		t.Fatal(err)
	}
	if row := restored.snapshot().ByCampaign["vizitka"]; row.Destination != destination || row.AddUTM || row.UpdatedAt == "" || row.Scans != 1 || row.Leads != 1 {
		t.Fatalf("saved destination lost: %+v", row)
	}
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "https://qoncept.example/qr?campaign=vizitka&destination=https://attacker.example", nil))
	if result.Code != http.StatusFound || result.Header().Get("Location") != destination {
		t.Fatalf("printed QR URL did not preserve signed destination: %d %q", result.Code, result.Header().Get("Location"))
	}
	if f.tracker.snapshot().ByCampaign["vizitka"].Scans != 2 {
		t.Fatal("external redirect was not counted")
	}
	updated := "https://another.example/landing#contact"
	result = qrEditForTest(t, handler, cookie, "vizitka", updated, token, false)
	if result.Code != http.StatusSeeOther {
		t.Fatal("cannot edit existing destination")
	}
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "https://qoncept.example/qr?campaign=vizitka", nil))
	if result.Header().Get("Location") != updated || f.tracker.snapshot().ByCampaign["vizitka"].Scans != 3 {
		t.Fatal("old printed code did not follow edited destination")
	}
	r := httptest.NewRequest(http.MethodGet, "https://qoncept.example/site/qr/code.png?campaign=vizitka&download=1", nil)
	r.AddCookie(cookie)
	result = httptest.NewRecorder()
	handler.ServeHTTP(result, r)
	image, err := png.DecodeConfig(result.Body)
	if result.Code != http.StatusOK || err != nil || image.Width != 768 || image.Height != 768 || result.Header().Get("Content-Disposition") == "" {
		t.Fatal("download PNG broke after destination editing")
	}
}

func TestQRDestinationAuthCSRFAndValidation(t *testing.T) {
	f := defaultFeatures()
	f.adminPassword = "owner-test"
	handler, err := newHandler("https://qoncept.example", f)
	if err != nil {
		t.Fatal(err)
	}
	if result := qrEditForTest(t, handler, nil, "poster", "https://external.example", "", false); result.Code != http.StatusSeeOther || !strings.Contains(result.Header().Get("Location"), "/login") {
		t.Fatal("unauthenticated destination change accepted")
	}
	cookie, _ := loginOwnerForTest(t, handler, "owner-test")
	token := qrOwnerTokenForTest(t, handler, cookie)
	for _, invalidToken := range []string{"", strings.Repeat("x", 43)} {
		if result := qrEditForTest(t, handler, cookie, "poster", "https://external.example", invalidToken, false); result.Code != http.StatusForbidden {
			t.Fatal("destination update accepted invalid CSRF")
		}
	}
	for _, destination := range []string{"javascript:alert(1)", "data:text/html,foo", "ftp://example.com", "//example.com", "/", "https://user:password@example.com", "https://example.com/\nfoo", "https://example.com\\@qoncept.by", "https://qoncept.example/qr?campaign=other", "https://qoncept.by/en/qr", "https://qoncept.example/site/qr/login", "https://external.example:99999/"} {
		result := qrEditForTest(t, handler, cookie, "poster", destination, token, false)
		if result.Code != http.StatusUnprocessableEntity {
			t.Errorf("unsafe destination %q: %d", destination, result.Code)
		}
	}
	if len(f.tracker.snapshot().ByCampaign) != 0 {
		t.Fatal("invalid requests changed campaign storage")
	}
	r := httptest.NewRequest(http.MethodPost, "https://qoncept.example/site/qr/save", strings.NewReader(url.Values{"campaign": {"poster"}, "destination": {"https://external.example"}, "csrf": {token}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://foreign.example")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatal("foreign-origin edit accepted")
	}
}

func TestQRUTMIsOptionalAndKeepsExistingAttribution(t *testing.T) {
	row := qrCampaign{Destination: "https://external.example/path?utm_source=partner&x=one#details", AddUTM: true}
	target, err := qrDestination(row, "poster", "https://qoncept.by")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(target)
	if u.Query().Get("utm_source") != "partner" || u.Query().Get("utm_campaign") != "poster" || u.Query().Get("x") != "one" || u.Fragment != "details" {
		t.Fatalf("UTM lost original query/fragment: %s", target)
	}
	legacy, err := qrDestination(qrCampaign{}, "old-vizitka", "https://qoncept.by")
	if err != nil || !strings.HasPrefix(legacy, "/?") || !strings.Contains(legacy, "utm_campaign=old-vizitka") {
		t.Fatalf("legacy QR changed: %s %v", legacy, err)
	}
}

func TestQROwnerControlsKeepWorkingInOtherLanguages(t *testing.T) {
	f := defaultFeatures()
	f.adminPassword = "owner-test"
	handler, err := newHandler("https://qoncept.example", f)
	if err != nil {
		t.Fatal(err)
	}
	cookie, _ := loginOwnerForTest(t, handler, "owner-test")
	if err := f.tracker.saveDestination("poster", "https://external.example", false); err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"en", "uk", "be"} {
		r := httptest.NewRequest(http.MethodGet, "https://qoncept.example/site/qr?campaign=poster", nil)
		r.AddCookie(cookie)
		r.AddCookie(&http.Cookie{Name: "qoncept_language", Value: language})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		body := w.Body.String()
		if w.Code != http.StatusOK || !strings.Contains(body, `action="/site/qr/save"`) || !strings.Contains(body, `href="/site/qr?campaign=poster"`) || strings.Contains(body, `action="/`+language+`/site/`) {
			t.Fatalf("owner controls broke in %s", language)
		}
	}
}

func TestQRFailedDestinationSaveKeepsPreviousState(t *testing.T) {
	tracker := newQRTracker("")
	if err := tracker.saveDestination("poster", "https://first.example", false); err != nil {
		t.Fatal(err)
	}
	if err := tracker.record("poster", false); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	tracker.path = filepath.Join(blocked, "qr.json")
	if err := tracker.saveDestination("poster", "https://second.example", true); err == nil {
		t.Fatal("failed storage reported success")
	}
	row := tracker.snapshot().ByCampaign["poster"]
	if row.Destination != "https://first.example" || row.Scans != 1 || row.AddUTM {
		t.Fatalf("failed save changed active destination or counts: %+v", row)
	}
}
