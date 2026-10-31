package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Sysoev973/Proks_site/internal/source"
)

func TestRenderIndexUsesBase(t *testing.T) {
	var buf bytes.Buffer
	// go test запускается из каталога пакета, поэтому шаблоны лежат на два уровня выше.
	r := New("../../templates")
	if err := r.Render(&buf, "index.html", Page{Base: "/Proks_site/", Title: "Proks"}); err != nil {
		t.Fatal(err)
	}
	html := buf.String()

	for _, want := range []string{
		`<title>Proks</title>`,
		`href="/Proks_site/static/css/site.css"`,
		`href="/Proks_site/#about"`,
		`Что такое Proks`,
		// Переключатель темы: скрипт от базового пути и все три варианта.
		`src="/Proks_site/static/js/theme.js"`,
		`data-theme-value="light"`,
		`data-theme-value="dark"`,
		`data-theme-value="auto"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("в странице нет %q", want)
		}
	}

	// Ссылки от корня домена на GitHub Pages ведут мимо подпапки сайта.
	for _, bad := range []string{`href="/static/`, `src="/static/`} {
		if strings.Contains(html, bad) {
			t.Errorf("в странице есть ссылка от корня домена %q", bad)
		}
	}
}

func TestRenderDocsAndChangelog(t *testing.T) {
	r := New("../../templates")
	pages := map[string]Page{
		"docs.html": {Base: "/Proks_site/", Title: "Docs", Active: "docs", Data: DocsData{
			Nav:       []NavLink{{Title: "Обзор", URL: "/Proks_site/docs/", Active: true}},
			Content:   "<p>текст документа</p>",
			SourceURL: "https://github.com/Sysoev973/Proks/blob/main/README.md",
		}},
		"changelog.html": {Base: "/Proks_site/", Title: "Изменения", Active: "changelog", Data: ChangelogData{
			Releases: []ReleaseView{{Release: source.Release{Name: "v1", Tag: "v1.0.0", Published: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}, BodyHTML: "<p>notes</p>"}},
			Commits:  []source.Commit{{SHA: "0123456789abcdef", Message: "feat: x", Date: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}},
		}},
	}
	wants := map[string][]string{
		"docs.html":      {`<p>текст документа</p>`, `aria-current="page">Обзор</a>`, `class="nav-link active" href="/Proks_site/docs/"`},
		"changelog.html": {`v1.0.0`, `01.10.2026`, `>0123456<`, `feat: x`, `class="nav-link active" href="/Proks_site/changelog/"`},
	}
	for name, p := range pages {
		var buf bytes.Buffer
		if err := r.Render(&buf, name, p); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		html := buf.String()
		for _, want := range wants[name] {
			if !strings.Contains(html, want) {
				t.Errorf("%s: нет %q", name, want)
			}
		}
		if strings.Contains(html, `href="/static/`) || strings.Contains(html, `href="/docs/`) {
			t.Errorf("%s: есть ссылка от корня домена", name)
		}
	}
}

func TestNormalizeBase(t *testing.T) {
	tests := map[string]string{
		"":             "/",
		"/":            "/",
		"Proks_site":   "/Proks_site/",
		"/Proks_site":  "/Proks_site/",
		"/Proks_site/": "/Proks_site/",
		"a/b/":         "/a/b/",
	}
	for in, want := range tests {
		if got := NormalizeBase(in); got != want {
			t.Errorf("NormalizeBase(%q) = %q, want %q", in, got, want)
		}
	}
}
