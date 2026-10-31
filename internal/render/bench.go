package render

import (
	"fmt"
	"html/template"

	"github.com/Sysoev973/Proks_site/internal/source"
)

// BenchData — данные шаблона bench.html.
type BenchData struct {
	Runs      int              // сколько запусков в истории
	Latest    *source.BenchRun // последний запуск; nil — замеров ещё нет
	ShortSHA  string
	CommitURL string
	Rows      []BenchRow // таблица последнего запуска
	HitChart  template.HTML
	P95Chart  template.HTML

	WorkflowURL string // bench.yml в Proks
	DataURL     string // history.json в ветке с результатами
}

// BenchRow — строка таблицы: метрика и её значение без предзагрузки и с ней.
type BenchRow struct {
	Name, Off, On string
}

// NewBenchData готовит страницу из истории запусков (от старых к новым) на языке l.
func NewBenchData(runs []source.BenchRun, repo, branch string, l Lang) BenchData {
	d := BenchData{
		Runs:        len(runs),
		WorkflowURL: "https://github.com/" + repo + "/actions/workflows/bench.yml",
		DataURL:     "https://github.com/" + repo + "/blob/" + branch + "/history.json",
	}
	if len(runs) == 0 {
		return d
	}

	latest := runs[len(runs)-1]
	d.Latest = &latest
	d.ShortSHA = latest.SHA
	if len(d.ShortSHA) > 7 {
		d.ShortSHA = d.ShortSHA[:7]
	}
	d.CommitURL = "https://github.com/" + repo + "/commit/" + latest.SHA

	unit := " " + l.Text("bench.ms")
	ms := func(v *float64) string { return format(v, 1, "%.2f"+unit) }
	off, on := latest.Off, latest.On
	d.Rows = []BenchRow{
		{l.Text("bench.row_authors_p50"), ms(off.Latency.Authors.P50), ms(on.Latency.Authors.P50)},
		{l.Text("bench.row_authors_p95"), ms(off.Latency.Authors.P95), ms(on.Latency.Authors.P95)},
		{l.Text("bench.row_authors_p99"), ms(off.Latency.Authors.P99), ms(on.Latency.Authors.P99)},
		{l.Text("bench.row_card_p95"), ms(off.Latency.Card.P95), ms(on.Latency.Card.P95)},
		{l.Text("bench.row_all_p95"), ms(off.Latency.All.P95), ms(on.Latency.All.P95)},
		{l.Text("bench.row_hit"), pct(off.Server.HitRatio), pct(on.Server.HitRatio)},
		{l.Text("bench.row_upstream"), rps(off.Server.UpstreamRPS), rps(on.Server.UpstreamRPS)},
		{l.Text("bench.row_useless"), pct(off.Server.UselessPrefetch), pct(on.Server.UselessPrefetch)},
		{l.Text("bench.row_failed"), pct(off.FailedRate), pct(on.FailedRate)},
	}

	labels := make([]string, len(runs))
	hitOff, hitOn := make([]*float64, len(runs)), make([]*float64, len(runs))
	p95Off, p95On := make([]*float64, len(runs)), make([]*float64, len(runs))
	for i, r := range runs {
		labels[i] = r.Date.Format("02.01.06")
		hitOff[i], hitOn[i] = percent(r.Off.Server.HitRatio), percent(r.On.Server.HitRatio)
		p95Off[i], p95On[i] = r.Off.Latency.Authors.P95, r.On.Latency.Authors.P95
	}
	offName, onName := l.Text("bench.off"), l.Text("bench.on")
	d.HitChart = LineChart(l.Text("bench.hit_chart"), "%", labels,
		Series{Name: offName, Class: "chart-off", Values: hitOff},
		Series{Name: onName, Class: "chart-on", Values: hitOn})
	d.P95Chart = LineChart(l.Text("bench.p95_chart"), unit, labels,
		Series{Name: offName, Class: "chart-off", Values: p95Off},
		Series{Name: onName, Class: "chart-on", Values: p95On})
	return d
}

// percent переводит долю 0..1 в проценты для графика.
func percent(v *float64) *float64 {
	if v == nil {
		return nil
	}
	p := *v * 100
	return &p
}

func pct(v *float64) string { return format(v, 100, "%.1f%%") }
func rps(v *float64) string { return format(v, 1, "%.1f rps") }

// format печатает v*scale по шаблону, а отсутствующее значение — прочерком.
func format(v *float64, scale float64, layout string) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf(layout, *v*scale)
}
