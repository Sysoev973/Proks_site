// Команда build собирает сайт в каталог dist/ из шаблонов и копии репозитория Proks.
//
//	git clone --depth 1 https://github.com/Sysoev973/Proks _proks
//	go run ./cmd/build -base /Proks_site/
//	go run ./cmd/build -base /Proks_site/ -serve :8080   # и открыть http://localhost:8080/Proks_site/
//
// Токен GitHub берётся из переменной GITHUB_TOKEN, если она задана: без него API даёт 60 запросов в час.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sysoev973/Proks_site/internal/render"
	"github.com/Sysoev973/Proks_site/internal/source"
)

// siteLangs — языки сайта; первый живёт в корне, остальные в подпапках /en/ и т.д.
var siteLangs = []string{"ru", "en"}

type config struct {
	base, out, templates, static string
	proks, repo, branch          string
	benchBranch, i18n            string
}

func main() {
	var c config
	flag.StringVar(&c.base, "base", "/", "базовый путь сайта, например /Proks_site/")
	flag.StringVar(&c.out, "out", "dist", "каталог для готового сайта (очищается перед сборкой)")
	flag.StringVar(&c.templates, "templates", "templates", "каталог с шаблонами")
	flag.StringVar(&c.static, "static", "static", "каталог со статикой, копируется в <out>/static")
	flag.StringVar(&c.proks, "proks", "_proks", "локальная копия репозитория Proks")
	flag.StringVar(&c.repo, "repo", "Sysoev973/Proks", "репозиторий Proks на GitHub (для API и ссылок)")
	flag.StringVar(&c.branch, "branch", "main", "ветка Proks, на файлы которой ведут ссылки")
	flag.StringVar(&c.benchBranch, "bench-branch", "bench-data", "ветка Proks с history.json от bench.yml")
	flag.StringVar(&c.i18n, "i18n", "content/i18n", "каталог со словарями <язык>.json")
	serve := flag.String("serve", "", "после сборки раздавать сайт по этому адресу, например :8080")
	flag.Parse()

	c.base = render.NormalizeBase(c.base)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := build(ctx, c, source.NewGitHub(os.Getenv("GITHUB_TOKEN"))); err != nil {
		log.Fatal(err)
	}
	log.Printf("сайт собран в %s (base %s)", c.out, c.base)

	if *serve != "" {
		log.Fatal(listen(*serve, c.base, c.out))
	}
}

// modules — карточки на главной: пакеты internal/ в Proks. Описания — mod.<имя> в словарях.
// Пакета, которого нет в копии Proks, на сайте не будет: карточки не врут после переименований.
var modules = []render.Module{
	{Name: "proxy", Badge: "CORE", Tags: []string{"http", "singleflight", "swr"}},
	{Name: "cache", Badge: "CORE", Tags: []string{"tinylfu", "lru", "sketch"}},
	{Name: "predictor", Tags: []string{"markov", "ewma"}},
	{Name: "prefetcher", Tags: []string{"prefetch", "workers"}},
	{Name: "events", Badge: "NEW", Tags: []string{"rabbitmq", "outbox", "at-least-once"}},
	{Name: "observability", Tags: []string{"prometheus", "metrics"}},
}

// chips — быстрые переходы с главной. Якоря — id заголовков README.md после goldmark;
// если заголовок в Proks переименуют, чип пропадёт, а сборка напишет об этом в лог.
var chips = []render.Chip{
	{Label: "chip.run", URL: "docs/#план-запуска-и-тестирования-проекта"},
	{Label: "chip.cache", URL: "docs/#политика-кэширования"},
	{Label: "chip.invalidation", URL: "docs/#инвалидация-для-любого-api"},
	{Label: "chip.prefetch", URL: "docs/#prefetch"},
	{Label: "chip.replicas", URL: "docs/#несколько-реплик"},
	{Label: "chip.bench", URL: "bench/"},
	{Label: "chip.ci", URL: "docs/#покрытие-и-ci"},
}

// siteData — всё, что сборщик получил из Proks и GitHub; одинаково для всех языков.
type siteData struct {
	docs     []source.Doc
	releases []source.Release
	commits  []source.Commit
	bench    []source.BenchRun
	project  source.Project
	site     render.Site
}

func build(ctx context.Context, c config, gh *source.GitHub) error {
	// Сначала собираем все данные: если GitHub API недоступен, старый dist/ останется целым.
	var d siteData
	var err error
	d.docs, err = source.LoadDocs(c.proks)
	if err != nil {
		return fmt.Errorf("%w (нужна копия Proks: git clone --depth 1 https://github.com/%s %s)", err, c.repo, c.proks)
	}
	if d.project, err = source.LoadProject(c.proks); err != nil {
		return err
	}
	if d.releases, err = gh.Releases(ctx, c.repo, 10); err != nil {
		return err
	}
	if d.commits, err = gh.Commits(ctx, c.repo, c.branch, 10); err != nil {
		return err
	}
	if d.bench, err = gh.BenchHistory(ctx, c.repo, c.benchBranch); err != nil {
		return err
	}
	repo, err := gh.Repo(ctx, c.repo)
	if err != nil {
		return err
	}
	d.site = render.Site{Repo: c.repo, Stars: repo.Stars, Year: time.Now().Year()}
	if len(d.commits) > 0 {
		d.site.SHA = d.commits[0].SHA
		d.site.CommitURL = d.commits[0].URL
	}
	langs, err := render.LoadLangs(c.i18n, siteLangs)
	if err != nil {
		return err
	}

	// Защита от случайного -out . : каталог удаляется целиком.
	if cl := filepath.Clean(c.out); cl == "." || cl == ".." || cl == filepath.VolumeName(cl)+string(filepath.Separator) {
		return fmt.Errorf("опасный каталог для -out: %q", c.out)
	}
	if err := os.RemoveAll(c.out); err != nil {
		return err
	}

	r := render.New(c.templates)
	for _, l := range langs {
		if err := buildLang(r, c, l, langs, d); err != nil {
			return fmt.Errorf("%s: %w", l.Code, err)
		}
	}

	return os.CopyFS(filepath.Join(c.out, "static"), os.DirFS(c.static))
}

