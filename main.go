package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"time"
)

type Project struct {
	ID, Number, Name, Category, Description, URL, Domain, Class string
	Tags                                                        []string
	Details                                                     []string
}
type Artwork struct{ Image, Title, Type string }
type PageData struct {
	MetaTitle, MetaDescription, ActivePath, PageClass, Eyebrow, HeroTitle, HeroText string
	Year                                                                            int
	Projects                                                                        []Project
	Artworks                                                                        []Artwork
}

func projects() []Project {
	return []Project{
		{ID: "avayusstroi", Number: "01", Name: "АВАЮССТРОЙ", Category: "Корпоративный сайт", Description: "Инженерные услуги, которые легко найти. Сайт строительной компании: от первого знакомства до заявки.", URL: "https://avayusstroi.by/", Domain: "avayusstroi.by", Class: "construction", Tags: []string{"Веб-дизайн", "Разработка", "SEO"}, Details: []string{"Структура услуг: автономная канализация, водопонижение, дренаж и аренда техники.", "Страницы услуг, галерея объектов и формы обратной связи с понятным путём к заявке.", "Адаптивная вёрстка, метаданные и структурированная разметка для поисковых систем."}},
		{ID: "workservice", Number: "02", Name: "WorkService", Category: "Веб-приложение", Description: "Рабочие процессы в одном интерфейсе. Внутренний сервис для сотрудников, объектов, расписания и табелей.", URL: "https://appworks.avayusstroi.by/", Domain: "appworks.avayusstroi.by", Class: "workservice", Tags: []string{"UI / UX", "Go", "Веб-приложение"}, Details: []string{"Управление сотрудниками и объектами, назначения по дням и рабочим сменам.", "Табель часов, профили, роли пользователей и закрытые разделы сервиса.", "Серверная разработка на Go, интерфейс для работы с компьютера и телефона. Доступ к рабочим данным требует авторизации."}},
	}
}
func artworks() []Artwork {
	return []Artwork{
		{"cover.jpg", "OGBIGTOY", "3D / Cover art"}, {"done.jpg", "Бюст / 3D-этюд", "3D / Personal artwork"}, {"trap-no-hook.jpg", "Trap No Hook", "Music / Cover art"}, {"4.jpg", "Cover study / 04", "Music / Visual design"}, {"2.jpg", "Cover study / 05", "Music / Cover art"}, {"55kr6392VCk.jpg", "Cover study / 06", "Music / Cover art"}, {"breakingPackZayobal.jpg", "Breaking Pack", "Music / Cover art"}, {"cocaina.jpg", "PLINOFFICIAL — Cocaina 80%", "Music / Cover art"},
	}
}
func main() {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	pages := []PageData{
		{ActivePath: "/", PageClass: "page-home", MetaTitle: "by Qoncept — дизайнер и разработчик", MetaDescription: "Сайты, веб-приложения и музыкальные обложки. Авторское портфолио by Qoncept / Chiraq Concept. От визуальной идеи до работающего продукта."},
		{ActivePath: "/websites", PageClass: "page-web", MetaTitle: "Сайты и веб-приложения — by Qoncept", MetaDescription: "Разработка и дизайн сайтов. Реальные проекты: АВАЮССТРОЙ и WorkService. Лендинги, корпоративные сайты, интерфейсы и сервисы на Go.", Eyebrow: "Web design & development", HeroTitle: "Дизайн, который работает. Код, который решает.", HeroText: "Создаю сайты для бизнеса и инструменты для ежедневной работы. Продумываю структуру, рисую интерфейс и довожу проект до запуска."},
		{ActivePath: "/design", PageClass: "page-design", MetaTitle: "Дизайн и обложки Chiraq Concept — by Qoncept", MetaDescription: "Обложки, 3D-арт и графический дизайн Chiraq Concept. Авторские работы, визуальная концепция релиза и графика для соцсетей.", Eyebrow: "Chiraq Concept / Visual design", HeroTitle: "Звук начинается с обложки.", HeroText: "Создаю визуальный мир вокруг музыки: обложки, 3D-арт и графику для релиза. От первого настроения до финального кадра."},
	}
	for _, p := range pages {
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
		tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/partials.html", file))
		mux.HandleFunc(page.ActivePath, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != page.ActivePath {
				http.NotFound(w, r)
				return
			}
			var output bytes.Buffer
			if err := tmpl.ExecuteTemplate(&output, name, page); err != nil {
				log.Printf("render: %v", err)
				http.Error(w, "Не удалось загрузить страницу", 500)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(output.Bytes())
		})
	}
	server := &http.Server{Addr: ":8089", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Println("Portfolio: http://localhost:8089")
	log.Fatal(server.ListenAndServe())
}
