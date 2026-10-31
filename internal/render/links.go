package render

import (
	"net/url"
	"path"
	"strings"
)

// Links переписывает относительные ссылки из Markdown репозитория Proks:
// ссылки на документы, у которых есть страница на сайте, ведут на эту страницу,
// картинки — на raw-файл в GitHub, всё остальное — на файл в GitHub.
type Links struct {
	Base   string // базовый путь сайта, "/Proks_site/"
	Repo   string // "Sysoev973/Proks"
	Branch string // "main"
	// Pages — путь документа в репозитории → путь его страницы от Base: "docs/a.md" → "docs/a/".
	Pages map[string]string
}

// Resolve возвращает адрес для ссылки dest из документа from (путь в репозитории, "" — корень).
func (l Links) Resolve(from, dest string, image bool) string {
	if dest == "" || strings.HasPrefix(dest, "#") {
		return dest
	}
	u, err := url.Parse(dest)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return dest // абсолютная ссылка, mailto: и т.п.
	}

	// "/x" в Markdown на GitHub означает путь от корня репозитория.
	var p string
	if strings.HasPrefix(u.Path, "/") {
		p = path.Clean(strings.TrimPrefix(u.Path, "/"))
	} else {
		p = path.Join(path.Dir(from), u.Path)
	}
	if p == ".." || strings.HasPrefix(p, "../") {
		return dest // ссылка за пределы репозитория, переписывать не во что
	}

	if !image {
		if page, ok := l.Pages[p]; ok {
			s := l.Base + page
			if u.Fragment != "" {
				s += "#" + u.Fragment
			}
			return s
		}
	}

	gh := url.URL{Scheme: "https", RawQuery: u.RawQuery, Fragment: u.Fragment}
	switch {
	case image:
		gh.Host = "raw.githubusercontent.com"
		gh.Path = "/" + l.Repo + "/" + l.Branch + "/" + p
	case p == ".":
		gh.Host = "github.com"
		gh.Path = "/" + l.Repo + "/tree/" + l.Branch
	default:
		// Для каталогов GitHub сам перенаправит blob → tree.
		gh.Host = "github.com"
		gh.Path = "/" + l.Repo + "/blob/" + l.Branch + "/" + p
	}
	return gh.String()
}

// SourceURL — страница файла path на GitHub.
func (l Links) SourceURL(p string) string {
	return "https://github.com/" + l.Repo + "/blob/" + l.Branch + "/" + p
}
