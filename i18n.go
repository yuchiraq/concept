package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

//go:embed locales/messages.json
var translationSource []byte

//go:embed locales/client.json
var clientTranslationSource []byte

var languages = []string{"ru", "en", "uk", "be"}
var languageNames = map[string]string{"ru": "Русский", "en": "English", "uk": "Українська", "be": "Беларуская"}
var languageLocales = map[string]string{"ru": "ru_RU", "en": "en_US", "uk": "uk_UA", "be": "be_BY"}
var translationOnce sync.Once
var translations map[string]map[string]string
var clientTranslationKeys []string
var translationError error

type localeContextKey struct{}
type languageLink struct {
	Code, Name, URL, Locale string
	Current                 bool
}
type LocaleData struct {
	Language, OGLocale    string
	AutoURL               string
	Languages, Alternates []languageLink
	ClientMessages        template.JS
}

func loadTranslations() error {
	translationOnce.Do(func() {
		translationError = json.Unmarshal(translationSource, &translations)
		if translationError == nil {
			translationError = json.Unmarshal(clientTranslationSource, &clientTranslationKeys)
		}
	})
	return translationError
}
func validLanguage(lang string) bool { _, ok := languageNames[lang]; return ok }
func translate(lang, text string) string {
	core := strings.TrimSpace(text)
	if lang != "ru" {
		if value := translations[core][lang]; value != "" {
			return strings.Replace(text, core, value, 1)
		}
	}
	return text
}
func localizedURL(lang, path string) string {
	if lang == "ru" || !validLanguage(lang) || strings.HasPrefix(path, "#") {
		return path
	}
	return "/" + lang + path
}
func splitLanguagePath(path string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	if !validLanguage(parts[0]) {
		return "", path, false
	}
	rest := "/"
	if len(parts) == 2 {
		rest += parts[1]
	}
	return parts[0], rest, true
}
func languageFromRequest(r *http.Request) string {
	if lang, ok := r.Context().Value(localeContextKey{}).(string); ok {
		return lang
	}
	return "ru"
}
func countryLanguage(country string) string {
	switch strings.ToUpper(country) {
	case "BY", "RU":
		return "ru"
	case "UA":
		return "uk"
	case "", "XX", "T1":
		return ""
	default:
		return "en"
	}
}
func (f *siteFeatures) automaticLanguage(r *http.Request) string {
	if cookie, err := r.Cookie("qoncept_language"); err == nil && validLanguage(cookie.Value) {
		return cookie.Value
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if f.trustedProxyHeaders && net.ParseIP(host).IsLoopback() {
		for _, header := range []string{"CF-IPCountry", "X-Country-Code"} {
			if lang := countryLanguage(r.Header.Get(header)); lang != "" {
				return lang
			}
		}
	}
	if hint, err := r.Cookie("qoncept_region"); err == nil {
		switch hint.Value {
		case "BY", "RU":
			return "ru"
		case "UA":
			return "uk"
		}
	}
	type preference struct {
		tag string
		q   float64
	}
	var prefs []preference
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		chunks := strings.Split(strings.TrimSpace(part), ";")
		q := 1.0
		for _, param := range chunks[1:] {
			if strings.HasPrefix(strings.TrimSpace(param), "q=") {
				value, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(param), "q="), 64)
				if err != nil || value < 0 || value > 1 {
					q = 0
				} else {
					q = value
				}
			}
		}
		if q > 0 && chunks[0] != "" && chunks[0] != "*" {
			prefs = append(prefs, preference{strings.ToLower(strings.ReplaceAll(chunks[0], "_", "-")), q})
		}
	}
	sort.SliceStable(prefs, func(i, j int) bool { return prefs[i].q > prefs[j].q })
	for _, pref := range prefs {
		parts := strings.Split(pref.tag, "-")
		// A regional locale takes precedence over its language: be-BY defaults to Russian.
		if len(parts) > 1 {
			for _, part := range parts[1:] {
				if len(part) == 2 {
					return countryLanguage(part)
				}
			}
		}
		switch parts[0] {
		case "ru", "be":
			return "ru"
		case "uk":
			return "uk"
		case "en":
			return "en"
		}
	}
	if len(prefs) > 0 {
		return "en"
	}
	return "ru"
}
func localeData(lang, path, origin string) LocaleData {
	data := LocaleData{Language: lang, OGLocale: languageLocales[lang]}
	data.AutoURL = "/language?" + url.Values{"lang": {"auto"}, "next": {path}}.Encode()
	for _, code := range languages {
		target := localizedURL(code, path)
		data.Alternates = append(data.Alternates, languageLink{Code: code, URL: origin + target, Locale: languageLocales[code], Current: lang == code})
		query := url.Values{"lang": {code}, "next": {path}}
		data.Languages = append(data.Languages, languageLink{Code: code, Name: languageNames[code], URL: "/language?" + query.Encode(), Current: lang == code})
	}
	messages := map[string]string{}
	for _, key := range clientTranslationKeys {
		messages[key] = translate(lang, key)
	}
	encoded, _ := json.Marshal(messages)
	data.ClientMessages = template.JS(encoded)
	return data
}
func localizedTemplates(files ...string) (map[string]*template.Template, error) {
	result := map[string]*template.Template{}
	files = append(files, "templates/language.html")
	for _, code := range languages {
		lang := code
		tmpl, err := template.New("localized").Funcs(template.FuncMap{"asset": assetURL, "t": func(text string) string { return translate(lang, text) }, "url": func(path string) string { return localizedURL(lang, path) }}).ParseFiles(files...)
		if err != nil {
			return nil, err
		}
		result[lang] = tmpl
	}
	return result, nil
}
func localizePage(page PageData, lang string) PageData {
	page.Landing = localizeLanding(page.Landing, lang)
	page.BreadcrumbTitle = translate(lang, page.BreadcrumbTitle)
	page.MetaTitle = translate(lang, page.MetaTitle)
	page.MetaDescription = translate(lang, page.MetaDescription)
	page.Eyebrow = translate(lang, page.Eyebrow)
	page.HeroTitle = translate(lang, page.HeroTitle)
	page.HeroText = translate(lang, page.HeroText)
	page.Projects = append([]Project(nil), page.Projects...)
	for i := range page.Projects {
		p := &page.Projects[i]
		p.Name = translate(lang, p.Name)
		p.Category = translate(lang, p.Category)
		p.Description = translate(lang, p.Description)
		p.CTALabel = translate(lang, p.CTALabel)
		p.Tags = append([]string(nil), p.Tags...)
		p.Details = append([]string(nil), p.Details...)
		for j := range p.Tags {
			p.Tags[j] = translate(lang, p.Tags[j])
		}
		for j := range p.Details {
			p.Details[j] = translate(lang, p.Details[j])
		}
	}
	page.Artworks = append([]Artwork(nil), page.Artworks...)
	for i := range page.Artworks {
		page.Artworks[i].Title = translate(lang, page.Artworks[i].Title)
	}
	return page
}
func (f *siteFeatures) languageMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") || r.URL.Path == "/healthz" || r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/language" {
			w.Header().Set("X-Robots-Tag", "noindex, nofollow")
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", "GET")
				http.Error(w, "Method not allowed", 405)
				return
			}
			lang := r.URL.Query().Get("lang")
			if !validLanguage(lang) && lang != "auto" {
				http.Error(w, "Unsupported language", 400)
				return
			}
			path := r.URL.Query().Get("next")
			target, err := url.Parse(path)
			if err != nil || target.IsAbs() || target.Host != "" || !strings.HasPrefix(target.Path, "/") || strings.HasPrefix(target.Path, "//") || strings.ContainsAny(target.Path, "\\\r\n") {
				path = "/"
			} else {
				path = target.EscapedPath()
				if target.RawQuery != "" {
					path += "?" + target.RawQuery
				}
			}
			if _, rest, ok := splitLanguagePath(path); ok {
				path = rest
			}
			if lang != "auto" && !strings.HasPrefix(path, "/site/") {
				path = localizedURL(lang, path)
			}
			cookie := &http.Cookie{Name: "qoncept_language", Value: lang, Path: "/", MaxAge: 365 * 24 * 3600, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: f.secureOwnerCookie(r)}
			if lang == "auto" {
				cookie.Value = ""
				cookie.MaxAge = -1
				if _, err := r.Cookie("qoncept_region"); err == nil {
					http.SetCookie(w, &http.Cookie{Name: "qoncept_region", Value: "", Path: "/", MaxAge: -1, SameSite: http.SameSiteLaxMode, Secure: cookie.Secure})
				}
			}
			http.SetCookie(w, cookie)
			if _, err := r.Cookie("qoncept_region_probe"); err == nil {
				http.SetCookie(w, &http.Cookie{Name: "qoncept_region_probe", Path: "/", MaxAge: -1, SameSite: http.SameSiteLaxMode, Secure: cookie.Secure})
			}
			w.Header().Set("Cache-Control", "no-store")
			http.Redirect(w, r, path, http.StatusSeeOther)
			return
		}
		lang, path, explicit := splitLanguagePath(r.URL.Path)
		if !explicit {
			lang = f.automaticLanguage(r)
			w.Header().Set("Cache-Control", "private, no-store")
		}
		public := publicPagePath(r.URL.Path)
		if public && !explicit && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			manual, manualErr := r.Cookie("qoncept_language")
			_, hintErr := r.Cookie("qoncept_region")
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			trustedCountry := f.trustedProxyHeaders && net.ParseIP(host).IsLoopback() && (countryLanguage(r.Header.Get("CF-IPCountry")) != "" || countryLanguage(r.Header.Get("X-Country-Code")) != "")
			if (manualErr != nil || !validLanguage(manual.Value)) && hintErr != nil && !trustedCountry {
				http.SetCookie(w, &http.Cookie{Name: "qoncept_region_probe", Value: url.QueryEscape(localizedURL(lang, r.URL.Path)), Path: "/", MaxAge: 60, SameSite: http.SameSiteLaxMode, Secure: f.secureOwnerCookie(r)})
			}
		}
		if public && !explicit && lang != "ru" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			target := localizedURL(lang, r.URL.Path)
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			w.Header().Add("Vary", "Accept-Language")
			w.Header().Add("Vary", "Cookie")
			w.Header().Set("Cache-Control", "private, no-store")
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
		if explicit {
			if lang == "ru" {
				target := path
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusPermanentRedirect)
				return
			}
			// Locale prefixes belong only to public pages, never API or owner routes.
			if strings.HasPrefix(path, "/static/") || strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/site/") || path == "/qr" || path == "/language" {
				http.NotFound(w, r)
				return
			}
			clone := r.Clone(r.Context())
			copyURL := *r.URL
			clone.URL = &copyURL
			clone.URL.Path = path
			clone.URL.RawPath = ""
			r = clone
		}
		r = r.WithContext(context.WithValue(r.Context(), localeContextKey{}, lang))
		w.Header().Set("Content-Language", lang)
		w.Header().Add("Vary", "Accept-Language")
		w.Header().Add("Vary", "Cookie")
		next.ServeHTTP(w, r)
	})
}
