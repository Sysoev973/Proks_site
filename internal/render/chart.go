package render

import (
	"fmt"
	"html"
	"html/template"
	"math"
	"strings"
)

// Series — линия графика. nil в Values — значения нет, линия в этом месте прерывается.
// Class задаёт цвет через site.css (chart-off, chart-on), поэтому график сам следует теме.
type Series struct {
	Name   string
	Class  string
	Values []*float64
}

// Размеры в единицах viewBox; на странице SVG растягивается по ширине колонки.
const (
	chartW, chartH                     = 640, 260
	padLeft, padRight, padTop, padDown = 56, 16, 12, 28
	chartTicks                         = 4
)

// LineChart рисует SVG-график: по X — запуски (подписи labels), по Y — значения от нуля.
// unit дописывается к числам на оси и во всплывающих подсказках.
func LineChart(title, unit string, labels []string, series ...Series) template.HTML {
	maxV := 0.0
	for _, s := range series {
		for _, v := range s.Values {
			if v != nil && *v > maxV {
				maxV = *v
			}
		}
	}
	top := niceCeil(maxV)
	plotW := float64(chartW - padLeft - padRight)
	plotH := float64(chartH - padTop - padDown)
	x := func(i int) float64 {
		if len(labels) < 2 {
			return padLeft + plotW/2
		}
		return padLeft + plotW*float64(i)/float64(len(labels)-1)
	}
	y := func(v float64) float64 { return padTop + plotH*(1-v/top) }

	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="chart" viewBox="0 0 %d %d" role="img" aria-label="%s">`, chartW, chartH, html.EscapeString(title))

	// Сетка и подписи оси Y.
	for i := 0; i <= chartTicks; i++ {
		v := top * float64(i) / chartTicks
		fmt.Fprintf(&b, `<line class="chart-grid" x1="%d" x2="%d" y1="%.1f" y2="%.1f"/>`, padLeft, chartW-padRight, y(v), y(v))
		fmt.Fprintf(&b, `<text class="chart-label" x="%d" y="%.1f" text-anchor="end" dominant-baseline="middle">%s</text>`,
			padLeft-6, y(v), html.EscapeString(formatTick(v)+unit))
	}

	// Подписи оси X: первая и последняя точки, иначе на телефоне они налезают друг на друга.
	if len(labels) > 0 {
		last := len(labels) - 1
		anchor := "start"
		if last == 0 {
			anchor = "middle"
		}
		fmt.Fprintf(&b, `<text class="chart-label" x="%.1f" y="%d" text-anchor="%s">%s</text>`, x(0), chartH-8, anchor, html.EscapeString(labels[0]))
		if last > 0 {
			fmt.Fprintf(&b, `<text class="chart-label" x="%.1f" y="%d" text-anchor="end">%s</text>`, x(last), chartH-8, html.EscapeString(labels[last]))
		}
	}

	for _, s := range series {
		cls := html.EscapeString(s.Class)
		// Ломаная рвётся на пропущенных значениях: каждый непрерывный кусок — свой polyline.
		var pts []string
		flush := func() {
			if len(pts) > 1 {
				fmt.Fprintf(&b, `<polyline class="chart-line %s" points="%s"/>`, cls, strings.Join(pts, " "))
			}
			pts = pts[:0]
		}
		for i, v := range s.Values {
			if v == nil || i >= len(labels) {
				flush()
				continue
			}
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", x(i), y(*v)))
		}
		flush()

		for i, v := range s.Values {
			if v == nil || i >= len(labels) {
				continue
			}
			tip := fmt.Sprintf("%s, %s: %.2f%s", s.Name, labels[i], *v, unit)
			fmt.Fprintf(&b, `<circle class="chart-dot %s" cx="%.1f" cy="%.1f" r="3.5"><title>%s</title></circle>`,
				cls, x(i), y(*v), html.EscapeString(tip))
		}
	}

	b.WriteString(`</svg>`)
	// Всё содержимое собрано здесь, а текст экранирован html.EscapeString.
	return template.HTML(b.String())
}

// niceCeil округляет верх оси до «круглого» числа 1, 2, 2.5 или 5 × 10^k, чтобы деления были ровными.
func niceCeil(v float64) float64 {
	if v <= 0 {
		return 1
	}
	p := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if v <= m*p {
			return m * p
		}
	}
	return 10 * p
}

// formatTick печатает деление оси без лишних нулей: 50, 2.5, 0.25.
func formatTick(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
