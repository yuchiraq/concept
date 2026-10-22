package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

var campaignPattern = regexp.MustCompile(`[^a-z0-9_-]+`)
var minskZone = time.FixedZone("Europe/Minsk", 3*60*60)

func normalizeCampaign(value string) string {
	value = strings.Trim(campaignPattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "-"), "-_")
	if len(value) > 64 {
		value = strings.Trim(value[:64], "-_")
	}
	if value == "" {
		return "main"
	}
	return value
}

type qrCampaign struct {
	Scans       uint64 `json:"scans"`
	Leads       uint64 `json:"leads"`
	Destination string `json:"destination,omitempty"`
	AddUTM      bool   `json:"addUtm,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}
type qrStatistics struct {
	Total      uint64                `json:"total"`
	Leads      uint64                `json:"leads"`
	LastScanAt string                `json:"lastScanAt,omitempty"`
	ByCampaign map[string]qrCampaign `json:"byCampaign"`
	ByDate     map[string]uint64     `json:"byDate"`
}
type qrTracker struct {
	mu    sync.Mutex
	path  string
	stats qrStatistics
	now   func() time.Time
}

func newQRTracker(path string) *qrTracker {
	return &qrTracker{path: path, now: time.Now, stats: qrStatistics{ByCampaign: map[string]qrCampaign{}, ByDate: map[string]uint64{}}}
}
func (t *qrTracker) load() error {
	if t.path == "" {
		return nil
	}
	data, err := os.ReadFile(t.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &t.stats); err != nil {
		return err
	}
	if t.stats.ByCampaign == nil {
		t.stats.ByCampaign = map[string]qrCampaign{}
	}
	if t.stats.ByDate == nil {
		t.stats.ByDate = map[string]uint64{}
	}
	return nil
}
func cloneStats(s qrStatistics) qrStatistics {
	c := s
	c.ByCampaign = map[string]qrCampaign{}
	c.ByDate = map[string]uint64{}
	for k, v := range s.ByCampaign {
		c.ByCampaign[k] = v
	}
	for k, v := range s.ByDate {
		c.ByDate[k] = v
	}
	return c
}
func (t *qrTracker) snapshot() qrStatistics {
	t.mu.Lock()
	defer t.mu.Unlock()
	return cloneStats(t.stats)
}
func (t *qrTracker) record(campaign string, lead bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	next := cloneStats(t.stats)
	campaign = normalizeCampaign(campaign)
	row := next.ByCampaign[campaign]
	if lead {
		next.Leads++
		row.Leads++
	} else {
		now := t.now().In(minskZone)
		next.Total++
		row.Scans++
		next.LastScanAt = now.Format(time.RFC3339)
		next.ByDate[now.Format("2006-01-02")]++
	}
	next.ByCampaign[campaign] = row
	if t.path != "" {
		if err := saveQRStats(t.path, next); err != nil {
			return err
		}
	}
	t.stats = next
	return nil
}
func (t *qrTracker) recordLead(c string) error { return t.record(c, true) }
func saveQRStats(path string, stats qrStatistics) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".qr-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err = file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(temporary, path); err == nil {
		return nil
	}
	if runtime.GOOS != "windows" {
		return err
	}
	if e := os.Remove(path); e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	return os.Rename(temporary, path)
}

type qrCampaignRow struct {
	Name         string
	Scans, Leads uint64
	Destination  string
	EditURL      string
}
type qrDateRow struct {
	Date  string
	Scans uint64
}
type qrDashboard struct {
	LocaleData
	Origin, Campaign, TrackingURL, LastScan string
	Total, Today, Leads                     uint64
	Campaigns                               []qrCampaignRow
	Dates                                   []qrDateRow
	TelegramReady                           bool
	CSRF                                    string
	Destination, Error, Notice              string
	AddUTM                                  bool
	OwnerSession                            bool
}

func (f *siteFeatures) registerQR(mux *http.ServeMux, origin func(*http.Request) string) error {
	if err := f.registerOwnerAuth(mux); err != nil {
		return err
	}
	dashboard, err := localizedTemplates("templates/qr.html")
	if err != nil {
		return err
	}
	mux.HandleFunc("/qr", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", 405)
			return
		}
		campaign := normalizeCampaign(r.URL.Query().Get("campaign"))
		target, err := qrDestination(f.tracker.snapshot().ByCampaign[campaign], campaign, origin(r))
		if err != nil {
			http.Error(w, translate(languageFromRequest(r), "Ссылка назначения недоступна. Свяжитесь с владельцем кода."), http.StatusServiceUnavailable)
			return
		}
		if r.Method == http.MethodGet {
			if f.tracker.record(campaign, false) != nil {
				http.Error(w, translate(languageFromRequest(r), "Не удалось учесть переход. Попробуйте ещё раз."), 503)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, target, http.StatusFound)
	})
	renderDashboard := func(w http.ResponseWriter, r *http.Request, campaign, message string, status int) {
		stats := f.tracker.snapshot()
		campaign = normalizeCampaign(campaign)
		data := qrDashboard{Origin: origin(r), Campaign: campaign, TrackingURL: origin(r) + "/qr?campaign=" + url.QueryEscape(campaign), LastScan: "Переходов пока нет", Total: stats.Total, Leads: stats.Leads, Today: stats.ByDate[f.tracker.now().In(minskZone).Format("2006-01-02")], TelegramReady: f.telegram.ready()}
		lang := languageFromRequest(r)
		data.LocaleData = localeData(lang, "/site/qr?campaign="+url.QueryEscape(campaign), origin(r))
		data.LastScan = translate(lang, data.LastScan)
		data.CSRF = f.qrEditToken(w, r)
		_, data.OwnerSession = f.ownerSessionFor(r)
		data.Destination, data.AddUTM = stats.ByCampaign[campaign].Destination, stats.ByCampaign[campaign].AddUTM
		if data.Destination == "" {
			data.Destination = origin(r) + "/"
		}
		if message != "" {
			data.Error = translate(lang, message)
			data.Destination, data.AddUTM = r.PostForm.Get("destination"), r.PostForm.Get("add_utm") == "1"
		} else if r.URL.Query().Get("saved") == "1" {
			data.Notice = translate(lang, "Ссылка сохранена. Ранее напечатанные коды уже ведут на новый адрес.")
		}
		if when, err := time.Parse(time.RFC3339, stats.LastScanAt); err == nil {
			data.LastScan = when.In(minskZone).Format("02.01.2006 15:04")
		}
		for name, c := range stats.ByCampaign {
			destination := c.Destination
			if destination == "" {
				destination = origin(r) + "/"
			}
			data.Campaigns = append(data.Campaigns, qrCampaignRow{Name: name, Scans: c.Scans, Leads: c.Leads, Destination: destination, EditURL: "/site/qr?campaign=" + url.QueryEscape(name)})
		}
		sort.Slice(data.Campaigns, func(i, j int) bool {
			if data.Campaigns[i].Scans == data.Campaigns[j].Scans {
				return data.Campaigns[i].Name < data.Campaigns[j].Name
			}
			return data.Campaigns[i].Scans > data.Campaigns[j].Scans
		})
		for date, count := range stats.ByDate {
			data.Dates = append(data.Dates, qrDateRow{date, count})
		}
		sort.Slice(data.Dates, func(i, j int) bool { return data.Dates[i].Date > data.Dates[j].Date })
		if len(data.Dates) > 30 {
			data.Dates = data.Dates[:30]
		}
		var output bytes.Buffer
		if err := dashboard[lang].ExecuteTemplate(&output, "qr", data); err != nil {
			http.Error(w, translate(languageFromRequest(r), "Не удалось загрузить статистику"), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.WriteHeader(status)
		w.Write(output.Bytes())
	}
	mux.HandleFunc("/site/qr", f.ownerOnly(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method not allowed", 405)
			return
		}
		renderDashboard(w, r, r.URL.Query().Get("campaign"), "", http.StatusOK)
	}))
	mux.HandleFunc("/site/qr/save", f.ownerOnly(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		if r.ParseForm() != nil {
			http.Error(w, translate(languageFromRequest(r), "Некорректный запрос"), http.StatusBadRequest)
			return
		}
		if !f.validQREdit(r, origin(r)) {
			http.Error(w, translate(languageFromRequest(r), "Обновите страницу и повторите попытку"), http.StatusForbidden)
			return
		}
		campaign := normalizeCampaign(r.PostForm.Get("campaign"))
		if raw := r.PostForm.Get("campaign"); raw != campaign {
			renderDashboard(w, r, campaign, "Метка кампании: до 64 латинских букв, цифр, дефисов и подчёркиваний.", http.StatusUnprocessableEntity)
			return
		}
		destination, err := validateQRDestination(r.PostForm.Get("destination"), origin(r), r.Host)
		if err != nil {
			renderDashboard(w, r, campaign, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if err := f.tracker.saveDestination(campaign, destination, r.PostForm.Get("add_utm") == "1"); err != nil {
			renderDashboard(w, r, campaign, "Не удалось сохранить ссылку. Повторите попытку.", http.StatusServiceUnavailable)
			return
		}
		http.Redirect(w, r, "/site/qr?"+url.Values{"campaign": {campaign}, "saved": {"1"}}.Encode(), http.StatusSeeOther)
	}))
	mux.HandleFunc("/site/qr/code.png", f.ownerOnly(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", 405)
			return
		}
		campaign := normalizeCampaign(r.URL.Query().Get("campaign"))
		trackingURL := origin(r) + "/qr?campaign=" + url.QueryEscape(campaign)
		png, err := qrcode.Encode(trackingURL, qrcode.Medium, 768)
		if err != nil {
			http.Error(w, translate(languageFromRequest(r), "Не удалось создать QR-код"), 500)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		if r.URL.Query().Get("download") == "1" {
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="qoncept-%s.png"`, campaign))
		}
		w.Write(png)
	}))
	return nil
}
