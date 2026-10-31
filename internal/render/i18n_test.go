package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLangs(t *testing.T) {
	ru, en, all := testLangs(t)
	if ru.Prefix != "" || en.Prefix != "en/" || len(all) != 2 {
		t.Errorf("префиксы: ru=%q en=%q", ru.Prefix, en.Prefix)
	}
	if got, _ := en.T("nav.docs"); got != "Docs" {
		t.Errorf("en nav.docs = %q", got)
	}
	if _, err := ru.T("no.such.key"); err == nil {
		t.Error("неизвестный ключ в шаблоне должен ронять сборку")
	}
}

func TestLoadLangsKeyMismatch(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ru.json", `{"a": "А", "b": "Б"}`)
	write("en.json", `{"a": "A", "c": "C"}`)

	_, err := LoadLangs(dir, []string{"ru", "en"})
	if err == nil || !strings.Contains(err.Error(), "нет ключей: b") {
		t.Errorf("err = %v, want про ключ b", err)
	}

	write("en.json", `{"a": "A", "b": "B", "c": "C"}`)
	_, err = LoadLangs(dir, []string{"ru", "en"})
	if err == nil || !strings.Contains(err.Error(), "лишние ключи: c") {
		t.Errorf("err = %v, want про ключ c", err)
	}
}

func TestAlternates(t *testing.T) {
	_, en, all := testLangs(t)
	alts := Alternates("/Proks_site/", "bench/", all, en)
	if alts[0].URL != "/Proks_site/bench/" || alts[0].Current || alts[0].Short != "RU" {
		t.Errorf("ru: %+v", alts[0])
	}
	if alts[1].URL != "/Proks_site/en/bench/" || !alts[1].Current {
		t.Errorf("en: %+v", alts[1])
	}
}
