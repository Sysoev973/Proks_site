package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Sysoev973/Proks_site/internal/source"
)

// testLangs читает настоящие словари: go test запускается из каталога пакета, поэтому они на два уровня выше.
func testLangs(t *testing.T) (ru, en Lang, all []Lang) {
	t.Helper()
	all, err := LoadLangs("../../content/i18n", []string{"ru", "en"})
	if err != nil {
		t.Fatal(err)
	}
	return all[0], all[1], all
}

// testPage — страница на языке l по пути path, как её собирает cmd/build.
func testPage(l Lang, all []Lang, path, active string, data any) Page {
	return Page{
		Base: "/Proks_site/", Home: "/Proks_site/" + l.Prefix, Lang: l, Langs: Alternates("/Proks_site/", path, all, l),
		Title: "Proks", Active: active, Data: data,
	}
}

func TestRenderIndexUsesBase(t *testing.T) {
	var buf bytes.Buffer
	ru, _, all := testLangs(t)
	// Шаблоны тоже на два уровня выше.
	r := New("../../templates")
	if err := r.Render(&buf, "index.html", testPage(ru, all, "", "home", nil)); err != nil {
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
		// Язык и переключатель на английскую версию той же страницы.
		`<html lang="ru">`,
		`<link rel="alternate" hreflang="en" href="/Proks_site/en/">`,
		`href="/Proks_site/en/" hreflang="en" lang="en"`,
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
	ru, _, all := testLangs(t)
	pages := map[string]Page{
		"docs.html": testPage(ru, all, "docs/", "docs", DocsData{
			Nav:       []NavLink{{Title: "Обзор", URL: "/Proks_site/docs/", Active: true}},
			Content:   "<p>текст документа</p>",
			SourceURL: "https://github.com/Sysoev973/Proks/blob/main/README.md",
		}),
		"changelog.html": testPage(ru, all, "changelog/", "changelog", ChangelogData{
			Releases: []ReleaseView{{Release: source.Release{Name: "v1", Tag: "v1.0.0", Published: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}, BodyHTML: "<p>notes</p>"}},
			Commits:  []source.Commit{{SHA: "0123456789abcdef", Message: "feat: x", Date: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}},
		}),
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

func TestRenderEnglish(t *testing.T) {
	r := New("../../templates")
	_, en, all := testLangs(t)
	var buf bytes.Buffer
	docs := DocsData{Nav: []NavLink{{Title: "Обзор", URL: "/Proks_site/en/docs/", Active: true}}, Content: "<p>текст</p>"}
	if err := r.Render(&buf, "docs.html", testPage(en, all, "docs/", "docs", docs)); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{
		`<html lang="en">`,
		`href="/Proks_site/static/css/site.css"`, // статика общая, не из /en/
		`class="nav-link active" href="/Proks_site/en/docs/"`,
		`href="/Proks_site/docs/" hreflang="ru" lang="ru"`,
		`>Changelog</a>`,
		`written in Russian`, // предупреждение, что документация не переведена
	} {
		if !strings.Contains(html, want) {
			t.Errorf("нет %q", want)
		}
	}
	if strings.Contains(html, "Документация</a>") {
		t.Error("в английской версии осталось русское меню")
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
