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

// NewBenchData готовит страницу из истории запусков (от старых к новым).
func NewBenchData(runs []source.BenchRun, repo, branch string) BenchData {
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

	off, on := latest.Off, latest.On
	d.Rows = []BenchRow{
		{"Авторы p50", ms(off.Latency.Authors.P50), ms(on.Latency.Authors.P50)},
		{"Авторы p95", ms(off.Latency.Authors.P95), ms(on.Latency.Authors.P95)},
		{"Авторы p99", ms(off.Latency.Authors.P99), ms(on.Latency.Authors.P99)},
		{"Карточка p95", ms(off.Latency.Card.P95), ms(on.Latency.Card.P95)},
		{"Все запросы p95", ms(off.Latency.All.P95), ms(on.Latency.All.P95)},
		{"Hit ratio (сервер)", pct(off.Server.HitRatio), pct(on.Server.HitRatio)},
		{"Запросы к upstream", rps(off.Server.UpstreamRPS), rps(on.Server.UpstreamRPS)},
		{"Бесполезные предзагрузки", pct(off.Server.UselessPrefetch), pct(on.Server.UselessPrefetch)},
		{"Ошибки", pct(off.FailedRate), pct(on.FailedRate)},
	}

	labels := make([]string, len(runs))
	hitOff, hitOn := make([]*float64, len(runs)), make([]*float64, len(runs))
	p95Off, p95On := make([]*float64, len(runs)), make([]*float64, len(runs))
	for i, r := range runs {
		labels[i] = r.Date.Format("02.01.06")
		hitOff[i], hitOn[i] = percent(r.Off.Server.HitRatio), percent(r.On.Server.HitRatio)
		p95Off[i], p95On[i] = r.Off.Latency.Authors.P95, r.On.Latency.Authors.P95
	}
	d.HitChart = LineChart("Hit ratio сервера по запускам", "%", labels,
		Series{Name: "без предзагрузки", Class: "chart-off", Values: hitOff},
		Series{Name: "с предзагрузкой", Class: "chart-on", Values: hitOn})
	d.P95Chart = LineChart("p95 запроса авторов по запускам", " мс", labels,
		Series{Name: "без предзагрузки", Class: "chart-off", Values: p95Off},
		Series{Name: "с предзагрузкой", Class: "chart-on", Values: p95On})
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

func ms(v *float64) string  { return format(v, 1, "%.2f мс") }
func pct(v *float64) string { return format(v, 100, "%.1f%%") }
func rps(v *float64) string { return format(v, 1, "%.1f rps") }

// format печатает v*scale по шаблону, а отсутствующее значение — прочерком.
func format(v *float64, scale float64, layout string) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf(layout, *v*scale)
}
