package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const contactURL = "https://t.me/yu_cq162"
const productionOrigin = "https://qoncept.by"

type Project struct {
	ID, Number, Name, Category, Description, URL, Domain, Class string
	CTAURL, CTALabel                                            string
	Tags                                                        []string
	Details                                                     []string
}
type Artwork struct{ Image, Title, Type string }
type PageData struct {
	LocaleData
	MetaTitle, MetaDescription, ActivePath, PageClass, Eyebrow, HeroTitle, HeroText string
	CanonicalURL, SiteURL, OGImage                                                  string
	StructuredData                                                                  template.JS
	Year                                                                            int
	Projects                                                                        []Project
	Artworks                                                                        []Artwork
	Landing                                                                         *serviceLanding
	ServiceLinks                                                                    []serviceLink
	ContactService, BreadcrumbTitle                                                 string
}

func projects() []Project {
	return []Project{
		{ID: "avayusstroi", Number: "01", Name: "АВАЮССТРОЙ", Category: "Корпоративный сайт", Description: "Сайт строительной компании: помогает найти нужную услугу, познакомиться с работами и оставить заявку.", URL: "https://avayusstroi.by/", Domain: "avayusstroi.by", CTAURL: "https://avayusstroi.by/", CTALabel: "Открыть сайт", Class: "construction", Tags: []string{"Веб-дизайн", "Разработка", "SEO"}, Details: []string{"Структура услуг: автономная канализация, водопонижение, дренаж и аренда техники.", "Страницы услуг, галерея объектов и формы обратной связи с понятным путём к заявке.", "Адаптивная вёрстка, метаданные и структурированная разметка для поисковых систем."}},
		{ID: "workservice", Number: "02", Name: "WorkService", Category: "Сервис под задачи бизнеса", Description: "Программное решение для ежедневной работы: автоматизация процессов, удобный контроль и меньше бумажной волокиты. Сотрудники, объекты, задачи, учёт топлива и инвентаря в одном интерфейсе.", CTAURL: contactURL, CTALabel: "Обсудить свой сервис", Class: "workservice", Tags: []string{"UI / UX", "Go", "Веб-приложение"}, Details: []string{"Разрабатываю не только сайты, но и полноценные программные решения: автоматизирую учёт, согласования, документы и другие рутинные процессы.", "Удобный контроль сотрудников, задач и объектов, учёт заправок и расхода топлива, инвентаря и выдачи инструментов. Меньше ручных операций и бумажной работы.", "Сервис под ваши правила: вы выбираете, где хранить данные и кому давать доступ. Разработку, размещение и поддержку согласуем заранее."}},
	}
}

func artworks() []Artwork {
	return []Artwork{
		{"cover.jpg", "OGBIGTOY", "3D / Cover art"}, {"done.jpg", "Бюст / 3D-этюд", "3D / Personal artwork"}, {"trap-no-hook.jpg", "Trap No Hook", "Music / Cover art"}, {"4.jpg", "ROZKOSH", "Music / Visual design"}, {"2.jpg", "Yung L.A. — Roll Up Dat Zaza Sht", "Music / Cover art"}, {"55kr6392VCk.jpg", "ZleGlizzy & Lil Gotit — GlizzyGotit", "Music / Cover art"}, {"breakingPackZayobal.jpg", "Breaking Pack", "Music / Cover art"}, {"cocaina.jpg", "PLINOFFICIAL — Cocaina 80%", "Music / Cover art"},
	}
}

