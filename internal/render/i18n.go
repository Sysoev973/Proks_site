package render

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Lang — язык сайта и словарь строк интерфейса из content/i18n/<code>.json.
type Lang struct {
	Code    string // "ru", "en"; идёт в <html lang>
	Prefix  string // путь версии сайта от base: "" у языка по умолчанию, "en/" у остальных
	Strings map[string]string
}

// LoadLangs читает словари codes из dir. Первый язык — язык по умолчанию, он живёт в корне сайта.
// Набор ключей во всех словарях должен совпадать, иначе на одной из версий сайта не хватит строк.
func LoadLangs(dir string, codes []string) ([]Lang, error) {
	langs := make([]Lang, len(codes))
	for i, code := range codes {
		data, err := os.ReadFile(filepath.Join(dir, code+".json"))
		if err != nil {
			return nil, err
		}
		l := Lang{Code: code}
		if err := json.Unmarshal(data, &l.Strings); err != nil {
			return nil, fmt.Errorf("%s.json: %w", code, err)
		}
		if i > 0 {
			l.Prefix = code + "/"
		}
		langs[i] = l
	}
	for _, l := range langs[1:] {
		if missing := diffKeys(langs[0].Strings, l.Strings); len(missing) > 0 {
			return nil, fmt.Errorf("в %s.json нет ключей: %s", l.Code, strings.Join(missing, ", "))
		}
		if extra := diffKeys(l.Strings, langs[0].Strings); len(extra) > 0 {
			return nil, fmt.Errorf("в %s.json лишние ключи: %s", l.Code, strings.Join(extra, ", "))
		}
	}
	return langs, nil
}

// diffKeys — ключи a, которых нет в b, по алфавиту.
func diffKeys(a, b map[string]string) []string {
	var keys []string
	for k := range a {
		if _, ok := b[k]; !ok {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

// T — функция t в шаблонах. Строки словаря пишем мы сами, поэтому в них можно
// использовать разметку (<code>, &nbsp;). Неизвестный ключ — ошибка сборки, а не пустое место на странице.
func (l Lang) T(key string) (template.HTML, error) {
	s, ok := l.Strings[key]
	if !ok {
		return "", fmt.Errorf("нет строки %q в %s.json", key, l.Code)
	}
	return template.HTML(s), nil
}

// Text — строка словаря для Go-кода, например подписи графиков. Неизвестный ключ
// возвращается как есть, чтобы его было видно на странице и в тестах.
func (l Lang) Text(key string) string {
	if s, ok := l.Strings[key]; ok {
		return s
	}
	return key
}

// Alternate — та же страница на одном из языков, для переключателя и <link hreflang>.
type Alternate struct {
	Code, Short, Name, URL string
	Current                bool
}

// Alternates возвращает адреса страницы path (от корня версии сайта) на всех языках.
func Alternates(base, path string, langs []Lang, current Lang) []Alternate {
	alts := make([]Alternate, len(langs))
	for i, l := range langs {
		alts[i] = Alternate{
			Code:    l.Code,
			Short:   l.Text("lang.short"),
			Name:    l.Text("lang.name"),
			URL:     base + l.Prefix + path,
			Current: l.Code == current.Code,
		}
	}
	return alts
}
