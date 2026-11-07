package main

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sysoev973/Proks_site/internal/render"
)

// urlset — минимальное отражение sitemap.xml для проверки encoding/xml.
type urlset struct {
	URLs []struct {
		Loc   string `xml:"loc"`
		Links []struct {
			Hreflang string `xml:"hreflang,attr"`
			Href     string `xml:"href,attr"`
		} `xml:"http://www.w3.org/1999/xhtml link"`
	} `xml:"url"`
}

func TestWriteSitemap(t *testing.T) {
	out := t.TempDir()
	langs := []render.Lang{{Code: "ru"}, {Code: "en", Prefix: "en/"}}
	pages := []string{"en/docs/", "docs/", "", "en/", "changelog/", "en/changelog/"}
	if err := writeSitemap(out, "https://sysoev973.github.io", "/Proks_site/", pages, langs); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}

	var s urlset
	if err := xml.Unmarshal(data, &s); err != nil {
		t.Fatalf("невалидный XML: %v", err)
	}
	if len(s.URLs) != len(pages) {
		t.Fatalf("страниц в sitemap %d, ожидалось %d", len(s.URLs), len(pages))
	}

	locs := map[string][]string{}
	for _, u := range s.URLs {
		if !strings.HasPrefix(u.Loc, "https://sysoev973.github.io/Proks_site/") {
			t.Errorf("loc без origin+base: %q", u.Loc)
		}
		for _, l := range u.Links {
			locs[u.Loc] = append(locs[u.Loc], l.Hreflang+"="+l.Href)
		}
	}
	if len(locs["https://sysoev973.github.io/Proks_site/docs/"]) != 2 {
		t.Errorf("у ru-страницы docs/ не две альтернативы: %v", locs["https://sysoev973.github.io/Proks_site/docs/"])
	}
	en := locs["https://sysoev973.github.io/Proks_site/en/docs/"]
	if len(en) != 2 || !strings.Contains(en[0]+en[1], "ru=https://sysoev973.github.io/Proks_site/docs/") {
		t.Errorf("у en-страницы docs/ нет альтернативы на ru: %v", en)
	}
}

func TestWriteRobots(t *testing.T) {
	out := t.TempDir()
	if err := writeRobots(out, "https://sysoev973.github.io", "/Proks_site/"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "robots.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"User-agent: *", "Allow: /", "Sitemap: https://sysoev973.github.io/Proks_site/sitemap.xml"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("в robots.txt нет %q", want)
		}
	}
}
