package source

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// BenchRun — один запуск .github/workflows/bench.yml в Proks: A/B-замер предзагрузки
// (k6/prefetch-chain.js) с выключенной и включённой предзагрузкой.
// Поля повторяют JSON из handleSummary скрипта; то, что сайту не нужно, не разбираем.
type BenchRun struct {
	Date   time.Time   `json:"date"`
	SHA    string      `json:"sha"`
	Ref    string      `json:"ref"`
	Event  string      `json:"event"`
	RunURL string      `json:"run_url"`
	Off    BenchResult `json:"prefetch_off"`
	On     BenchResult `json:"prefetch_on"`
}

// BenchResult — итог одного режима. Указатели, потому что k6 пишет null или
// пропускает поле, если величину не из чего посчитать.
type BenchResult struct {
	Latency struct {
		All     Percentiles `json:"all"`
		Card    Percentiles `json:"card"`
		Authors Percentiles `json:"authors"`
	} `json:"latency_ms"`
	Server struct {
		HitRatio        *float64 `json:"hit_ratio"`
		UpstreamRPS     *float64 `json:"upstream_rps"`
		UselessPrefetch *float64 `json:"useless_prefetch_share"`
	} `json:"server"`
	FailedRate *float64 `json:"failed_rate"`
}

// Percentiles — задержка в миллисекундах.
type Percentiles struct {
	P50 *float64 `json:"p50"`
	P95 *float64 `json:"p95"`
	P99 *float64 `json:"p99"`
}

// BenchHistory читает history.json из ветки branch репозитория repo (от старых запусков к новым).
// Если ветки или файла ещё нет, то есть bench.yml ни разу не запускался, возвращает nil без ошибки.
func (g *GitHub) BenchHistory(ctx context.Context, repo, branch string) ([]BenchRun, error) {
	var runs []BenchRun
	p := fmt.Sprintf("/repos/%s/contents/history.json?ref=%s", repo, url.QueryEscape(branch))
	// raw — содержимое файла как есть, а не JSON-обёртка с base64.
	err := g.getAs(ctx, p, "application/vnd.github.raw+json", &runs)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return runs, err
}
