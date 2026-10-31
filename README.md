# Proks_site

Статический сайт о проекте [Proks](https://github.com/Sysoev973/Proks): что это, как устроен, документация и последние изменения.

Сайт: https://sysoev973.github.io/Proks_site/

## Стек

- HTML + Bootstrap 5.3 (CDN)
- Сборщик на Go (`html/template`, позже goldmark для Markdown)
- GitHub Actions + GitHub Pages

## Сборка

Нужен Go 1.27+. Шаблоны лежат в `templates/`, стили в `static/`, готовый сайт собирается в `dist/` (в git не коммитится).

Документация берётся из копии репозитория Proks в `_proks/` (тоже не коммитится): `README.md` → `/docs/`, `docs/x.md` → `/docs/x/`. Релизы и последние коммиты для страницы «Изменения» сборщик получает через GitHub API; если задана переменная `GITHUB_TOKEN`, запросы идут с ней (без токена — 60 запросов в час).

```powershell
git clone --depth 1 https://github.com/Sysoev973/Proks _proks   # один раз; обновить: git -C _proks pull
go run ./cmd/build -base /Proks_site/               # собрать в dist/
go run ./cmd/build -base /Proks_site/ -serve :8080  # собрать и открыть http://localhost:8080/Proks_site/
go test ./...
```

Сайт живёт в подпапке `/Proks_site/`, поэтому все ссылки строятся от `-base`. В Git Bash путь `/Proks_site/` подменяется на путь Windows, там запускайте с `MSYS_NO_PATHCONV=1` или из PowerShell.

`index.html` в корне репозитория — временная копия страницы для публикации из ветки, её уберём, когда сайт начнёт публиковаться из Actions.
