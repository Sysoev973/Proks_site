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

func build(ctx context.Context, c config, gh *source.GitHub) error {
	// Сначала собираем все данные: если GitHub API недоступен, старый dist/ останется целым.
	docs, err := source.LoadDocs(c.proks)
	if err != nil {
		return fmt.Errorf("%w (нужна копия Proks: git clone --depth 1 https://github.com/%s %s)", err, c.repo, c.proks)
	}
	releases, err := gh.Releases(ctx, c.repo, 10)
	if err != nil {
		return err
	}
	commits, err := gh.Commits(ctx, c.repo, c.branch, 10)
	if err != nil {
		return err
	}
	bench, err := gh.BenchHistory(ctx, c.repo, c.benchBranch)
	if err != nil {
		return err
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
		if err := buildLang(r, c, l, langs, docs, releases, commits, bench); err != nil {
			return fmt.Errorf("%s: %w", l.Code, err)
		}
	}

	return os.CopyFS(filepath.Join(c.out, "static"), os.DirFS(c.static))
}

// buildLang пишет все страницы сайта на языке l в <out>/<l.Prefix>.
// Тексты из Proks (документация, релизы, коммиты) не переводятся, меняется только интерфейс вокруг них.
func buildLang(r *render.Renderer, c config, l render.Lang, langs []render.Lang,
	docs []source.Doc, releases []source.Release, commits []source.Commit, bench []source.BenchRun) error {
	home := c.base + l.Prefix
	page := func(path, title, active string, data any) render.Page {
		return render.Page{
			Base: c.base, Home: home, Lang: l, Langs: render.Alternates(c.base, path, langs, l),
			Title: title, Active: active, Data: data,
		}
	}
	write := func(path, name string, p render.Page) error {
		return writePage(r, c.out, l.Prefix+path, name, p)
	}

	// Ссылки между документами ведут на страницы той же языковой версии.
	links := render.Links{Base: home, Repo: c.repo, Branch: c.branch, Pages: map[string]string{}}
	for _, d := range docs {
		links.Pages[d.Path] = docPage(d.Path)
	}

	if err := write("", "index.html", page("", l.Text("title.home"), "home", nil)); err != nil {
		return err
	}

	for _, d := range docs {
		content, err := render.Markdown(d.Markdown, d.Path, links)
		if err != nil {
			return fmt.Errorf("markdown %s: %w", d.Path, err)
		}
		nav := make([]render.NavLink, len(docs))
		for i, o := range docs {
			nav[i] = render.NavLink{Title: o.Title, URL: home + docPage(o.Path), Active: o.Path == d.Path}
		}
		data := render.DocsData{Nav: nav, Content: content, SourceURL: links.SourceURL(d.Path)}
		if err := write(docPage(d.Path), "docs.html", page(docPage(d.Path), d.Title+" — Proks", "docs", data)); err != nil {
			return err
		}
	}

	views := make([]render.ReleaseView, len(releases))
	for i, rel := range releases {
		body, err := render.Markdown([]byte(rel.Body), "", links)
		if err != nil {
			return fmt.Errorf("markdown релиза %s: %w", rel.Tag, err)
		}
		views[i] = render.ReleaseView{Release: rel, BodyHTML: body}
	}
	changelog := render.ChangelogData{
		Releases:    views,
		Commits:     commits,
		ReleasesURL: "https://github.com/" + c.repo + "/releases",
		CommitsURL:  "https://github.com/" + c.repo + "/commits/" + c.branch,
	}
	if err := write("changelog/", "changelog.html", page("changelog/", l.Text("title.changelog"), "changelog", changelog)); err != nil {
		return err
	}

	benchData := render.NewBenchData(bench, c.repo, c.benchBranch, l)
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
