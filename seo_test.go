package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"regexp"
	"strings"
	"testing"
)

func TestServiceSEOAndTranslatedContent(t *testing.T) {
	handler := testHandler(t, productionOrigin)
	jsonLDPattern := regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`)
	for _, page := range pages() {
		if page.ActivePath == "/" {
			continue
		}
		for _, lang := range languages {
			address := productionOrigin + localizedURL(lang, page.ActivePath)
			body := getPage(t, handler, address).Body.String()
			var data struct {
				Graph []map[string]any `json:"@graph"`
			}
			match := jsonLDPattern.FindStringSubmatch(body)
			if len(match) != 2 {
				t.Fatalf("%s lacks JSON-LD", address)
			}
			if err := json.Unmarshal([]byte(match[1]), &data); err != nil {
				t.Fatal(err)
			}
			var service, breadcrumb map[string]any
			for _, entry := range data.Graph {
				switch entry["@type"] {
				case "Service":
					service = entry
				case "BreadcrumbList":
					breadcrumb = entry
				case "LocalBusiness", "Review", "AggregateRating", "FAQPage":
					t.Fatalf("unexpected schema %v", entry["@type"])
				}
			}
			if service == nil || service["url"] != address || service["description"] != translate(lang, page.MetaDescription) {
				t.Fatalf("%s service differs from visible page", address)
			}
			if breadcrumb == nil || !strings.Contains(body, `class="breadcrumbs"`) {
				t.Fatalf("%s needs matching visible breadcrumbs", address)
			}
			if data.Graph[2]["inLanguage"] != lang {
				t.Fatal("schema language incorrect")
			}
			if page.Landing != nil {
				if !strings.Contains(body, translate(lang, page.Landing.IntroTitle)) || !strings.Contains(body, translate(lang, page.Landing.FAQ[0].Question)) {
					t.Fatalf("%s landing copy missing", address)
				}
			}
			for _, link := range serviceLinks(lang, page.ActivePath) {
				if !strings.Contains(body, `href="`+link.URL+`"`) {
					t.Fatalf("%s missing service link %s", address, link.URL)
				}
			}
		}
	}
}

func TestSitemapLanguageAlternates(t *testing.T) {
	handler := testHandler(t, productionOrigin)
	w := getPage(t, handler, productionOrigin+"/sitemap.xml")
	decoder := xml.NewDecoder(bytes.NewReader(w.Body.Bytes()))
	locations, alternates := 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "loc" {
			locations++
		}
		if start.Name.Local != "link" {
			continue
		}
		if start.Name.Space != "http://www.w3.org/1999/xhtml" {
			t.Fatalf("alternate missing XHTML namespace: %v", start.Name)
		}
		attributes := map[string]string{}
		for _, attr := range start.Attr {
			attributes[attr.Name.Local] = attr.Value
		}
		if attributes["rel"] != "alternate" || attributes["hreflang"] == "" || !strings.HasPrefix(attributes["href"], productionOrigin+"/") {
			t.Fatalf("invalid alternate: %v", attributes)
		}
		alternates++
	}
	if locations != 24 || alternates != locations*5 {
		t.Fatalf("sitemap locations=%d alternates=%d", locations, alternates)
	}
}
