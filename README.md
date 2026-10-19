# by Qoncept

Портфолио дизайнера и разработчика на Go, HTML templates, CSS и JavaScript.

Запуск: `go run .`, затем http://localhost:8089.
Страницы: `/`, `/websites`, `/design`.

## Контент

- `main.go`: описания проектов, ссылки, список обложек и метаданные.
- `templates/partials.html`: повторяющиеся секции и контакты.
- `static/images/covers`: исходные работы Chiraq Concept.
- `static/images/optimized`: веб-версии обложек 1200 × 1200.
- `static/images/projects`: материалы веб-проектов.

В карточках сайтов используются оформленные превью. WorkService показывает демонстрацию структуры без внутренних данных компании. Рабочий сервис требует авторизации.

Референсы: https://getartcraft.com/, https://bigstudio.dev/, https://animata.design/docs/text/counter. Анимации написаны на CSS и JavaScript без новых зависимостей. Учтён prefers-reduced-motion.

Проверка: `go test ./...`, `go build .`, `node --check static/app.js`.
