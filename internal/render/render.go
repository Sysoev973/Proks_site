// Package render собирает HTML-страницы сайта из шаблонов в templates/.
package render

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"

	"github.com/Sysoev973/Proks_site/internal/source"
)

// Page — данные, которые получает каждый шаблон.
type Page struct {
	// Base — корень сайта, "/Proks_site/": от него ссылки на static/.
	Base string
	// Home — корень версии сайта на языке Lang: "/Proks_site/" или "/Proks_site/en/". От него ссылки между страницами.
	Home string
	// Lang — язык страницы; его словарь доступен в шаблонах как {{t "ключ"}}.
	Lang Lang
	// Langs — эта же страница на всех языках, для переключателя.
	Langs []Alternate
	// Site — общие для всех страниц сведения: звёзды, коммит Proks, год.
	Site  Site
	Title string
	// Active — раздел для подсветки в меню: "home", "docs", "changelog" или "bench".
	Active string
	// Data — данные конкретной страницы: HomeData, DocsData, ChangelogData или BenchData.
	Data any
}

// NavLink — пункт бокового меню документации.
type NavLink struct {
	Title  string
	URL    string
	Active bool
}

// DocsData — данные шаблона docs.html.
type DocsData struct {
	Nav       []NavLink
	Content   template.HTML
	SourceURL string // документ на GitHub
}

// ReleaseView — релиз с уже отрендеренным описанием.
type ReleaseView struct {
	source.Release
	BodyHTML template.HTML
}

// ChangelogData — данные шаблона changelog.html.
type ChangelogData struct {
	Releases    []ReleaseView
	Commits     []source.Commit
	ReleasesURL string
	CommitsURL  string
}

// Renderer рендерит страницы: общий каркас layout.html плюс шаблон страницы.
type Renderer struct {
	dir string
}

// New создаёт Renderer, который читает шаблоны из каталога dir.
func New(dir string) *Renderer {
	return &Renderer{dir: dir}
}

// Render пишет в w страницу name (например "index.html"), вставленную в layout.html.
// Каждая страница парсится отдельно, потому что все они определяют свой блок "content";
// заодно функция t привязывается к языку страницы.
func (r *Renderer) Render(w io.Writer, name string, p Page) error {
	t, err := template.New(name).Funcs(template.FuncMap{"t": p.Lang.T}).ParseFiles(
		filepath.Join(r.dir, "layout.html"),
		filepath.Join(r.dir, name),
	)
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	if err := t.ExecuteTemplate(w, "layout", p); err != nil {
		return fmt.Errorf("execute %s: %w", name, err)
	}
	return nil
}

// NormalizeBase приводит базовый путь к виду "/a/b/": "Proks_site" и "/Proks_site" дают "/Proks_site/".
func NormalizeBase(base string) string {
	base = strings.Trim(base, "/")
	if base == "" {
		return "/"
	}
	return "/" + base + "/"
}
