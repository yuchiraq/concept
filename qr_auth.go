package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const ownerCookie = "qoncept_qr_session"
const ownerCSRFCookie = "qoncept_qr_login"
const ownerSessionTTL = 8 * time.Hour

type ownerSession struct {
	Expires      time.Time
	CSRF         string
	PasswordHash [32]byte
}

func ownerToken() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}
func (f *siteFeatures) localOwner(r *http.Request) bool {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return f.adminPassword == "" && f.localAdmin && net.ParseIP(host).IsLoopback()
}
func (f *siteFeatures) ownerSessionFor(r *http.Request) (ownerSession, bool) {
	cookie, err := r.Cookie(ownerCookie)
	if err != nil || len(cookie.Value) != 43 || f.adminPassword == "" {
		return ownerSession{}, false
	}
	key := sha256.Sum256([]byte(cookie.Value))
	f.mu.Lock()
	defer f.mu.Unlock()
	session, ok := f.ownerSessions[key]
	if !ok {
		return ownerSession{}, false
	}
	if !f.now().Before(session.Expires) || session.PasswordHash != sha256.Sum256([]byte(f.adminPassword)) {
		delete(f.ownerSessions, key)
		return ownerSession{}, false
	}
	return session, true
}
func (f *siteFeatures) secureOwnerCookie(r *http.Request) bool {
	return r.TLS != nil || strings.HasPrefix(f.publicOrigin, "https://")
}
func (f *siteFeatures) setOwnerCookie(w http.ResponseWriter, r *http.Request, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/site/qr", MaxAge: age, HttpOnly: true, Secure: f.secureOwnerCookie(r), SameSite: http.SameSiteStrictMode})
}
func ownerNext(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path != "/site/qr" || len(value) > 512 || strings.ContainsAny(value, "\r\n") {
		return "/site/qr"
	}
	return parsed.RequestURI()
}
func (f *siteFeatures) allowOwnerLogin(r *http.Request) bool {
	host := f.clientAddress(r)
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	for key, entry := range f.loginLimits {
		if now.Sub(entry.Start) >= 15*time.Minute {
			delete(f.loginLimits, key)
		}
	}
	entry := f.loginLimits[host]
	if entry.Start.IsZero() {
		entry.Start = now
	}
	if entry.Count >= 8 {
		return false
	}
	entry.Count++
	f.loginLimits[host] = entry
	return true
}
func ownerCSRFValid(r *http.Request) bool {
	cookie, err := r.Cookie(ownerCSRFCookie)
	token := r.FormValue("csrf")
	return err == nil && len(token) == 43 && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(token)) == 1
}
func (f *siteFeatures) registerOwnerAuth(mux *http.ServeMux) error {
	login, err := localizedTemplates("templates/qr_login.html")
	if err != nil {
		return err
	}
	mux.HandleFunc("/site/qr/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next := ownerNext(r.URL.Query().Get("next"))
		if _, ok := f.ownerSessionFor(r); ok || f.localOwner(r) {
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		}
		status, message := http.StatusOK, ""
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			if r.ParseForm() != nil {
				http.Error(w, translate(languageFromRequest(r), "Некорректный запрос"), http.StatusBadRequest)
				return
			}
			next = ownerNext(r.PostForm.Get("next"))
			switch {
			case !ownerCSRFValid(r):
				status, message = http.StatusForbidden, "Форма устарела. Введите пароль ещё раз."
			case f.adminPassword == "":
				status, message = http.StatusServiceUnavailable, "Сначала задайте пароль кабинета в настройках сайта."
			case !f.allowOwnerLogin(r):
				status, message = http.StatusTooManyRequests, "Слишком много попыток. Попробуйте через 15 минут."
				w.Header().Set("Retry-After", "900")
			default:
				got, want := sha256.Sum256([]byte(r.PostForm.Get("password"))), sha256.Sum256([]byte(f.adminPassword))
				if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
					status, message = http.StatusUnauthorized, "Пароль не подошёл. Попробуйте ещё раз."
				} else {
					token, err := ownerToken()
					if err != nil {
						http.Error(w, translate(languageFromRequest(r), "Попробуйте позже"), 500)
						return
					}
					csrf, err := ownerToken()
					if err != nil {
						http.Error(w, translate(languageFromRequest(r), "Попробуйте позже"), 500)
						return
					}
					f.mu.Lock()
					for key, session := range f.ownerSessions {
						if !f.now().Before(session.Expires) {
							delete(f.ownerSessions, key)
						}
					}
					if len(f.ownerSessions) >= 100 {
						f.mu.Unlock()
						http.Error(w, translate(languageFromRequest(r), "Слишком много активных сессий"), 503)
						return
					}
					f.ownerSessions[sha256.Sum256([]byte(token))] = ownerSession{Expires: f.now().Add(ownerSessionTTL), CSRF: csrf, PasswordHash: want}
					delete(f.loginLimits, f.clientAddress(r))
					f.mu.Unlock()
					f.setOwnerCookie(w, r, ownerCookie, token, int(ownerSessionTTL.Seconds()))
					f.setOwnerCookie(w, r, ownerCSRFCookie, "", -1)
					http.Redirect(w, r, next, http.StatusSeeOther)
					return
				}
			}
		}
		csrf, err := ownerToken()
		if err != nil {
			http.Error(w, translate(languageFromRequest(r), "Попробуйте позже"), 500)
			return
		}
		f.setOwnerCookie(w, r, ownerCSRFCookie, csrf, 900)
		var output bytes.Buffer
		data := struct {
			LocaleData
			Next, CSRF, Error string
			Configured        bool
		}{localeData(languageFromRequest(r), "/site/qr/login", requestOrigin(r)), next, csrf, translate(languageFromRequest(r), message), f.adminPassword != ""}
		if err := login[languageFromRequest(r)].ExecuteTemplate(&output, "qr-login", data); err != nil {
			http.Error(w, translate(languageFromRequest(r), "Не удалось загрузить страницу"), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		w.Write(output.Bytes())
	})
	mux.HandleFunc("/site/qr/logout", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "Method not allowed", 405)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		session, ok := f.ownerSessionFor(r)
		if !ok || r.ParseForm() != nil || subtle.ConstantTimeCompare([]byte(session.CSRF), []byte(r.PostForm.Get("csrf"))) != 1 {
			http.Error(w, translate(languageFromRequest(r), "Обновите страницу и повторите попытку"), 403)
			return
		}
		cookie, _ := r.Cookie(ownerCookie)
		f.mu.Lock()
		delete(f.ownerSessions, sha256.Sum256([]byte(cookie.Value)))
		f.mu.Unlock()
		f.setOwnerCookie(w, r, ownerCookie, "", -1)
		http.Redirect(w, r, "/site/qr/login", http.StatusSeeOther)
	})
	return nil
}
