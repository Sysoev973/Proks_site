package render

import (
	"strings"
	"testing"
)

func TestSearchIndex(t *testing.T) {
	src := []byte(`# Документ

Вступление с ` + "`inline-кодом`" + ` и **жирным**.

## Установка и запуск

Поставь зависимости:

- раз
- два

` + "```sh" + `
go mod tidy
` + "```" + `

### Проверка

Запусти тесты.

## Пустая секция

## Последняя

Текст последней.
`)
	entries, err := SearchIndex(src, "Документ", "/Proks_site/docs/")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("записей %d, ожидалось 5 (вступление + 4 секции): %+v", len(entries), entries)
	}

	intro := entries[0]
	if intro.Section != "" || intro.URL != "/Proks_site/docs/" {
		t.Errorf("вступление: %+v", intro)
	}
	if !strings.Contains(intro.Text, "Вступление с inline-кодом и жирным.") {
		t.Errorf("текст вступления с разметкой или потерян: %q", intro.Text)
	}

	install := entries[1]
	if install.Section != "Установка и запуск" || install.URL != "/Proks_site/docs/#установка-и-запуск" {
		t.Errorf("секция установки: %+v", install)
	}
	if !strings.Contains(install.Text, "go mod tidy") {
		t.Errorf("текст блока кода потерян: %q", install.Text)
	}
	if !strings.Contains(install.Text, "раз два") {
		t.Errorf("пункты списка слиплись: %q", install.Text)
	}

	check := entries[2]
	if check.Section != "Проверка" || check.URL != "/Proks_site/docs/#проверка" {
		t.Errorf("секция h3: %+v", check)
	}

	empty := entries[3]
	if empty.Section != "Пустая секция" || empty.Text != "" {
		t.Errorf("пустая секция: %+v", empty)
	}

	for _, e := range entries {
		if e.Title != "Документ" {
			t.Errorf("title: %+v", e)
		}
		for _, bad := range []string{"<", "**", "`", "# "} {
			if strings.Contains(e.Text, bad) {
				t.Errorf("в Text осталась разметка %q: %q", bad, e.Text)
			}
		}
	}
}

func TestSearchIndexTruncates(t *testing.T) {
	src := []byte("# T\n\n## S\n\n" + strings.Repeat("а", 600) + "\n")
	entries, err := SearchIndex(src, "T", "/x/")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("записей %d: %+v", len(entries), entries)
	}
	if n := len([]rune(entries[0].Text)); n != 500 {
		t.Errorf("текст не обрезан до 500 символов: %d", n)
	}
}

func TestSearchIndexNoHeadings(t *testing.T) {
	entries, err := SearchIndex([]byte("Просто текст без заголовков.\n"), "T", "/x/")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Section != "" || entries[0].URL != "/x/" {
		t.Fatalf("записи: %+v", entries)
	}
}
