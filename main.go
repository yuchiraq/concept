package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type NavItem struct {
	Label string
	Href  string
}

type HeroStat struct {
	Value string
	Label string
}

type TrackCard struct {
	Href   string
	Label  string
	Title  string
	Text   string
	Accent string
	Chips  []string
}

type Placeholder struct {
	Label string
	Title string
	Text  string
	Class string
}

type InfoCard struct {
	Label string
	Title string
	Text  string
}

type ProcessStep struct {
	Number string
	Title  string
	Text   string
}

type HomePageData struct {
	MetaTitle       string
	MetaDescription string
	ActivePath      string
	PageClass       string
	Nav             []NavItem
	HeroStats       []HeroStat
	Tracks          []TrackCard
	Gallery         []Placeholder
	Process         []ProcessStep
	Year            int
}

type ServicePageData struct {
	MetaTitle       string
	MetaDescription string
	ActivePath      string
	PageClass       string
	Nav             []NavItem
	Eyebrow         string
	HeroTitle       string
	HeroText        string
	DisplayTitle    string
	DisplayClass    string
	HeroStats       []HeroStat
	Capabilities    []InfoCard
	Gallery         []Placeholder
	Process         []ProcessStep
	ClosingTitle    string
	ClosingText     string
	ClosingPoints   []string
	PrimaryHref     string
	PrimaryLabel    string
	SecondaryHref   string
	SecondaryLabel  string
	Year            int
}

func main() {
	templates := map[string]*template.Template{
		"home": template.Must(template.ParseFiles(
			filepath.Join("templates", "base.html"),
			filepath.Join("templates", "home.html"),
		)),
		"service": template.Must(template.ParseFiles(
			filepath.Join("templates", "base.html"),
			filepath.Join("templates", "service.html"),
		)),
	}

	nav := []NavItem{
		{Label: "Главная", Href: "/"},
		{Label: "Сайты", Href: "/websites"},
		{Label: "Дизайн", Href: "/design"},
	}

	homeData := buildHomePage(nav)
	websitesData := buildWebsitesPage(nav)
	designData := buildDesignPage(nav)

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		renderTemplate(w, templates["home"], "home", homeData)
	})
	mux.HandleFunc("/websites", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/websites" {
			http.NotFound(w, r)
			return
		}
		renderTemplate(w, templates["service"], "service", websitesData)
	})
	mux.HandleFunc("/design", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/design" {
			http.NotFound(w, r)
			return
		}
		renderTemplate(w, templates["service"], "service", designData)
	})

	server := &http.Server{
		Addr:              ":8089",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Studio site is live on http://localhost:8089")
	log.Fatal(server.ListenAndServe())
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
		log.Printf("template execute %s: %v", name, err)
	}
}

func buildHomePage(nav []NavItem) HomePageData {
	return HomePageData{
		MetaTitle:       "by Qoncept | Создание сайтов и digital art direction",
		MetaDescription: "Мультистраничный сайт студии by Qoncept: сайты, наружная реклама, соцсети, афиши, музыкальные обложки и сильная digital art подача.",
		ActivePath:      "/",
		PageClass:       "theme-home",
		Nav:             nav,
		HeroStats: []HeroStat{
			{Value: "2", Label: "отдельных направления с собственной подачей"},
			{Value: "dark / light", Label: "полноценная тема под дневной и ночной сценарий"},
			{Value: "8089", Label: "порт запуска Go-приложения"},
		},
		Tracks: []TrackCard{
			{
				Href:   "/websites",
				Label:  "Направление 01",
				Title:  "Создание сайтов",
				Text:   "Лендинги, промо-сайты, портфолио и digital-витрины с сильным первым экраном, понятной структурой и живой подачей бренда.",
				Accent: "accent-web",
				Chips:  []string{"лендинги", "бренд-сайты", "портфолио", "responsive"},
			},
			{
				Href:   "/design",
				Label:  "Направление 02",
				Title:  "Дизайн и визуалы",
				Text:   "Наружка, соцсети, афиши, музыкальные обложки и визуальные системы, которые держатся на одном art direction.",
				Accent: "accent-design",
				Chips:  []string{"наружка", "social packs", "афиши", "cover art"},
			},
		},
		Gallery: []Placeholder{
			{
				Label: "Placeholder 01",
				Title: "Главный кейс сайта",
				Text:  "Сюда позже можно поставить большой скрин hero-секции, интерфейса или финального лендинга.",
				Class: "span-7 tall tone-cyan",
			},
			{
				Label: "Placeholder 02",
				Title: "Серия музыкальных обложек",
				Text:  "Градиентный блок под cover art, release visuals или тизеры релиза.",
				Class: "span-5 tall tone-coral",
			},
			{
				Label: "Placeholder 03",
				Title: "Наружная реклама",
				Text:  "Место для билборда, ситилайта, постера события или промо-кампании.",
				Class: "span-4 short tone-lime",
			},
			{
				Label: "Placeholder 04",
				Title: "Stories и social pack",
				Text:  "Можно заменить на серию stories, post-set или motion-превью для соцсетей.",
				Class: "span-4 short tone-ice",
			},
			{
				Label: "Placeholder 05",
				Title: "Деталь кейса",
				Text:  "Подойдет под крупный фрагмент интерфейса, постера или графическую деталь проекта.",
				Class: "span-4 short tone-sunset",
			},
		},
		Process: []ProcessStep{
			{Number: "01", Title: "Собираем настроение", Text: "Определяем характер проекта, аудиторию и визуальную температуру, чтобы сайт или дизайн сразу попадали в тон."},
			{Number: "02", Title: "Строим систему", Text: "Собираем сетку, типографику, стеклянные слои, акценты и блоки под будущие изображения."},
			{Number: "03", Title: "Готовим к наполнению", Text: "Оставляем удобные слоты под кейсы и картинки, чтобы потом ты просто заменил цветовые блоки на реальные материалы."},
		},
		Year: time.Now().Year(),
	}
}

