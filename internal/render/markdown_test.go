package render

import (
	"strings"
	"testing"
)

var testLinks = Links{
	Base:   "/Proks_site/",
	Repo:   "Sysoev973/Proks",
	Branch: "main",
	Pages: map[string]string{
		"README.md": "docs/",
		"docs/a.md": "docs/a/",
		"docs/b.md": "docs/b/",
	},
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name, from, dest string
		image            bool
		want             string
	}{
		{"документ → документ", "docs/a.md", "b.md", false, "/Proks_site/docs/b/"},
		{"документ с якорем", "docs/a.md", "./b.md#setup", false, "/Proks_site/docs/b/#setup"},
		{"README → docs", "README.md", "docs/a.md", false, "/Proks_site/docs/a/"},
		{"docs → README", "docs/a.md", "../README.md", false, "/Proks_site/docs/"},
		{"от корня репозитория", "docs/a.md", "/docs/b.md", false, "/Proks_site/docs/b/"},
		{"файл кода → GitHub", "README.md", "internal/cache/cache.go", false,
			"https://github.com/Sysoev973/Proks/blob/main/internal/cache/cache.go"},
		{"файл кода из docs", "docs/a.md", "../k6/mixed.js", false,
			"https://github.com/Sysoev973/Proks/blob/main/k6/mixed.js"},
		{"каталог", "README.md", "./", false, "https://github.com/Sysoev973/Proks/tree/main"},
		{"картинка → raw", "docs/a.md", "img/arch.png", true,
			"https://raw.githubusercontent.com/Sysoev973/Proks/main/docs/img/arch.png"},
		{"картинка с пробелом", "README.md", "img/my%20arch.png", true,
			"https://raw.githubusercontent.com/Sysoev973/Proks/main/img/my%20arch.png"},
		{"картинка .md не страница", "README.md", "docs/a.md", true,
			"https://raw.githubusercontent.com/Sysoev973/Proks/main/docs/a.md"},
		{"абсолютная", "README.md", "https://k6.io/docs", false, "https://k6.io/docs"},
		{"mailto", "README.md", "mailto:a@b.c", false, "mailto:a@b.c"},
		{"якорь", "README.md", "#install", false, "#install"},
		{"за пределы репозитория", "README.md", "../other/x.md", false, "../other/x.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testLinks.Resolve(tt.from, tt.dest, tt.image); got != tt.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", tt.from, tt.dest, got, tt.want)
			}
		})
	}
}

func TestMarkdown(t *testing.T) {
	src := []byte(`# Заголовок

## Установка и запуск (Windows)

## Установка и запуск (Windows)

Смотри [второй документ](b.md) и [код](../main.go).

![схема](img/arch.png)

| a | b |
|---|---|
| 1 | 2 |

<script>alert(1)</script>
<p align="center">центр</p>
`)
	got, err := Markdown(src, "docs/a.md", testLinks)
	if err != nil {
		t.Fatal(err)
	}
	html := string(got)

	for _, want := range []string{
		`<h1 id="заголовок">`,
		`<h2 id="установка-и-запуск-windows">`,
		`<h2 id="установка-и-запуск-windows-1">`,
		`href="/Proks_site/docs/b/"`,
		`href="https://github.com/Sysoev973/Proks/blob/main/main.go"`,
		`src="https://raw.githubusercontent.com/Sysoev973/Proks/main/docs/img/arch.png"`,
		`<table class="table table-sm table-bordered">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("нет %q в\n%s", want, html)
		}
	}
	for _, bad := range []string{`<script`, `<p align`} {
		if strings.Contains(html, bad) {
			t.Errorf("сырой HTML %q попал в результат:\n%s", bad, html)
		}
	}
}
