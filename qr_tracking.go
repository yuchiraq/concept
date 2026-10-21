package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
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
	Scans uint64 `json:"scans"`
	Leads uint64 `json:"leads"`
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
}
type qrDateRow struct {
	Date  string
	Scans uint64
}
type qrDashboard struct {
	Origin, Campaign, TrackingURL, LastScan string
	Total, Today, Leads                     uint64
	Campaigns                               []qrCampaignRow
	Dates                                   []qrDateRow
	TelegramReady                           bool
}

func (f *siteFeatures) registerQR(mux *http.ServeMux, origin func(*http.Request) string) error {
	dashboard, err := template.ParseFiles("templates/qr.html")
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
		if r.Method == http.MethodGet {
			if f.tracker.record(campaign, false) != nil {
				http.Error(w, "Не удалось учесть переход. Попробуйте ещё раз.", 503)
				return
			}
		}
		query := url.Values{"utm_source": {"qr"}, "utm_medium": {"offline"}, "utm_campaign": {campaign}}
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		http.Redirect(w, r, "/?"+query.Encode(), http.StatusFound)
	})
	mux.HandleFunc("/site/qr", f.ownerOnly(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method not allowed", 405)
			return
		}
		stats := f.tracker.snapshot()
		campaign := normalizeCampaign(r.URL.Query().Get("campaign"))
		data := qrDashboard{Origin: origin(r), Campaign: campaign, TrackingURL: origin(r) + "/qr?campaign=" + url.QueryEscape(campaign), LastScan: "Переходов пока нет", Total: stats.Total, Leads: stats.Leads, Today: stats.ByDate[f.tracker.now().In(minskZone).Format("2006-01-02")], TelegramReady: f.telegram.ready()}
		if when, err := time.Parse(time.RFC3339, stats.LastScanAt); err == nil {
			data.LastScan = when.In(minskZone).Format("02.01.2006 15:04")
		}
		for name, c := range stats.ByCampaign {
			data.Campaigns = append(data.Campaigns, qrCampaignRow{name, c.Scans, c.Leads})
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
		if err := dashboard.ExecuteTemplate(&output, "qr", data); err != nil {
			http.Error(w, "Не удалось загрузить статистику", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(output.Bytes())
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
			http.Error(w, "Не удалось создать QR-код", 500)
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
