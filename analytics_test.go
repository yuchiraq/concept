package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalyticsAndYandexVerificationInPublicLayout(t *testing.T) {
	handler := testHandler(t, "https://qoncept.example")
	for _, page := range pages() {
		for _, lang := range []string{"ru", "en", "uk", "be"} {
			path := localizedURL(lang, page.ActivePath)
			body := getPage(t, handler, "https://qoncept.example"+path).Body.String()
			for _, required := range []string{`/static/analytics.js`, `data-counter-id="113798986"`, `<meta name="yandex-verification" content="4abc4bee356feb4a">`, `<meta name="referrer" content="strict-origin-when-cross-origin">`} {
				if !strings.Contains(body, required) {
					t.Errorf("%s is missing %s", path, required)
				}
			}
			if got := strings.Count(body, `data-counter-id="113798986"`); got != 1 {
				t.Errorf("%s has %d analytics tags; want 1", path, got)
			}
		}
	}
	for _, path := range []string{"/site/qr", "/site/qr/login", "/api/contact"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "https://qoncept.example"+path, nil))
		body := response.Body.String()
		if strings.Contains(body, "/static/analytics.js") || strings.Contains(body, "mc.yandex.ru") {
			t.Errorf("private endpoint %s includes analytics", path)
		}
	}
}