func buildWebsitesPage(nav []NavItem) ServicePageData {
	return ServicePageData{
		MetaTitle:       "by Qoncept | Создание сайтов",
		MetaDescription: "Отдельная страница про создание сайтов: лендинги, брендовые веб-пространства, промо-сайты и готовые зоны под будущие скриншоты.",
		ActivePath:      "/websites",
		PageClass:       "theme-websites",
		Nav:             nav,
		Eyebrow:         "Создание сайтов",
		HeroTitle:       "Сайты, которые ощущаются как премиальный digital-объект.",
		HeroText:        "Проектируем смелые лендинги, промо-сайты и брендовые страницы, где структура помогает продавать, а визуал цепляет с первого экрана. Все сразу собирается с местом под будущие скриншоты, кейсы и интерфейсные детали.",
		DisplayTitle:    "Блок под будущий главный экран сайта",
		DisplayClass:    "tone-cyan",
		HeroStats: []HeroStat{
			{Value: "7", Label: "дней до первой концепции"},
			{Value: "UX + art", Label: "структура и атмосфера одновременно"},
			{Value: "responsive", Label: "адаптив под desktop и mobile"},
		},
		Capabilities: []InfoCard{
			{Label: "01", Title: "Лендинги и запуск", Text: "Сильный first screen, акцентная композиция, понятный путь к заявке и эффектная подача оффера."},
			{Label: "02", Title: "Брендовые сайты", Text: "Многоуровневые страницы для студий, агентств, продуктов и личных брендов с живой навигацией и характером."},
			{Label: "03", Title: "Портфолио и showcase", Text: "Витрины под кейсы, релизы и продукты, где легко позже заменить цветовые блоки на реальные изображения."},
		},
		Gallery: []Placeholder{
			{Label: "Web mockup 01", Title: "Hero screen / desktop", Text: "Большой плейсхолдер под главный экран страницы, где позже можно показать уникальный стиль проекта.", Class: "span-8 tall tone-cyan"},
			{Label: "Web mockup 02", Title: "Mobile adaptation", Text: "Высокий стеклянный блок для мобильной версии, карточек интерфейса или сетки экранов.", Class: "span-4 tall tone-lime"},
			{Label: "Web mockup 03", Title: "Section with motion", Text: "Под анимированную секцию, ценностный блок, тизер или фрагмент landing-повествования.", Class: "span-6 short tone-ice"},
			{Label: "Web mockup 04", Title: "Case fragment", Text: "Отдельное место под деталь интерфейса, сетку карточек, UI-компоненты или короткий видео-скрин.", Class: "span-6 short tone-sunset"},
			{Label: "Web mockup 05", Title: "Wide strip for screens", Text: "Широкий блок под серию экранов, workflow, схему страницы или набор будущих макетов.", Class: "span-12 short tone-neutral"},
		},
		Process: []ProcessStep{
			{Number: "01", Title: "Архитектура страницы", Text: "Расставляем акценты, порядок блоков и точки взаимодействия, чтобы визуал был не только красивым, но и полезным."},
			{Number: "02", Title: "UI и стеклянные слои", Text: "Собираем glassmorphism-каркас, сетку, крупную типографику, акцентные градиенты и зоны под визуальный контент."},
			{Number: "03", Title: "Подготовка к наполнению", Text: "Оставляем чистые плиты под изображения и скриншоты, чтобы потом ты просто заменил цветовые блоки на реальные материалы."},
		},
		ClosingTitle:   "Удобно, если кейсы и картинки появятся чуть позже.",
		ClosingText:    "Страница уже сейчас выглядит цельно и дорого, а когда у тебя появятся реальные материалы, их можно по очереди вставить в готовые визуальные слоты без переделки структуры.",
		ClosingPoints:  []string{"места под hero-screen и mobile screens", "готовые блоки под кейсы и фрагменты интерфейса", "навигация уже разделяет продукт на понятные сценарии"},
		PrimaryHref:    "mailto:chiraqconcept@icloud.com",
		PrimaryLabel:   "Обсудить сайт",
		SecondaryHref:  "/design",
		SecondaryLabel: "Смотреть дизайн",
		Year:           time.Now().Year(),
	}
}

