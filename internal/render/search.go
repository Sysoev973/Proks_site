package render

import (
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// SearchEntry — секция документа в поисковом индексе (dist/<язык>/search-index.json).
type SearchEntry struct {
	Title   string `json:"title"`   // заголовок документа (h1)
	URL     string `json:"url"`     // страница с якорем секции: "/Proks_site/docs/a/#запуск"
	Section string `json:"section"` // заголовок секции h2/h3, пусто у вступления
	Text    string `json:"text"`    // чистый текст секции без разметки, обрезан до searchTextLimit
}

// searchTextLimit — сколько символов текста храним на секцию: индекс остаётся маленьким,
// а для сниппета в выдаче хватает.
const searchTextLimit = 500

// SearchIndex режет Markdown документа на секции по заголовкам h2/h3 и извлекает из них
// чистый текст. pageURL — адрес страницы документа с языковым префиксом, к нему добавляются
// якоря тех же id, что ставит Markdown (генератор githubIDs общий).
func SearchIndex(src []byte, title, pageURL string) ([]SearchEntry, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	ctx := parser.NewContext(parser.WithIDs(&githubIDs{seen: map[string]int{}}))
	doc := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))

	var entries []SearchEntry
	section, url := "", pageURL
	var b strings.Builder
	started := false // был заголовок или текст: не плодим пустую запись у документа, начинающегося с h2
	flush := func() {
		if !started {
			return
		}
		t := strings.Join(strings.Fields(b.String()), " ") // схлопываем переносы и отступы кода
		if utf8.RuneCountInString(t) > searchTextLimit {
			t = string([]rune(t)[:searchTextLimit])
		}
		entries = append(entries, SearchEntry{Title: title, URL: url, Section: section, Text: t})
		b.Reset()
	}

	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*ast.Heading); ok {
			switch {
			case h.Level == 1:
				continue // h1 и так в Title
			case h.Level == 2 || h.Level == 3:
				flush()
				started = true
				section = plainText(src, h)
				if id, ok := h.AttributeString("id"); ok {
					if b, ok := id.([]byte); ok {
						url = pageURL + "#" + string(b)
					}
				}
				continue
			}
		}
		started = true
		plainTextInto(src, n, &b)
	}
	flush()
	return entries, nil
}

// plainText — текст узла без разметки.
func plainText(src []byte, n ast.Node) string {
	var b strings.Builder
	plainTextInto(src, n, &b)
	return strings.TrimSpace(b.String())
}

// plainTextInto собирает в b текстовые сегменты узла: обычный текст, код в строке и блоки кода.
func plainTextInto(src []byte, n ast.Node, b *strings.Builder) {
	if cb, ok := n.(*ast.CodeBlock); ok {
		writeLines(src, cb.Lines(), b)
		return
	}
	if fb, ok := n.(*ast.FencedCodeBlock); ok {
		writeLines(src, fb.Lines(), b)
		return
	}
	_ = ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		// Блочные узлы отделяем пробелом: иначе пункты списка и ячейки таблиц сливаются.
		case *ast.Paragraph, *ast.ListItem, *ast.Heading, *east.TableCell:
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
		}
		switch v := n.(type) {
		case *ast.Text:
			b.Write(v.Segment.Value(src))
			if v.SoftLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(v.Value)
		case *ast.CodeBlock, *ast.FencedCodeBlock:
			// Блоки кода внутри списков и цитат тоже идут в текст.
			writeLines(src, v.(interface{ Lines() *text.Segments }).Lines(), b)
		}
		return ast.WalkContinue, nil
	})
	b.WriteByte(' ')
}

func writeLines(src []byte, lines *text.Segments, b *strings.Builder) {
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(src))
		b.WriteByte(' ')
	}
}
