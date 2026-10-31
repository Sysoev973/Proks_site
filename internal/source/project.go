package source

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Project — сведения о коде Proks из локальной копии репозитория.
type Project struct {
	GoVersion string         // из go.mod без патч-версии: "1.25"
	Tests     map[string]int // пакет internal/<имя> → число файлов *_test.go
}

// LoadProject читает go.mod и считает тестовые файлы в пакетах internal/.
func LoadProject(root string) (Project, error) {
	p := Project{Tests: map[string]int{}}

	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return p, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "go "); ok {
			// "1.25.0" → "1.25": на сайте важна линейка, а не патч.
			parts := strings.SplitN(strings.TrimSpace(v), ".", 3)
			p.GoVersion = strings.Join(parts[:min(2, len(parts))], ".")
			break
		}
	}
	if err := sc.Err(); err != nil {
		return p, err
	}
	if p.GoVersion == "" {
		return p, fmt.Errorf("в %s нет строки go", filepath.Join(root, "go.mod"))
	}

	dirs, err := os.ReadDir(filepath.Join(root, "internal"))
	if err != nil {
		return p, err
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		tests, err := filepath.Glob(filepath.Join(root, "internal", d.Name(), "*_test.go"))
		if err != nil {
			return p, err
		}
		p.Tests[d.Name()] = len(tests)
	}
	return p, nil
}

// Repo — сведения о репозитории на GitHub.
type Repo struct {
	Stars int `json:"stargazers_count"`
}

// Repo возвращает сведения о репозитории repo ("owner/name").
func (g *GitHub) Repo(ctx context.Context, repo string) (Repo, error) {
	var r Repo
	err := g.get(ctx, "/repos/"+repo, &r)
	return r, err
}
