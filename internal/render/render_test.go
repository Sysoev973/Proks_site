package render

import (
	"bytes"
	"strings"
	"testing"
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
