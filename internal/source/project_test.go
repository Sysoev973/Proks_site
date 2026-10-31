package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadProject(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module proks\n\ngo 1.25.0\n\nrequire x v1\n")
	writeFile(t, root, "internal/cache/cache.go", "package cache")
	writeFile(t, root, "internal/cache/cache_test.go", "package cache")
	writeFile(t, root, "internal/cache/lru_test.go", "package cache")
	writeFile(t, root, "internal/events/events.go", "package events")
	writeFile(t, root, "internal/README.md", "не пакет")

	p, err := LoadProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if p.GoVersion != "1.25" {
		t.Errorf("GoVersion = %q, want 1.25", p.GoVersion)
	}
	if p.Tests["cache"] != 2 || p.Tests["events"] != 0 || len(p.Tests) != 2 {
		t.Errorf("Tests = %v", p.Tests)
	}
}

func TestLoadProjectWithoutGoLine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module proks\n")
	if _, err := LoadProject(root); err == nil {
		t.Fatal("ожидали ошибку без строки go")
	}
}

func TestRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"full_name":"o/r","stargazers_count":7}`))
	}))
	defer srv.Close()

	gh := NewGitHub("")
	gh.API = srv.URL
	repo, err := gh.Repo(context.Background(), "o/r")
	if err != nil || repo.Stars != 7 {
		t.Errorf("repo = %+v, err = %v", repo, err)
	}
}
