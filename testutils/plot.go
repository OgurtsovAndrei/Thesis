package testutils

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// SeriesData describes a single data series on a tradeoff plot.
type SeriesData struct {
	Name   string
	Color  string
	Dashed bool
	Marker string // "circle", "square", "diamond", "triangle", "star"
	Points []Point
}

// Point is an (X, Y) data point.
type Point struct {
	X, Y float64
}

// AxisScale selects linear or log10 scaling for an axis.
type AxisScale int

const (
	Linear AxisScale = iota
	Log10
)

// PlotConfig controls axes and layout of a generated SVG.
type PlotConfig struct {
	Title  string
	XLabel string
	YLabel string
	XScale AxisScale
	YScale AxisScale
}

// GeneratePerformanceSVG creates an SVG plot with configurable axis scales.
func GeneratePerformanceSVG(cfg PlotConfig, series []SeriesData, outPath string) error {
	w, h := 960.0, 600.0
	mL, mR, mT, mB := 90.0, 40.0, 40.0, 50.0
	plotW := w - mL - mR
	plotH := h - mT - mB

	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, s := range series {
		for _, p := range s.Points {
			if p.X < minX {
				minX = p.X
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.Y > maxY {
				maxY = p.Y
			}
		}
	}

	var xToPlot func(float64) float64
	var yToPlot func(float64) float64

	var xTicks []float64
	var xTickLabels []string
	var yTicks []float64
	var yTickLabels []string

	switch cfg.XScale {
	case Log10:
		lMin := math.Floor(math.Log10(minX))
		lMax := math.Ceil(math.Log10(maxX))
		xToPlot = func(x float64) float64 {
			if x <= 0 {
				x = minX
			}
			return mL + plotW*(math.Log10(x)-lMin)/(lMax-lMin)
		}
		for e := int(lMin); e <= int(lMax); e++ {
			xTicks = append(xTicks, math.Pow(10, float64(e)))
			xTickLabels = append(xTickLabels, fmtPow10(e))
		}
	default:
		padX := (maxX - minX) * 0.05
		axMinX := math.Max(0, minX-padX)
		axMaxX := maxX + padX
		xToPlot = func(x float64) float64 {
			return mL + plotW*(x-axMinX)/(axMaxX-axMinX)
		}
		step := niceStep(axMinX, axMaxX, 8)
		for v := math.Ceil(axMinX/step) * step; v <= axMaxX; v += step {
			xTicks = append(xTicks, v)
			xTickLabels = append(xTickLabels, fmtNum(v))
		}
	}

	switch cfg.YScale {
	case Log10:
		const floor = 1e-8
		for i := range series {
			for j := range series[i].Points {
				if series[i].Points[j].Y <= 0 {
					series[i].Points[j].Y = floor
				}
			}
		}
		if minY <= 0 {
			minY = floor
		}
		lMin := math.Floor(math.Log10(minY)) - 0.5
		lMax := math.Ceil(math.Log10(maxY)) + 0.5
		yToPlot = func(y float64) float64 {
			if y <= 0 {
				y = floor
			}
			return mT + plotH*(1-(math.Log10(y)-lMin)/(lMax-lMin))
		}
		for e := int(math.Ceil(lMin)); e <= int(math.Floor(lMax)); e++ {
			yTicks = append(yTicks, math.Pow(10, float64(e)))
			yTickLabels = append(yTickLabels, fmtPow10(e))
		}
	default:
		axMinY := 0.0
		axMaxY := maxY * 1.1
		yToPlot = func(y float64) float64 {
			return mT + plotH*(1-(y-axMinY)/(axMaxY-axMinY))
		}
		step := niceStep(axMinY, axMaxY, 6)
		for v := 0.0; v <= axMaxY; v += step {
			yTicks = append(yTicks, v)
			yTickLabels = append(yTickLabels, fmtNum(v))
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`+"\n", w, h, w, h))
	sb.WriteString(`<style>text{font-family:Menlo,Monaco,monospace;font-size:12px;fill:#222} .axis{stroke:#333;stroke-width:1} .grid{stroke:#eee;stroke-width:0.5} .label{font-size:11px;fill:#444}</style>` + "\n")
	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="28" text-anchor="middle" style="font-size:14px;font-weight:bold">%s</text>`+"\n", w/2, cfg.Title))

	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT+plotH, mL+plotW, mT+plotH))
	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT, mL, mT+plotH))

	for i, tv := range yTicks {
		py := yToPlot(tv)
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, py, mL+plotW, py))
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="end">%s</text>`+"\n", mL-8, py+4, yTickLabels[i]))
	}

	for i, tv := range xTicks {
		px := xToPlot(tv)
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", px, mT, px, mT+plotH))
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`+"\n", px, mT+plotH+16, xTickLabels[i]))
	}

	drawSeriesLines(&sb, series, xToPlot, yToPlot)

	drawLegend(&sb, series, mL, mT, plotW)

	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" text-anchor="middle">%s</text>`+"\n", mL+plotW/2, h-10, cfg.XLabel))
	sb.WriteString(fmt.Sprintf(`<text transform="translate(16,%.0f) rotate(-90)" text-anchor="middle">%s</text>`+"\n", mT+plotH/2, cfg.YLabel))
	sb.WriteString("</svg>\n")

	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}

