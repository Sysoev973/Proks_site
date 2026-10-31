package render

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Markdown превращает Markdown документа from в HTML и переписывает в нём ссылки через links.
//
// Сырой HTML из Markdown не выводится (goldmark без html.WithUnsafe): так чужой
// <script> из README не попадёт на сайт. Цена — пропадают <p align>, <img width> и т.п.
func Markdown(src []byte, from string, links Links) (template.HTML, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(rewriter{from: from, links: links}, 100)),
		),
	)
	var buf bytes.Buffer
	ctx := parser.NewContext(parser.WithIDs(&githubIDs{seen: map[string]int{}}))
	if err := md.Convert(src, &buf, parser.WithContext(ctx)); err != nil {
		return "", err
	}
	// Безопасно: goldmark сам экранирует текст и не пропускает сырой HTML.
	return template.HTML(buf.String()), nil
}

// githubIDs строит id заголовков как GitHub: "Установка и запуск" → "установка-и-запуск".
// Встроенный генератор goldmark выбрасывает кириллицу, и ссылки "#раздел" из README не работали бы.
type githubIDs struct {
	seen map[string]int
}

func (g *githubIDs) Generate(value []byte, _ ast.NodeKind) []byte {
	var b strings.Builder
	for _, r := range strings.ToLower(string(value)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	id := b.String()
	if id == "" {
		id = "section"
	}
	// Повторы получают суффикс, как на GitHub: "x", "x-1", "x-2".
	n := g.seen[id]
	g.seen[id]++
	if n > 0 {
		id = fmt.Sprintf("%s-%d", id, n)
	}
	return []byte(id)
}

func (g *githubIDs) Put(value []byte) {
	g.seen[string(value)]++
}

// rewriter правит дерево документа до рендера: ссылки, картинки и классы Bootstrap для таблиц.
type rewriter struct {
	from  string
	links Links
}

func (r rewriter) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Link:
			v.Destination = []byte(r.links.Resolve(r.from, string(v.Destination), false))
		case *ast.Image:
			v.Destination = []byte(r.links.Resolve(r.from, string(v.Destination), true))
		case *east.Table:
			v.SetAttributeString("class", []byte("table table-sm table-bordered"))
		}
		return ast.WalkContinue, nil
	})
}