func pages() []PageData {
	base := []PageData{
		{ActivePath: "/", PageClass: "page-home", MetaTitle: "Сайты, дизайн и веб-сервисы на заказ — by Qoncept", MetaDescription: "Дизайн и разработка сайтов, сервисов для бизнеса, обложек и наружной рекламы. Реальные работы by Qoncept / Chiraq Concept. Обсудим задачу, сроки и стоимость."},
		{ActivePath: "/websites", PageClass: "page-web", MetaTitle: "Разработка сайтов и сервисов для бизнеса — by Qoncept", MetaDescription: "Сайты и программные решения под задачи бизнеса: автоматизация процессов, удобный контроль и меньше бумажной работы. Дизайн, разработка и запуск собственного сервиса.", Eyebrow: "Сайты и сервисы для бизнеса", HeroTitle: "От сайта до автоматизации бизнеса.", HeroText: "Создаю сайты и полноценные программные решения. Помогаю автоматизировать рутину, наладить удобный контроль и сократить бумажную волокиту. Разберём ваши процессы и разработаем систему, с которой проще работать."},
		{ActivePath: "/design", PageClass: "page-design", MetaTitle: "Обложки, наружная реклама и анимация — Chiraq Concept", MetaDescription: "Дизайн обложек, 3D-арт, наружная реклама и анимация от Chiraq Concept / by Qoncept. Авторские работы и реальные примеры. Обсудим визуал для вашего проекта.", Eyebrow: "Chiraq Concept / Дизайн и анимация", HeroTitle: "Визуал, который привлекает внимание.", HeroText: "Обложки релизов, наружная реклама, 3D и анимация для вашего проекта. Разработаю образ и подготовлю материалы под нужные форматы: от экрана до печати."},
	}
	base[0].ContactService = "unsure"
	base[0].MetaTitle = "Создание сайтов, программы и дизайн на заказ — by Qoncept"
	base[0].MetaDescription = "Сайты для бизнеса, программы на заказ и дизайн. Автоматизация, обложки и макеты рекламы. Работа с клиентами из Беларуси, Бреста и других стран."
	base[1].ContactService, base[1].BreadcrumbTitle = "website", "Создание сайтов"
	base[1].MetaTitle = "Создание сайтов под ключ — дизайн и разработка | by Qoncept"
	base[1].MetaDescription = "Создание лендингов и корпоративных сайтов: дизайн, мобильная версия, формы заявок и запуск. Беларусь, Брест и удалённая работа с другими странами."
	base[1].HeroTitle = "Создание сайтов для бизнеса."
	base[1].HeroText = "Разрабатываю лендинги и корпоративные сайты: структура, дизайн, адаптация для телефона, формы заявок и подготовка к поисковому продвижению. Работаю удалённо с клиентами из разных стран."
	base[2].ContactService, base[2].BreadcrumbTitle = "design", "Дизайн и обложки"
	base[2].MetaTitle = "Дизайн для бизнеса, обложки и наружная реклама — by Qoncept"
	base[2].MetaDescription = "Обложки релизов, 3D, анимация и рекламная графика. Реальные работы Chiraq Concept. Заказы из Беларуси, Бреста и других стран."
	return append(base, servicePages()...)
}

// siteOrigin accepts an origin only, so canonical URLs cannot accidentally include
// credentials, a query string or a second path prefix.
func siteOrigin(value string) (string, error) {
	value = strings.TrimSpace(value)
	u, err := url.Parse(value)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", fmt.Errorf("SITE_URL must be an absolute http(s) origin, for example https://example.com")
	}
	return strings.TrimSuffix(u.Scheme+"://"+u.Host, "/"), nil
}

