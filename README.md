# Proks_site

Статический сайт о проекте [Proks](https://github.com/Sysoev973/Proks): что это, как устроен, документация и последние изменения.

Сайт: https://sysoev973.github.io/Proks_site/

## Стек

- HTML + Bootstrap 5.3 (CDN)
- Сборщик на Go (`html/template` + goldmark для Markdown)
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

## Публикация

Сайт публикует workflow `.github/workflows/deploy.yml`: клонирует Proks, собирает `dist/` и выкладывает его на GitHub Pages. Он запускается:

- при push в `main` — после мерджа PR;
- по событию `repository_dispatch` с типом `proks-updated` — его присылает Proks, когда меняются `README.md`, `docs/` или выходит релиз;
- кнопкой Run workflow во вкладке Actions;
- раз в сутки по расписанию как страховка. GitHub отключает расписание, если в репозитории 60 дней не было активности: тогда workflow нужно включить заново во вкладке Actions.
