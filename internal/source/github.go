package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Release — опубликованный релиз репозитория.
type Release struct {
	Name       string    `json:"name"`
	Tag        string    `json:"tag_name"`
	URL        string    `json:"html_url"`
	Body       string    `json:"body"` // Markdown
	Published  time.Time `json:"published_at"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
}

// Commit — коммит ветки; Message — только первая строка сообщения.
type Commit struct {
	SHA     string
	Message string
	URL     string
	Date    time.Time
}

// GitHub — минимальный клиент GitHub REST API.
type GitHub struct {
	API   string // https://api.github.com, в тестах — адрес httptest-сервера
	Token string // без токена API даёт 60 запросов в час
	HTTP  *http.Client
}

// NewGitHub создаёт клиент для api.github.com. token может быть пустым.
func NewGitHub(token string) *GitHub {
	return &GitHub{
		API:   "https://api.github.com",
		Token: token,
		HTTP:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Releases возвращает до n последних опубликованных релизов repo ("owner/name"), черновики пропускает.
func (g *GitHub) Releases(ctx context.Context, repo string, n int) ([]Release, error) {
	var all []Release
	if err := g.get(ctx, fmt.Sprintf("/repos/%s/releases?per_page=%d", repo, n), &all); err != nil {
		return nil, err
	}
	releases := all[:0]
	for _, r := range all {
		if !r.Draft {
			releases = append(releases, r)
		}
	}
	return releases, nil
}

// Commits возвращает до n последних коммитов ветки branch.
func (g *GitHub) Commits(ctx context.Context, repo, branch string, n int) ([]Commit, error) {
	var raw []struct {
		SHA    string `json:"sha"`
		URL    string `json:"html_url"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Date time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}
	p := fmt.Sprintf("/repos/%s/commits?sha=%s&per_page=%d", repo, url.QueryEscape(branch), n)
	if err := g.get(ctx, p, &raw); err != nil {
		return nil, err
	}
	commits := make([]Commit, 0, len(raw))
	for _, c := range raw {
		msg, _, _ := strings.Cut(c.Commit.Message, "\n")
		commits = append(commits, Commit{SHA: c.SHA, Message: msg, URL: c.URL, Date: c.Commit.Author.Date})
	}
	return commits, nil
}

func (g *GitHub) get(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.API+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := g.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Начало тела ответа объясняет причину, например превышение лимита запросов.
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("GET %s: %s: %s", path, resp.Status, strings.TrimSpace(string(msg)))
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("GET %s: decode: %w", path, err)
	}
	return nil
}
