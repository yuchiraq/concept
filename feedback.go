package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type siteSettings struct {
	TrustedProxyHeaders bool `json:"trustedProxyHeaders"`
	Telegram            struct {
		BotToken string   `json:"botToken"`
		ChatIDs  []string `json:"chatIds"`
	} `json:"telegram"`
	QR struct {
		AdminPassword string `json:"adminPassword"`
		StatsPath     string `json:"statsPath"`
	} `json:"qr"`
}

func loadSiteSettings() (siteSettings, error) {
	var cfg siteSettings
	path := os.Getenv("SETTINGS_FILE")
	if path == "" {
		path = "settings.json"
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return cfg, errors.New("cannot read site settings")
	}
	if err == nil {
		if json.Unmarshal(data, &cfg) != nil {
			return cfg, errors.New("invalid settings JSON")
		}
	}
	if value := os.Getenv("TELEGRAM_BOT_TOKEN"); value != "" {
		cfg.Telegram.BotToken = value
	}
	if value := os.Getenv("TELEGRAM_CHAT_IDS"); value != "" {
		cfg.Telegram.ChatIDs = strings.Split(value, ",")
	}
	if value := os.Getenv("QR_ADMIN_PASSWORD"); value != "" {
		cfg.QR.AdminPassword = value
	}
	if value := os.Getenv("QR_STATS_PATH"); value != "" {
		cfg.QR.StatsPath = value
	}
	cfg.Telegram.BotToken = strings.TrimSpace(cfg.Telegram.BotToken)
	chats := []string{}
	for _, chat := range cfg.Telegram.ChatIDs {
		if chat = strings.TrimSpace(chat); chat != "" {
			chats = append(chats, chat)
		}
	}
	cfg.Telegram.ChatIDs = chats
	if cfg.QR.StatsPath == "" {
		cfg.QR.StatsPath = "data/qr_stats.json"
	}
	return cfg, nil
}

type contactSubmission struct {
	Name          string `json:"name"`
	ContactMethod string `json:"contactMethod"`
	Contact       string `json:"contact"`
	Service       string `json:"service"`
	Message       string `json:"message"`
	Page          string `json:"page"`
	Source        string `json:"source"`
	UTMSource     string `json:"utmSource"`
	UTMMedium     string `json:"utmMedium"`
	UTMCampaign   string `json:"utmCampaign"`
	Website       string `json:"website"`
}

var serviceNames = map[string]string{"website": "Сайт", "software": "Сервис и автоматизация", "design": "Дизайн и анимация", "unsure": "Нужна помощь с выбором"}
var methodNames = map[string]string{"telegram": "Telegram", "phone": "Телефон", "email": "Email"}

func validContactPhone(value string) bool {
	compact := strings.NewReplacer(" ", "", "(", "", ")", "", "-", "", ".", "").Replace(value)
	compact = strings.TrimPrefix(compact, "+")
	if len(compact) < 7 || len(compact) > 15 {
		return false
	}
	for _, c := range compact {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (s *contactSubmission) validate() bool {
	s.Name = strings.TrimSpace(s.Name)
	s.Contact = strings.TrimSpace(s.Contact)
	s.Message = strings.TrimSpace(s.Message)
	if utf8.RuneCountInString(s.Name) < 2 || utf8.RuneCountInString(s.Name) > 80 || strings.ContainsAny(s.Name, "\r\n") || strings.ContainsAny(s.Contact, "\r\n") || len(s.Contact) > 200 || utf8.RuneCountInString(s.Message) > 1200 || serviceNames[s.Service] == "" || methodNames[s.ContactMethod] == "" {
		return false
	}
	switch s.ContactMethod {
	case "telegram":
		if validContactPhone(s.Contact) {
			break
		}
		handle := strings.TrimPrefix(s.Contact, "https://t.me/")
		handle = strings.TrimPrefix(handle, "@")
		if len(handle) < 5 || len(handle) > 32 {
			return false
		}
		if !((handle[0] >= 'a' && handle[0] <= 'z') || (handle[0] >= 'A' && handle[0] <= 'Z')) {
			return false
		}
		for _, c := range handle {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				return false
			}
		}
		s.Contact = "@" + handle
	case "phone":
		if !validContactPhone(s.Contact) {
			return false
		}
	case "email":
		at := strings.LastIndex(s.Contact, "@")
		if at < 1 || at >= len(s.Contact)-3 || strings.ContainsAny(s.Contact, " <>\t") || !strings.Contains(s.Contact[at+1:], ".") {
			return false
		}
	}
	if len(s.Page) > 300 || !strings.HasPrefix(s.Page, "/") || strings.HasPrefix(s.Page, "//") || strings.ContainsAny(s.Page, "\r\n") {
		return false
	}
	for _, v := range []string{s.Source, s.UTMSource, s.UTMMedium, s.UTMCampaign} {
		if utf8.RuneCountInString(v) > 120 || strings.ContainsAny(v, "\r\n") {
			return false
		}
	}
	return true
}

type telegramDelivery struct {
	Token   string
	Chats   []string
	Client  *http.Client
	BaseURL string
}

func (t telegramDelivery) ready() bool { return t.Token != "" && len(t.Chats) > 0 }
func (t telegramDelivery) send(ctx context.Context, s contactSubmission) error {
	if !t.ready() {
		return errors.New("telegram unavailable")
	}
	text := "Новая заявка · by Qoncept\n\nИмя: " + s.Name + "\nСвязь: " + methodNames[s.ContactMethod] + " · " + s.Contact + "\nЗадача: " + serviceNames[s.Service]
	if s.Message != "" {
		text += "\n\nО проекте:\n" + s.Message
	}
	text += "\n\nСтраница: " + s.Page
	if s.Source != "" {
		text += "\nКнопка: " + s.Source
	}
	if s.UTMSource != "" {
		text += "\nИсточник: " + s.UTMSource + " / " + s.UTMMedium
	}
	if s.UTMCampaign != "" {
		text += "\nКампания: " + normalizeCampaign(s.UTMCampaign)
	}
	base := t.BaseURL
	if base == "" {
		base = "https://api.telegram.org"
	}
	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	delivered := false
	for _, chat := range t.Chats {
		form := url.Values{"chat_id": {chat}, "text": {text}, "disable_web_page_preview": {"true"}}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/bot"+t.Token+"/sendMessage", strings.NewReader(form.Encode()))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		var result struct {
			OK bool `json:"ok"`
		}
		err = json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&result)
		resp.Body.Close()
		if err == nil && resp.StatusCode == http.StatusOK && result.OK {
			delivered = true
		}
	}
	if !delivered {
		return errors.New("telegram delivery failed")
	}
	return nil
}