func buildDesignPage(nav []NavItem) ServicePageData {
	return ServicePageData{
		MetaTitle:       "by Qoncept | Дизайн, наружка, соцсети, обложки",
		MetaDescription: "Отдельная страница про визуальный дизайн: наружная реклама, соцсети, афиши и музыкальные обложки с местами под будущие изображения.",
		ActivePath:      "/design",
		PageClass:       "theme-design",
		Nav:             nav,
		Eyebrow:         "Дизайн и визуальные системы",
		HeroTitle:       "Наружка, соцсети, афиши и музыкальные обложки в одном визуальном коде.",
		HeroText:        "Собираем дизайн так, чтобы бренд одинаково сильно звучал на постере, в stories, на билборде, в релизе и в промо-материалах. Пока это стильные цветовые блоки с glassmorphism-настроением, а позже сюда легко встанут реальные изображения и кейсы.",
		DisplayTitle:    "Плита под будущий key visual кампании",
		DisplayClass:    "tone-coral",
		HeroStats: []HeroStat{
			{Value: "360°", Label: "от digital до физической среды"},
			{Value: "social", Label: "stories, posts, snippets"},
			{Value: "cover", Label: "обложки и релизные визуалы"},
		},
		Capabilities: []InfoCard{
			{Label: "01", Title: "Наружная реклама", Text: "Постеры, билборды, ситилайты, афиши и ключевые носители с выразительной композицией и читаемым месседжем."},
			{Label: "02", Title: "Соцсети и кампании", Text: "Stories, посты, motion-обложки и рекламные наборы, где один стиль сохраняется во всех форматах."},
			{Label: "03", Title: "Музыкальные обложки", Text: "Cover art, релизные тизеры и визуальные системы для артистов, альбомов, EP и синглов."},
		},
		Gallery: []Placeholder{
			{Label: "Design mockup 01", Title: "Билборд / outdoor", Text: "Крупный блок под наружную рекламу, постер или city-format с ярким key visual.", Class: "span-5 tall tone-coral"},
			{Label: "Design mockup 02", Title: "Постерная серия", Text: "Широкая стеклянная панель для серии афиш, обложек или кампаний с типографикой.", Class: "span-7 tall tone-lime"},
			{Label: "Design mockup 03", Title: "Stories pack", Text: "Блок под stories, тизеры релиза, social cards или короткие motion-превью.", Class: "span-4 short tone-ice"},
			{Label: "Design mockup 04", Title: "Single cover", Text: "Плита под музыкальную обложку, digital cover или арт для стримингов.", Class: "span-4 short tone-cyan"},
			{Label: "Design mockup 05", Title: "Event poster", Text: "Подойдет под афишу события, фестиваля, ночного ивента или промо-серии.", Class: "span-4 short tone-sunset"},
			{Label: "Design mockup 06", Title: "Wide visual system", Text: "Широкий блок для будущего набора материалов: наружка, соцсети, афиши и music assets.", Class: "span-12 short tone-neutral"},
		},
		Process: []ProcessStep{
			{Number: "01", Title: "Art direction", Text: "Формируем визуальную температуру проекта: типографика, цвет, ритм, плотность и характер подачи."},
			{Number: "02", Title: "Адаптация в носители", Text: "Собираем систему, которая одинаково уверенно работает в stories, наружке, афишах и обложках."},
			{Number: "03", Title: "Подготовка к наполнению", Text: "Оставляем красивые цветовые блоки-плейсхолдеры, которые потом можно заменить на финальные визуалы без редизайна страницы."},
		},
		ClosingTitle:   "Подойдет, если хочешь постепенно наращивать библиотеку визуалов.",
		ClosingText:    "Сейчас сайт уже выглядит как полноценная креативная студия, а позже ты просто добавишь реальные афиши, обложки, наружку и social-наборы в готовую структуру.",
		ClosingPoints:  []string{"зоны под постеры, билборды и ситилайты", "плейсхолдеры для social pack и motion snippets", "отдельные блоки под музыкальные обложки и release visuals"},
		PrimaryHref:    "mailto:chiraqconcept@icloud.com",
		PrimaryLabel:   "Обсудить дизайн",
		SecondaryHref:  "/websites",
		SecondaryLabel: "Смотреть сайты",
		Year:           time.Now().Year(),
	}
}
