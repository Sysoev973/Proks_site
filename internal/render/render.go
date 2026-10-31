// Package render собирает HTML-страницы сайта из шаблонов в templates/.
package render

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
)

// Page — данные, которые получает каждый шаблон.
type Page struct {
	// Base — путь, от которого строятся все ссылки сайта, например "/Proks_site/".
	// Всегда начинается и заканчивается на "/".
	Base  string
	Title string
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
// Каждая страница парсится отдельно, потому что все они определяют свой блок "content".
func (r *Renderer) Render(w io.Writer, name string, p Page) error {
	t, err := template.ParseFiles(
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
