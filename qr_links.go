package main

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const qrEditCookie = "qoncept_qr_edit"

// Destinations are owner-controlled, never taken from a public redirect query.
func validateQRDestination(value, origin, requestHost string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > 4096 || strings.Contains(value, "\\") || strings.IndexFunc(value, unicode.IsSpace) >= 0 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", errors.New("Введите полный адрес сайта, начиная с https:// или http://.")
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return "", errors.New("Введите полный адрес сайта, начиная с https:// или http://.")
	}
	if port := u.Port(); port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return "", errors.New("Введите полный адрес сайта, начиная с https:// или http://.")
		}
	}
	public, _ := url.Parse(origin)
	actual, _ := url.Parse("http://" + requestHost)
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	sameService := host == "qoncept.by" || host == "www.qoncept.by" || (public != nil && strings.EqualFold(host, strings.TrimSuffix(public.Hostname(), "."))) || (actual != nil && strings.EqualFold(host, strings.TrimSuffix(actual.Hostname(), ".")))
	if sameService {
		cleaned := path.Clean("/" + strings.TrimPrefix(u.Path, "/"))
		if _, rest, localized := splitLanguagePath(cleaned); localized {
			cleaned = rest
		}
		if cleaned == "/qr" || cleaned == "/language" || cleaned == "/site/qr" || strings.HasPrefix(cleaned, "/site/qr/") {
			return "", errors.New("Укажите страницу назначения, а не адрес QR-перехода или кабинета.")
		}
	}
	return value, nil
}

func (t *qrTracker) saveDestination(campaign, destination string, addUTM bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	next := cloneStats(t.stats)
	campaign = normalizeCampaign(campaign)
	row := next.ByCampaign[campaign]
	row.Destination, row.AddUTM = destination, addUTM
	row.UpdatedAt = t.now().UTC().Format(time.RFC3339)
	next.ByCampaign[campaign] = row
	if t.path != "" {
		if err := saveQRStats(t.path, next); err != nil {
			return err
		}
	}
	t.stats = next
	return nil
}

func qrDestination(row qrCampaign, campaign, origin string) (string, error) {
	destination := row.Destination
	addUTM := row.AddUTM
	if destination == "" {
		destination, addUTM = "/", true
	} else if _, err := validateQRDestination(destination, origin, ""); err != nil {
		return "", err
	}
	if !addUTM {
		// Do not decode/re-encode external signed URLs unless the owner opted in.
		return destination, nil
	}
	u, err := url.Parse(destination)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for key, value := range map[string]string{"utm_source": "qr", "utm_medium": "offline", "utm_campaign": campaign} {
		if !q.Has(key) {
			q.Set(key, value)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (f *siteFeatures) qrEditToken(w http.ResponseWriter, r *http.Request) string {
	if session, ok := f.ownerSessionFor(r); ok {
		return session.CSRF
	}
	if !f.localOwner(r) {
		return ""
	}
	if cookie, err := r.Cookie(qrEditCookie); err == nil && len(cookie.Value) == 43 {
		return cookie.Value
	}
	token, err := ownerToken()
	if err != nil {
		return ""
	}
	f.setOwnerCookie(w, r, qrEditCookie, token, int(ownerSessionTTL.Seconds()))
	return token
}

func (f *siteFeatures) validQREdit(r *http.Request, origin string) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	if source := r.Header.Get("Origin"); source != "" && source != origin && source != requestOrigin(r) {
		return false
	}
	token := r.PostForm.Get("csrf")
	if len(token) != 43 {
		return false
	}
	if session, ok := f.ownerSessionFor(r); ok {
		return subtle.ConstantTimeCompare([]byte(session.CSRF), []byte(token)) == 1
	}
	if f.localOwner(r) {
		cookie, err := r.Cookie(qrEditCookie)
		return err == nil && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(token)) == 1
	}
	return false
}
