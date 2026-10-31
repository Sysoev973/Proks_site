package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Sysoev973/Proks_site/internal/source"
)

func f(v float64) *float64 { return &v }

func benchRun(day int, hitOff, hitOn, p95Off, p95On *float64) source.BenchRun {
	var r source.BenchRun
	r.Date = time.Date(2026, 11, day, 4, 30, 0, 0, time.UTC)
	r.SHA = "0123456789abcdef"
	r.RunURL = "https://github.com/o/r/actions/runs/1"
	r.Off.Server.HitRatio, r.On.Server.HitRatio = hitOff, hitOn
	r.Off.Latency.Authors.P95, r.On.Latency.Authors.P95 = p95Off, p95On
	return r
}

func TestLineChart(t *testing.T) {
	svg := string(LineChart(`p95 <script>`, " мс", []string{"01.11.26", "08.11.26", "15.11.26"},
		Series{Name: "a", Class: "chart-on", Values: []*float64{f(1), nil, f(3)}},
		Series{Name: "b", Class: "chart-off", Values: []*float64{f(2), f(2), f(2)}},
	))

	if !strings.HasPrefix(svg, `<svg class="chart"`) || !strings.HasSuffix(svg, `</svg>`) {
		t.Fatalf("не SVG: %s", svg)
	}
	if strings.Contains(svg, "<script>") {
		t.Error("подпись не экранирована")
	}
	// У серии a пропуск в середине: две одиночные точки, ломаной нет. У b — одна ломаная.
	if n := strings.Count(svg, `class="chart-line chart-on"`); n != 0 {
		t.Errorf("ломаных серии a: %d, want 0", n)
	}
	if n := strings.Count(svg, `class="chart-line chart-off"`); n != 1 {
		t.Errorf("ломаных серии b: %d, want 1", n)
	}
	if n := strings.Count(svg, `<circle`); n != 5 {
		t.Errorf("точек: %d, want 5", n)
	}
	// Максимум 3 округляется до 5, верхнее деление — «5 мс».
	if !strings.Contains(svg, ">5 мс<") {
		t.Error("нет верхнего деления 5 мс")
	}
	if !strings.Contains(svg, "<title>a, 15.11.26: 3.00 мс</title>") {
		t.Error("нет подсказки у точки")
	}
}

func TestNiceCeil(t *testing.T) {
	tests := map[float64]float64{0: 1, 0.8: 1, 1.6: 2, 2.3: 2.5, 3: 5, 59.3: 100, 41: 50, 120: 200}
	for in, want := range tests {
		if got := niceCeil(in); got != want {
			t.Errorf("niceCeil(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestNewBenchData(t *testing.T) {
	empty := NewBenchData(nil, "o/r", "bench-data")
	if empty.Latest != nil || empty.HitChart != "" {
		t.Errorf("без запусков: %+v", empty)
	}
	if empty.DataURL != "https://github.com/o/r/blob/bench-data/history.json" {
		t.Errorf("DataURL = %q", empty.DataURL)
	}

	d := NewBenchData([]source.BenchRun{
		benchRun(1, f(0.40), f(0.58), f(2.2), f(1.7)),
		benchRun(8, f(0.411), f(0.593), f(2.09), f(1.63)),
	}, "o/r", "bench-data")

	if d.Runs != 2 || d.Latest.Date.Day() != 8 || d.ShortSHA != "0123456" {
		t.Errorf("последний запуск: runs=%d day=%d sha=%q", d.Runs, d.Latest.Date.Day(), d.ShortSHA)
	}
	rows := map[string]BenchRow{}
	for _, r := range d.Rows {
		rows[r.Name] = r
	}
	if r := rows["Авторы p95"]; r.Off != "2.09 мс" || r.On != "1.63 мс" {
		t.Errorf("Авторы p95 = %+v", r)
	}
	if r := rows["Hit ratio (сервер)"]; r.Off != "41.1%" || r.On != "59.3%" {
		t.Errorf("Hit ratio = %+v", r)
	}
	if r := rows["Бесполезные предзагрузки"]; r.Off != "—" {
		t.Errorf("пустое значение = %q, want прочерк", r.Off)
	}
	if !strings.Contains(string(d.HitChart), "59.30%") {
		t.Error("hit ratio на графике должен быть в процентах")
	}
}

func TestRenderBench(t *testing.T) {
	r := New("../../templates")
	for name, data := range map[string]BenchData{
		"пусто":   NewBenchData(nil, "o/r", "bench-data"),
		"запуски": NewBenchData([]source.BenchRun{benchRun(1, f(0.4), f(0.6), f(2), f(1.5))}, "o/r", "bench-data"),
	} {
		var buf bytes.Buffer
		err := r.Render(&buf, "bench.html", Page{Base: "/Proks_site/", Title: "Бенчмарки", Active: "bench", Data: data})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		html := buf.String()
		if !strings.Contains(html, `class="nav-link active" href="/Proks_site/bench/"`) {
			t.Errorf("%s: пункт меню не подсвечен", name)
		}
		if data.Latest == nil && !strings.Contains(html, "Замеров пока нет") {
			t.Errorf("%s: нет сообщения о пустой истории", name)
		}
		// SVG вставлен как разметка, а не экранирован в текст.
		if data.Latest != nil && (!strings.Contains(html, `<svg class="chart"`) || !strings.Contains(html, "1.50 мс")) {
			t.Errorf("%s: нет графика или таблицы", name)
		}
	}
}