func requestOrigin(r *http.Request) string {
	if productionHost(r.Host) {
		return productionOrigin
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Do not trust client-supplied forwarding headers. Production deployments
	// behind a proxy should set SITE_URL to their public origin.
	origin, err := siteOrigin(scheme + "://" + r.Host)
	if err != nil {
		return "http://localhost:8089"
	}
	return origin
}

func structuredData(page PageData) (template.JS, error) {
	lang := page.Language
	if !validLanguage(lang) {
		lang = "ru"
	}
	organizationID := page.SiteURL + "/#business"
	websiteID := page.SiteURL + "/#website"
	services := []map[string]any{
		{"@type": "Service", "name": "Дизайн и разработка сайтов", "serviceType": "Лендинги и корпоративные сайты", "url": page.SiteURL + "/websites", "provider": map[string]string{"@id": organizationID}},
		{"@type": "Service", "name": "Разработка программного обеспечения", "serviceType": "Разработка веб-приложений", "url": page.SiteURL + "/software", "provider": map[string]string{"@id": organizationID}},
		{"@type": "Service", "name": "Дизайн для компании", "serviceType": "Графический дизайн и 3D", "url": page.SiteURL + "/business-design", "provider": map[string]string{"@id": organizationID}},
		{"@type": "Service", "name": "Дизайн наружной рекламы", "serviceType": "Дизайн наружной рекламы", "url": page.SiteURL + "/outdoor-advertising", "provider": map[string]string{"@id": organizationID}},
		{"@type": "Service", "name": "Обложки и 3D", "serviceType": "Моушн-дизайн", "url": page.SiteURL + "/design", "provider": map[string]string{"@id": organizationID}},
	}
	for _, service := range services {
		service["name"] = translate(lang, service["name"].(string))
		service["serviceType"] = translate(lang, service["serviceType"].(string))
		service["url"] = page.SiteURL + localizedURL(lang, strings.TrimPrefix(service["url"].(string), page.SiteURL))
		service["@id"] = service["url"].(string) + "#service"
		service["areaServed"] = []any{map[string]string{"@type": "Country", "name": "Belarus"}, map[string]any{"@type": "City", "name": "Brest", "containedInPlace": map[string]string{"@type": "Country", "name": "Belarus"}}, map[string]string{"@type": "Place", "name": "Worldwide"}}
	}

	offers := make([]map[string]any, 0, len(services))
	for _, service := range services {
		offers = append(offers, map[string]any{"@type": "Offer", "itemOffered": service})
	}
	data := map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{"@type": "Organization", "@id": organizationID, "name": "by Qoncept", "alternateName": "Chiraq Concept", "url": page.SiteURL + "/", "image": page.OGImage, "email": "chiraqconcept@icloud.com", "sameAs": []string{contactURL, "https://instagram.com/chiraqconcept162"}, "hasOfferCatalog": map[string]any{"@type": "OfferCatalog", "name": translate(lang, "Дизайн и разработка"), "itemListElement": offers}},
			map[string]any{"@type": "WebSite", "@id": websiteID, "name": "by Qoncept", "url": page.SiteURL + localizedURL(lang, "/"), "inLanguage": lang, "publisher": map[string]string{"@id": organizationID}},
			map[string]any{"@type": "WebPage", "@id": page.CanonicalURL + "#page", "url": page.CanonicalURL, "name": page.MetaTitle, "description": page.MetaDescription, "inLanguage": lang, "isPartOf": map[string]string{"@id": websiteID}, "about": map[string]string{"@id": organizationID}},
		},
	}
	graph := data["@graph"].([]any)
	organization := graph[0].(map[string]any)
	organization["alternateName"] = []string{"Chiraq Concept", "qoncept.by"}
	organization["areaServed"] = []any{map[string]string{"@type": "Country", "name": "Belarus"}, map[string]any{"@type": "City", "name": "Brest", "containedInPlace": map[string]string{"@type": "Country", "name": "Belarus"}}, map[string]string{"@type": "Place", "name": "Worldwide"}}
	organization["contactPoint"] = map[string]any{"@type": "ContactPoint", "contactType": "customer service", "email": "chiraqconcept@icloud.com", "url": contactURL, "availableLanguage": languages}
	webPage := graph[2].(map[string]any)
	for _, service := range services {
		if service["url"] == page.CanonicalURL {
			service["description"] = page.MetaDescription
			webPage["mainEntity"] = map[string]any{"@id": service["@id"]}
			graph = append(graph, service)
			break
		}
	}
	if page.ActivePath != "" && page.ActivePath != "/" {
		label := page.BreadcrumbTitle
		if label == "" {
			label = page.MetaTitle
		}
		breadcrumb := map[string]any{"@type": "BreadcrumbList", "@id": page.CanonicalURL + "#breadcrumbs", "itemListElement": []any{map[string]any{"@type": "ListItem", "position": 1, "name": translate(lang, "Главная"), "item": page.SiteURL + localizedURL(lang, "/")}, map[string]any{"@type": "ListItem", "position": 2, "name": label, "item": page.CanonicalURL}}}
		webPage["breadcrumb"] = map[string]string{"@id": page.CanonicalURL + "#breadcrumbs"}
		graph = append(graph, breadcrumb)
	}
	data["@graph"] = graph
	encoded, err := json.Marshal(data)
	// encoding/json escapes HTML-significant characters before the result is
	// marked as trusted script data for an application/ld+json element.
	return template.JS(encoded), err
}