type rateWindow struct {
	Start time.Time
	Count int
}
type siteFeatures struct {
	telegram            telegramDelivery
	tracker             *qrTracker
	adminPassword       string
	localAdmin          bool
	publicOrigin        string
	trustedProxyHeaders bool
	mu                  sync.Mutex
	limits              map[string]rateWindow
	loginLimits         map[string]rateWindow
	ownerSessions       map[[32]byte]ownerSession
	now                 func() time.Time
}

func defaultFeatures() *siteFeatures {
	return &siteFeatures{tracker: newQRTracker(""), limits: map[string]rateWindow{}, loginLimits: map[string]rateWindow{}, ownerSessions: map[[32]byte]ownerSession{}, now: time.Now}
}
func (f *siteFeatures) clientAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	// Forwarded addresses are only used when explicitly enabled for a local
	// reverse proxy. They never grant access to the owner dashboard.
	if f.trustedProxyHeaders && net.ParseIP(host).IsLoopback() {
		forwarded := r.Header.Get("X-Real-IP")
		if forwarded == "" {
			parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
			forwarded = strings.TrimSpace(parts[len(parts)-1])
		}
		if ip := net.ParseIP(strings.TrimSpace(forwarded)); ip != nil {
			host = ip.String()
		}
	}
	return host
}
func (f *siteFeatures) allowContact(r *http.Request) bool {
	host := f.clientAddress(r)
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	for key, entry := range f.limits {
		if now.Sub(entry.Start) >= 10*time.Minute {
			delete(f.limits, key)
		}
	}
	entry := f.limits[host]
	if entry.Start.IsZero() {
		entry.Start = now
	}
	if entry.Count >= 5 {
		return false
	}
	entry.Count++
	f.limits[host] = entry
	return true
}
func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func (f *siteFeatures) contactHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		jsonResponse(w, 405, map[string]string{"error": "method_not_allowed"})
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || (u.Host != r.Host && strings.TrimSuffix(origin, "/") != f.publicOrigin) || (u.Scheme != "http" && u.Scheme != "https") {
			jsonResponse(w, 403, map[string]string{"error": "invalid_origin"})
			return
		}
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		jsonResponse(w, 415, map[string]string{"error": "invalid_content_type"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var s contactSubmission
	if decoder.Decode(&s) != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid_request"})
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		jsonResponse(w, 400, map[string]string{"error": "invalid_request"})
		return
	}
	if s.Website != "" {
		jsonResponse(w, 200, map[string]bool{"success": true})
		return
	}
	if !s.validate() {
		jsonResponse(w, 422, map[string]string{"error": "invalid_fields"})
		return
	}
	if !f.telegram.ready() {
		jsonResponse(w, 503, map[string]string{"error": "delivery_unavailable"})
		return
	}
	if !f.allowContact(r) {
		w.Header().Set("Retry-After", "600")
		jsonResponse(w, 429, map[string]string{"error": "rate_limit"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	if f.telegram.send(ctx, s) != nil {
		log.Print("Contact delivery failed; customer data was not logged")
		jsonResponse(w, 502, map[string]string{"error": "delivery_failed"})
		return
	}
	if s.UTMSource == "qr" {
		if err := f.tracker.recordLead(s.UTMCampaign); err != nil {
			log.Print("Could not update QR lead statistics")
		}
	}
	jsonResponse(w, 200, map[string]bool{"success": true})
}
func (f *siteFeatures) ownerOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		if _, ok := f.ownerSessionFor(r); !ok && !f.localOwner(r) {
			nextURL := "/site/qr"
			if r.URL.Path == nextURL && r.URL.RawQuery != "" {
				nextURL += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, "/site/qr/login?next="+url.QueryEscape(nextURL), http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
func (f *siteFeatures) register(mux *http.ServeMux, origin func(*http.Request) string) error {
	mux.HandleFunc("/api/contact", f.contactHandler)
	return f.registerQR(mux, origin)
}
func featuresFromSettings(cfg siteSettings, address string) (*siteFeatures, error) {
	f := defaultFeatures()
	f.telegram = telegramDelivery{Token: cfg.Telegram.BotToken, Chats: cfg.Telegram.ChatIDs}
	f.trustedProxyHeaders = cfg.TrustedProxyHeaders
	f.adminPassword = cfg.QR.AdminPassword
	host, _, _ := net.SplitHostPort(address)
	ip := net.ParseIP(host)
	f.localAdmin = os.Getenv("LOCAL_PREVIEW") == "1" && os.Getenv("SITE_URL") == "" && ip != nil && ip.IsLoopback()
	f.tracker = newQRTracker(cfg.QR.StatsPath)
	if err := f.tracker.load(); err != nil {
		return nil, fmt.Errorf("cannot load QR statistics: %w", err)
	}
	return f, nil
}
