package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDocs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "Бейдж\n\n# Proks #\n\nТекст")
	writeFile(t, root, "docs/b.md", "\xef\xbb\xbf# Второй") // UTF-8 с BOM, как сохраняет PowerShell 5.1
	writeFile(t, root, "docs/a.md", "без заголовка")
	writeFile(t, root, "docs/notes.txt", "не Markdown")

	docs, err := LoadDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ path, title string }{
		{"README.md", "Proks"},
		{"docs/a.md", "a"},
		{"docs/b.md", "Второй"},
	}
	if len(docs) != len(want) {
		t.Fatalf("получили %d документов, want %d", len(docs), len(want))
	}
	for i, w := range want {
		if docs[i].Path != w.path || docs[i].Title != w.title {
			t.Errorf("docs[%d] = %q %q, want %q %q", i, docs[i].Path, docs[i].Title, w.path, w.title)
		}
	}
}

func TestLoadDocsWithoutDocsDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Proks")
	docs, err := LoadDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("получили %d документов, want 1", len(docs))
	}
}

func TestLoadDocsWithoutReadme(t *testing.T) {
	if _, err := LoadDocs(t.TempDir()); err == nil {
		t.Fatal("ожидали ошибку без README.md")
	}
}

func TestGitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/repos/o/r/releases":
			w.Write([]byte(`[
				{"name":"v1","tag_name":"v1.0.0","html_url":"https://x/v1","body":"**ok**","published_at":"2026-10-01T10:00:00Z"},
				{"name":"draft","draft":true}
			]`))
		case "/repos/o/r/commits":
			if r.URL.Query().Get("sha") != "main" {
				t.Errorf("sha = %q", r.URL.Query().Get("sha"))
			}
			w.Write([]byte(`[{"sha":"abc123","html_url":"https://x/c","commit":{"message":"feat: x\n\nподробности","author":{"date":"2026-10-02T10:00:00Z"}}}]`))
		case "/repos/o/r/contents/history.json":
			if r.URL.Query().Get("ref") != "bench-data" {
				t.Errorf("ref = %q", r.URL.Query().Get("ref"))
			}
			if got := r.Header.Get("Accept"); got != "application/vnd.github.raw+json" {
				t.Errorf("Accept = %q", got)
			}
			w.Write([]byte(`[{"date":"2026-11-02T04:30:00Z","sha":"abc123",
				"prefetch_off":{"latency_ms":{"authors":{"p95":2.1}},"server":{"hit_ratio":0.41,"useless_prefetch_share":null}},
				"prefetch_on":{"latency_ms":{"authors":{"p95":1.6}},"server":{"hit_ratio":0.59}}}]`))
		case "/repos/o/empty/contents/history.json":
			http.Error(w, `{"message":"No commit found for the ref bench-data"}`, http.StatusNotFound)
		default:
			http.Error(w, `{"message":"API rate limit exceeded"}`, http.StatusForbidden)
		}
	}))
	defer srv.Close()

	gh := NewGitHub("secret")
	gh.API = srv.URL
	ctx := context.Background()

	releases, err := gh.Releases(ctx, "o/r", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].Tag != "v1.0.0" || releases[0].Published.Day() != 1 {
		t.Errorf("releases = %+v", releases)
	}

	commits, err := gh.Commits(ctx, "o/r", "main", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 || commits[0].Message != "feat: x" {
		t.Errorf("commits = %+v", commits)
	}

	if _, err := gh.Releases(ctx, "o/missing", 5); err == nil {
		t.Error("ожидали ошибку на 403")
	}

	runs, err := gh.BenchHistory(ctx, "o/r", "bench-data")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || *runs[0].On.Latency.Authors.P95 != 1.6 || *runs[0].Off.Server.HitRatio != 0.41 {
		t.Errorf("runs = %+v", runs)
	}
	if runs[0].Off.Server.UselessPrefetch != nil || runs[0].On.Latency.Card.P95 != nil {
		t.Error("null и пропущенные поля должны остаться nil")
	}

	// Ветки bench-data ещё нет: это не ошибка, а пустая история.
	runs, err = gh.BenchHistory(ctx, "o/empty", "bench-data")
	if err != nil || runs != nil {
		t.Errorf("пустая история: runs = %v, err = %v", runs, err)
	}
	// Другие ошибки API не глотаются.
	if _, err := gh.BenchHistory(ctx, "o/missing", "bench-data"); err == nil {
		t.Error("ожидали ошибку на 403")
	}
}