// buildLang пишет все страницы сайта на языке l в <out>/<l.Prefix>.
// Тексты из Proks (документация, релизы, коммиты) не переводятся, меняется только интерфейс вокруг них.
func buildLang(r *render.Renderer, c config, l render.Lang, langs []render.Lang, d siteData) error {
	home := c.base + l.Prefix
	page := func(path, title, active string, data any) render.Page {
		return render.Page{
			Base: c.base, Home: home, Lang: l, Langs: render.Alternates(c.base, path, langs, l),
			Site: d.site, Title: title, Active: active, Data: data,
		}
	}
	write := func(path, name string, p render.Page) error {
		return writePage(r, c.out, l.Prefix+path, name, p)
	}

	// Ссылки между документами ведут на страницы той же языковой версии.
	links := render.Links{Base: home, Repo: c.repo, Branch: c.branch, Pages: map[string]string{}}
	for _, doc := range d.docs {
		links.Pages[doc.Path] = docPage(doc.Path)
	}

	// Документы рендерим заранее: по README проверяем якоря чипов на главной.
	contents := make([]template.HTML, len(d.docs))
	var readme string
	for i, doc := range d.docs {
		content, err := render.Markdown(doc.Markdown, doc.Path, links)
		if err != nil {
			return fmt.Errorf("markdown %s: %w", doc.Path, err)
		}
		contents[i] = content
		if doc.Path == "README.md" {
			readme = string(content)
		}
	}

	homeData := render.HomeData{GoVersion: d.project.GoVersion}
	for _, m := range modules {
		tests, ok := d.project.Tests[m.Name]
		if !ok {
			log.Printf("в %s нет internal/%s, карточка пропущена", c.proks, m.Name)
			continue
		}
		m.Tests = tests
		m.URL = "https://github.com/" + c.repo + "/tree/" + c.branch + "/internal/" + m.Name
		homeData.Modules = append(homeData.Modules, m)
	}
	for _, ch := range chips {
		if _, anchor, ok := strings.Cut(ch.URL, "#"); ok && !strings.Contains(readme, `id="`+anchor+`"`) {
			log.Printf("в README нет заголовка #%s, чип %s пропущен", anchor, ch.Label)
			continue
		}
		ch.URL = home + ch.URL
		homeData.Chips = append(homeData.Chips, ch)
	}
	if err := write("", "index.html", page("", l.Text("title.home"), "home", homeData)); err != nil {
		return err
	}

	for i, doc := range d.docs {
		nav := make([]render.NavLink, len(d.docs))
		for j, o := range d.docs {
			nav[j] = render.NavLink{Title: o.Title, URL: home + docPage(o.Path), Active: o.Path == doc.Path}
		}
		data := render.DocsData{Nav: nav, Content: contents[i], SourceURL: links.SourceURL(doc.Path)}
		if err := write(docPage(doc.Path), "docs.html", page(docPage(doc.Path), doc.Title+" — Proks", "docs", data)); err != nil {
			return err
		}
	}

	views := make([]render.ReleaseView, len(d.releases))
	for i, rel := range d.releases {
		body, err := render.Markdown([]byte(rel.Body), "", links)
		if err != nil {
			return fmt.Errorf("markdown релиза %s: %w", rel.Tag, err)
		}
		views[i] = render.ReleaseView{Release: rel, BodyHTML: body}
	}
	changelog := render.ChangelogData{
		Releases:    views,
		Commits:     d.commits,
		ReleasesURL: "https://github.com/" + c.repo + "/releases",
		CommitsURL:  "https://github.com/" + c.repo + "/commits/" + c.branch,
	}
	if err := write("changelog/", "changelog.html", page("changelog/", l.Text("title.changelog"), "changelog", changelog)); err != nil {
		return err
	}

	benchData := render.NewBenchData(d.bench, c.repo, c.benchBranch, l)
	return write("bench/", "bench.html", page("bench/", l.Text("title.bench"), "bench", benchData))
}

// docPage — путь страницы документа от base: README.md → "docs/", docs/a.md → "docs/a/".
func docPage(p string) string {
	if p == "README.md" {
		return "docs/"
	}
	return "docs/" + strings.ToLower(strings.TrimSuffix(path.Base(p), ".md")) + "/"
}

// writePage рендерит шаблон name в <out>/<dir>index.html.
func writePage(r *render.Renderer, out, dir, name string, p render.Page) error {
	var buf bytes.Buffer
	if err := r.Render(&buf, name, p); err != nil {
		return err
	}
	full := filepath.Join(out, filepath.FromSlash(dir))
	if err := os.MkdirAll(full, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(full, "index.html"), buf.Bytes(), 0o644)
}

// listen раздаёт out по пути base, как это сделает GitHub Pages.
func listen(addr, base, out string) error {
	mux := http.NewServeMux()
	mux.Handle(base, http.StripPrefix(strings.TrimSuffix(base, "/"), http.FileServer(http.Dir(out))))
	if base != "/" {
		mux.Handle("/", http.RedirectHandler(base, http.StatusFound))
	}
	log.Printf("http://localhost%s%s", addr, base)
	return http.ListenAndServe(addr, mux)
}
