// Package source читает данные о Proks: Markdown из локальной копии репозитория и релизы с коммитами из GitHub API.
package source

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Doc — один Markdown-документ из репозитория Proks.
type Doc struct {
	// Path — путь в репозитории через "/": "README.md", "docs/setup.md".
	Path     string
	Title    string
	Markdown []byte
}

// LoadDocs читает README.md и docs/*.md из копии репозитория root.
// README обязателен, каталога docs может не быть. Документы из docs/ идут по алфавиту.
func LoadDocs(root string) ([]Doc, error) {
	readme, err := readDoc(root, "README.md")
	if err != nil {
		return nil, err
	}
	docs := []Doc{readme}

	// Glob возвращает файлы уже отсортированными.
	matches, err := filepath.Glob(filepath.Join(root, "docs", "*.md"))
	if err != nil {
		return nil, err
	}
	for _, m := range matches {
		d, err := readDoc(root, path.Join("docs", filepath.Base(m)))
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func readDoc(root, rel string) (Doc, error) {
	md, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return Doc{}, fmt.Errorf("read %s: %w", rel, err)
	}
	// Редакторы на Windows пишут UTF-8 с BOM; с ним первая строка "# ..." не считается заголовком.
	md = bytes.TrimPrefix(md, []byte("\xef\xbb\xbf"))
	return Doc{Path: rel, Title: title(md, strings.TrimSuffix(path.Base(rel), ".md")), Markdown: md}, nil
}

// title возвращает текст первого заголовка "# ..." или fallback, если его нет.
func title(md []byte, fallback string) string {
	sc := bufio.NewScanner(bytes.NewReader(md))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "# "); ok {
			if t := strings.TrimSpace(strings.TrimRight(rest, "#")); t != "" {
				return t
			}
		}
	}
	return fallback
}
