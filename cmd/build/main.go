// Команда build собирает сайт в каталог dist/.
//
//	go run ./cmd/build -base /Proks_site/
//	go run ./cmd/build -base /Proks_site/ -serve :8080   # и открыть http://localhost:8080/Proks_site/
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sysoev973/Proks_site/internal/render"
)

func main() {
	base := flag.String("base", "/", "базовый путь сайта, например /Proks_site/")
	out := flag.String("out", "dist", "каталог для готового сайта (очищается перед сборкой)")
	templates := flag.String("templates", "templates", "каталог с шаблонами")
	static := flag.String("static", "static", "каталог со статикой, копируется в <out>/static")
	serve := flag.String("serve", "", "после сборки раздавать сайт по этому адресу, например :8080")
	flag.Parse()

	b := render.NormalizeBase(*base)
	if err := build(b, *out, *templates, *static); err != nil {
		log.Fatal(err)
	}
	log.Printf("сайт собран в %s (base %s)", *out, b)

	if *serve != "" {
		log.Fatal(listen(*serve, b, *out))
	}
}

func build(base, out, templates, static string) error {
	// Защита от случайного -out . : каталог удаляется целиком.
	if c := filepath.Clean(out); c == "." || c == ".." || c == filepath.VolumeName(c)+string(filepath.Separator) {
		return fmt.Errorf("опасный каталог для -out: %q", out)
	}
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	r := render.New(templates)
	var buf bytes.Buffer
	if err := r.Render(&buf, "index.html", render.Page{Base: base, Title: "Proks — кэширующий прокси"}); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), buf.Bytes(), 0o644); err != nil {
		return err
	}

	return os.CopyFS(filepath.Join(out, "static"), os.DirFS(static))
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