type sitemapURL struct {
	Location   string             `xml:"loc"`
	Alternates []sitemapAlternate `xml:"xhtml:link"`
}
type sitemapAlternate struct {
	Rel      string `xml:"rel,attr"`
	Language string `xml:"hreflang,attr"`
	URL      string `xml:"href,attr"`
}
type sitemapDocument struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	XHTML   string       `xml:"xmlns:xhtml,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func newHandler(configuredSiteURL string, configuredFeatures ...*siteFeatures) (http.Handler, error) {
	if err := loadTranslations(); err != nil {
		return nil, err
	}
	origin := ""
	if strings.TrimSpace(configuredSiteURL) != "" {
		var err error
		origin, err = siteOrigin(configuredSiteURL)
		if err != nil {
			return nil, err
		}
	}
	resolveOrigin := func(r *http.Request) string {
		if origin != "" {
			return origin
		}
		return requestOrigin(r)
	}
	notFoundTemplates, err := localizedTemplates("templates/base.html", "templates/partials.html", "templates/contact.html", "templates/404.html")
	if err != nil {
		return nil, err
	}
	notFound := func(w http.ResponseWriter, r *http.Request) {
		data := PageData{MetaTitle: "Страница не найдена — by Qoncept", MetaDescription: "Вернитесь на главную и посмотрите проекты by Qoncept.", PageClass: "page-not-found", SiteURL: resolveOrigin(r), CanonicalURL: resolveOrigin(r) + r.URL.EscapedPath(), OGImage: resolveOrigin(r) + "/static/images/covers/cover.jpg", Year: time.Now().Year()}
		lang := languageFromRequest(r)
		data = localizePage(data, lang)
		data.LocaleData = localeData(lang, r.URL.Path, data.SiteURL)
		data.Alternates = nil
		data.CanonicalURL = data.SiteURL + localizedURL(lang, r.URL.EscapedPath())
		var output bytes.Buffer
		if err := notFoundTemplates[lang].ExecuteTemplate(&output, "not-found", data); err != nil {
			http.Error(w, translate(languageFromRequest(r), "Страница не найдена"), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Robots-Tag", "noindex, follow")
		w.WriteHeader(http.StatusNotFound)
		w.Write(output.Bytes())
	}
	mux := http.NewServeMux()
	features := defaultFeatures()
	if len(configuredFeatures) > 0 && configuredFeatures[0] != nil {
		features = configuredFeatures[0]
	}
	features.publicOrigin = origin
	if err := features.register(mux, resolveOrigin); err != nil {
		return nil, err
	}
	mux.Handle("/static/", staticAssets(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("ok"))
	})
	pageList := pages()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /site/\nDisallow: /api/\nDisallow: /qr\nDisallow: /language\n\nSitemap: %s/sitemap.xml\n", resolveOrigin(r))
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		document := sitemapDocument{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9", XHTML: "http://www.w3.org/1999/xhtml"}
		for _, page := range pageList {
			for _, lang := range languages {
				entry := sitemapURL{Location: resolveOrigin(r) + localizedURL(lang, page.ActivePath)}
				for _, alternate := range languages {
					entry.Alternates = append(entry.Alternates, sitemapAlternate{"alternate", alternate, resolveOrigin(r) + localizedURL(alternate, page.ActivePath)})
				}
				entry.Alternates = append(entry.Alternates, sitemapAlternate{"alternate", "x-default", resolveOrigin(r) + page.ActivePath})
				document.URLs = append(document.URLs, entry)
			}
		}
		output, err := xml.Marshal(document)
		if err != nil {
			http.Error(w, translate(languageFromRequest(r), "Не удалось загрузить карту сайта"), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write([]byte(xml.Header))
		w.Write(output)
	})
	for _, p := range pageList {
		page := p
		page.Year = time.Now().Year()
		page.Projects = projects()
		page.Artworks = artworks()
		file := "templates/home.html"
		name := "home"
		if page.ActivePath != "/" {
			file = "templates/service.html"
			name = "service"
		}
		if page.Landing != nil {
			file = "templates/landing.html"
			name = "landing"
			if page.Landing.Case == "software" {
				page.Projects = page.Projects[1:]
			}
		}
		templates, err := localizedTemplates("templates/base.html", "templates/partials.html", "templates/media.html", "templates/contact.html", "templates/demos.html", "templates/seo.html", file)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		mux.HandleFunc(page.ActivePath, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != page.ActivePath {
				notFound(w, r)
				return
			}
			// The stored template data stays immutable across concurrent requests.
			lang := languageFromRequest(r)
			data := localizePage(page, lang)
			data.ServiceLinks = serviceLinks(lang, page.ActivePath)
			data.SiteURL = resolveOrigin(r)
			data.LocaleData = localeData(lang, page.ActivePath, data.SiteURL)
			data.CanonicalURL = data.SiteURL + localizedURL(lang, page.ActivePath)
			data.OGImage = data.SiteURL + "/static/images/covers/cover.jpg"
			var err error
			data.StructuredData, err = structuredData(data)
			if err != nil {
				log.Printf("structured data: %v", err)
				http.Error(w, translate(languageFromRequest(r), "Не удалось загрузить страницу"), http.StatusInternalServerError)
				return
			}
			var output bytes.Buffer
			if err := templates[lang].ExecuteTemplate(&output, name, data); err != nil {
				log.Printf("render: %v", err)
				http.Error(w, translate(languageFromRequest(r), "Не удалось загрузить страницу"), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(output.Bytes())
		})
	}
	return compressResponses(features.languageMiddleware(mux)), nil
}

func main() {
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8089"
	}
	settings, err := loadSiteSettings()
	if err != nil {
		log.Fatal(err)
	}
	settings.SiteURL = publicSiteURL(settings.SiteURL, address)
	features, err := featuresFromSettings(settings, address)
	if err != nil {
		log.Fatal(err)
	}
	handler, err := newHandler(settings.SiteURL, features)
	if err != nil {
		log.Fatal(err)
	}
	if !features.telegram.ready() {
		log.Print("Telegram contact delivery is not configured; fill settings.json")
	}
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("Portfolio listening on %s", address)
	log.Fatal(server.ListenAndServe())
}