func fmtPow10(e int) string {
	switch {
	case e == 0:
		return "1"
	case e == 1:
		return "10"
	case e == 2:
		return "100"
	case e == 3:
		return "1K"
	case e == 4:
		return "10K"
	case e == 5:
		return "100K"
	case e == 6:
		return "1M"
	default:
		return fmt.Sprintf("10^%d", e)
	}
}

func fmtNum(v float64) string {
	if v == 0 {
		return "0"
	}
	abs := math.Abs(v)
	switch {
	case abs >= 1e6:
		return fmt.Sprintf("%.0fM", v/1e6)
	case abs >= 1e3:
		return fmt.Sprintf("%.0fK", v/1e3)
	case abs >= 1:
		return fmt.Sprintf("%.0f", v)
	case abs >= 0.01:
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%.4f", v)
	}
}

func niceStep(min, max float64, targetTicks int) float64 {
	raw := (max - min) / float64(targetTicks)
	if raw <= 0 {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	norm := raw / mag
	var nice float64
	switch {
	case norm <= 1.5:
		nice = 1
	case norm <= 3.5:
		nice = 2
	case norm <= 7.5:
		nice = 5
	default:
		nice = 10
	}
	return nice * mag
}

func drawSeriesLines(sb *strings.Builder, series []SeriesData, toX, toY func(float64) float64) {
	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}
		var pts []string
		for _, p := range s.Points {
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", toX(p.X), toY(p.Y)))
		}
		dash := ""
		if s.Dashed {
			dash = ` stroke-dasharray="8,5"`
		}
		sb.WriteString(fmt.Sprintf(`<polyline fill="none" stroke="%s" stroke-width="2"%s points="%s"/>`+"\n",
			s.Color, dash, strings.Join(pts, " ")))
		marker := s.Marker
		if marker == "" {
			marker = "circle"
		}
		for _, p := range s.Points {
			drawMarker(sb, marker, s.Color, toX(p.X), toY(p.Y))
		}
	}
}

func drawLegend(sb *strings.Builder, series []SeriesData, mL, mT, plotW float64) {
	ly := mT + 20.0
	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}
		dash := ""
		if s.Dashed {
			dash = ` stroke-dasharray="8,5"`
		}
		lx := mL + plotW - 220
		sb.WriteString(fmt.Sprintf(`<line x1="%.0f" y1="%.0f" x2="%.0f" y2="%.0f" stroke="%s" stroke-width="2"%s/>`+"\n",
			lx, ly, lx+16, ly, s.Color, dash))
		marker := s.Marker
		if marker == "" {
			marker = "circle"
		}
		drawMarker(sb, marker, s.Color, lx+8, ly)
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.0f" y="%.0f">%s</text>`+"\n", lx+22, ly+4, s.Name))
		ly += 18
	}
}

func drawMarker(sb *strings.Builder, marker, color string, cx, cy float64) {
	switch marker {
	case "square":
		sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="6" height="6" fill="%s"/>`+"\n", cx-3, cy-3, color))
	case "diamond":
		sb.WriteString(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`+"\n",
			cx, cy-4, cx+4, cy, cx, cy+4, cx-4, cy, color))
	case "triangle":
		sb.WriteString(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`+"\n",
			cx, cy-4, cx+4, cy+3, cx-4, cy+3, color))
	case "star":
		r1, r2 := 5.0, 2.0
		var pts []string
		for i := 0; i < 10; i++ {
			angle := math.Pi/2 + float64(i)*math.Pi/5
			r := r1
			if i%2 == 1 {
				r = r2
			}
			px := cx + r*math.Cos(angle)
			py := cy - r*math.Sin(angle)
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", px, py))
		}
		sb.WriteString(fmt.Sprintf(`<polygon points="%s" fill="%s"/>`+"\n", strings.Join(pts, " "), color))
	default:
		sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="3" fill="%s"/>`+"\n", cx, cy, color))
	}
}

// GenerateTradeoffSVG creates an SVG plot with log-scale Y axis (FPR) and linear X axis (BPK).
func GenerateTradeoffSVG(title, xLabel, yLabel string, series []SeriesData, outPath string) error {
	return GeneratePerformanceSVG(PlotConfig{
		Title:  title,
		XLabel: xLabel,
		YLabel: yLabel,
		XScale: Linear,
		YScale: Log10,
	}, series, outPath)
}
