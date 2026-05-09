package testutils

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// Theme controls all visual rendering parameters for SVG plots.
// Zero value falls back to DefaultTheme().
type Theme struct {
	// Font sizes (px)
	FontBase  int // axis titles (XLabel, YLabel), base text
	FontLabel int // tick values, legend series names
	FontTitle int // plot title
	FontHint  int // muted annotations: floor line, threshold labels, end captions

	// Stroke widths (px)
	StrokeSeries float64 // data series polylines
	StrokeAxis   float64 // axis border lines
	StrokeGrid   float64 // grid lines
	StrokeFloor  float64 // floor / threshold dashed lines

	// Markers
	MarkerScale float64 // size multiplier applied to all marker shapes (1.0 = original)

	// Legend geometry
	LegendLineLen float64 // length of the colour-swatch line in the legend
	LegendSpacing float64 // vertical pitch between legend entries
}

// DefaultTheme returns the standard theme used for all defence / thesis plots.
func DefaultTheme() Theme {
	return Theme{
		FontBase:  22,
		FontLabel: 20,
		FontTitle: 24,
		FontHint:  17,

		StrokeSeries: 2.5,
		StrokeAxis:   1.5,
		StrokeGrid:   0.5,
		StrokeFloor:  1.5,

		MarkerScale: 1.2,

		LegendLineLen: 20,
		LegendSpacing: 24,
	}
}

// resolve fills any zero field with the corresponding DefaultTheme value.
func (t Theme) resolve() Theme {
	d := DefaultTheme()
	if t.FontBase == 0 {
		t.FontBase = d.FontBase
	}
	if t.FontLabel == 0 {
		t.FontLabel = d.FontLabel
	}
	if t.FontTitle == 0 {
		t.FontTitle = d.FontTitle
	}
	if t.FontHint == 0 {
		t.FontHint = d.FontHint
	}
	if t.StrokeSeries == 0 {
		t.StrokeSeries = d.StrokeSeries
	}
	if t.StrokeAxis == 0 {
		t.StrokeAxis = d.StrokeAxis
	}
	if t.StrokeGrid == 0 {
		t.StrokeGrid = d.StrokeGrid
	}
	if t.StrokeFloor == 0 {
		t.StrokeFloor = d.StrokeFloor
	}
	if t.MarkerScale == 0 {
		t.MarkerScale = d.MarkerScale
	}
	if t.LegendLineLen == 0 {
		t.LegendLineLen = d.LegendLineLen
	}
	if t.LegendSpacing == 0 {
		t.LegendSpacing = d.LegendSpacing
	}
	return t
}

// SeriesData describes a single data series on a tradeoff plot.
type SeriesData struct {
	Name   string
	Color  string
	Dashed bool
	Marker string // "circle", "square", "diamond", "triangle", "star"
	Points []Point

	// NoLine, when true, suppresses the connecting polyline so the series
	// renders as a marker-only point cloud. Use for families whose points
	// span a 2D parameter space and so cannot be meaningfully ordered along
	// a single curve (e.g. SuRF across (suffixType, bitCount)).
	NoLine bool

	// EndStop, when true, replaces the last marker with a small X (cross)
	// signalling that the series terminates here for an external reason
	// (e.g. an underlying library cannot operate beyond this point) rather
	// than just having no more measurements. The X is drawn on top of the
	// regular line+markers so it always reads as an explicit terminator.
	EndStop bool
	// EndCaption is rendered below the series name in the legend with a
	// matching small X marker, in italic muted style. Empty = no caption.
	EndCaption string
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

// PlotConfig controls axes, layout, and visual theme of a generated SVG.
type PlotConfig struct {
	Title    string
	XLabel   string
	YLabel   string
	XScale   AxisScale
	YScale   AxisScale
	YFloor   float64 // if >0 and YScale==Log10, draws a measurement floor line at this value
	YCeil    float64 // if >0 and YScale==Log10, hard-cap the upper axis at this value (clips data above)
	XMax     float64 // if >0 and using linear X-scale, hard-cap the X-axis at this value
	Theme    Theme   // zero value uses DefaultTheme()
}

// GeneratePerformanceSVG creates an SVG plot with configurable axis scales.
func GeneratePerformanceSVG(cfg PlotConfig, series []SeriesData, outPath string) error {
	t := cfg.Theme.resolve()

	hasThresholds := cfg.YScale == Log10 && cfg.YFloor > 0
	thresholdH := 0.0
	if hasThresholds {
		thresholdH = 100.0
	}
	w, h := 1020.0, 600.0+thresholdH
	mL, mR, mT, mB := 90.0, 160.0, 40.0, 50.0+thresholdH
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
		lMax := math.Log10(maxX)
		if lMax == math.Floor(lMax) {
			lMax += 0.1
		}
		lMaxCeil := math.Ceil(lMax)
		xToPlot = func(x float64) float64 {
			if x <= 0 {
				x = minX
			}
			return mL + plotW*(math.Log10(x)-lMin)/(lMaxCeil-lMin)
		}
		for e := int(lMin); e <= int(lMaxCeil); e++ {
			v := math.Pow(10, float64(e))
			if v > maxX*1.5 {
				break
			}
			xTicks = append(xTicks, v)
			xTickLabels = append(xTickLabels, fmtPow10(e))
		}
	default:
		padX := (maxX - minX) * 0.05
		axMinX := math.Max(0, minX-padX)
		axMaxX := maxX + padX
		if cfg.XMax > 0 && axMaxX > cfg.XMax {
			axMaxX = cfg.XMax
		}
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
		floor := 1e-8
		if cfg.YFloor > 0 {
			floor = cfg.YFloor
		}
		for i := range series {
			hitFloor := false
			for j := range series[i].Points {
				if series[i].Points[j].Y <= floor {
					series[i].Points[j].Y = floor
					if hitFloor {
						// Trim: keep only first floor point
						series[i].Points = series[i].Points[:j]
						break
					}
					hitFloor = true
				}
			}
		}
		if minY <= 0 || minY < floor {
			// minY was computed before clamping; cap at floor so the axis
			// doesn't extend into orders of magnitude with no plotted data
			// (e.g. theoretical curves at high K with FPR ≈ 2^-K).
			minY = floor
		}
		if cfg.YCeil > 0 && maxY > cfg.YCeil {
			maxY = cfg.YCeil
		}
		// Compute axis bounds.
		//
		// YFloor mode (FPR plots): snap to decade boundaries and give the
		// floor dashed line one extra decade of breathing room.
		//
		// Normal mode (no floor): adapt both bounds to the actual data
		// range so the series fill most of the plot area. Use 10 % of the
		// log-span as padding on each side, with a floor of 0.15 decades
		// so widely-separated points still have some breathing room.
		var lMin, lMax float64
		if cfg.YFloor > 0 {
			lMin = math.Floor(math.Log10(minY)) - 1.0
			lMax = math.Ceil(math.Log10(maxY)) + 0.5
		} else {
			logSpan := math.Log10(maxY) - math.Log10(minY)
			pad := math.Max(0.15, logSpan*0.10)
			lMin = math.Log10(minY) - pad
			lMax = math.Log10(maxY) + pad
		}
		yToPlot = func(y float64) float64 {
			if y <= 0 {
				y = floor
			}
			return mT + plotH*(1-(math.Log10(y)-lMin)/(lMax-lMin))
		}
		// Decade ticks always. When the visible range is narrow (< 2
		// decades) also add 2× and 5× sub-decade labels so there are
		// enough reference lines to read off values.
		axLo := math.Pow(10, lMin)
		axHi := math.Pow(10, lMax)
		type yTick struct {
			v     float64
			label string
		}
		var rawTicks []yTick
		eMin := int(math.Floor(lMin))
		eMax := int(math.Ceil(lMax)) + 1
		for e := eMin; e <= eMax; e++ {
			base := math.Pow(10, float64(e))
			if base >= axLo && base <= axHi {
				rawTicks = append(rawTicks, yTick{base, fmtPow10(e)})
			}
			if lMax-lMin < 2.0 {
				for _, mult := range []float64{2, 5} {
					v := mult * math.Pow(10, float64(e))
					if v > axLo && v < axHi {
						rawTicks = append(rawTicks, yTick{v, fmtNum(v)})
					}
				}
			}
		}
		sort.Slice(rawTicks, func(i, j int) bool { return rawTicks[i].v < rawTicks[j].v })
		for _, tk := range rawTicks {
			yTicks = append(yTicks, tk.v)
			yTickLabels = append(yTickLabels, tk.label)
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
	sb.WriteString(fmt.Sprintf(`<defs><clipPath id="plot-region"><rect x="%.1f" y="%.1f" width="%.1f" height="%.1f"/></clipPath></defs>`+"\n", mL, mT, plotW, plotH))
	sb.WriteString(fmt.Sprintf(
		`<style>text{font-family:Menlo,Monaco,monospace;font-size:%dpx;fill:#222} `+
			`.axis{stroke:#333;stroke-width:%.1f} `+
			`.grid{stroke:#eee;stroke-width:%.1f} `+
			`.label{font-size:%dpx;fill:#444}</style>`+"\n",
		t.FontBase, t.StrokeAxis, t.StrokeGrid, t.FontLabel))
	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="30" text-anchor="middle" style="font-size:%dpx;font-weight:bold">%s</text>`+"\n",
		w/2, t.FontTitle, cfg.Title))

	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT+plotH, mL+plotW, mT+plotH))
	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT, mL, mT+plotH))

	tickGap := float64(t.FontLabel) + 4
	for i, tv := range yTicks {
		py := yToPlot(tv)
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, py, mL+plotW, py))
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="end">%s</text>`+"\n", mL-8, py+4, yTickLabels[i]))
	}

	for i, tv := range xTicks {
		px := xToPlot(tv)
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", px, mT, px, mT+plotH))
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`+"\n", px, mT+plotH+tickGap, xTickLabels[i]))
	}

	// Measurement floor line (log-scale Y only)
	if cfg.YScale == Log10 && cfg.YFloor > 0 {
		py := yToPlot(cfg.YFloor)
		sb.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#999" stroke-width="%.1f" stroke-dasharray="6,4"/>`+"\n",
			mL, py, mL+plotW, py, t.StrokeFloor))
		sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="end" style="font-size:%dpx;fill:#999">0 FP observed (floor = %.0e)</text>`+"\n",
			mL+plotW, py-4, t.FontHint, cfg.YFloor))
	}

	sb.WriteString(`<g clip-path="url(#plot-region)">` + "\n")
	drawSeriesLines(&sb, t, series, xToPlot, yToPlot)
	sb.WriteString("</g>\n")

	drawLegend(&sb, t, series, mL, mT, plotW)

	axisLabelY := mT + plotH + tickGap + float64(t.FontBase) + 4
	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" text-anchor="middle" style="font-size:%dpx">%s</text>`+"\n",
		mL+plotW/2, axisLabelY, t.FontBase, cfg.XLabel))
	sb.WriteString(fmt.Sprintf(`<text transform="translate(%.0f,%.0f) rotate(-90)" text-anchor="middle" style="font-size:%dpx">%s</text>`+"\n",
		float64(t.FontBase), mT+plotH/2, t.FontBase, cfg.YLabel))

	// Threshold sub-chart: 3 number lines showing BPK where each filter hits 10^-2, 10^-3, 0 FP
	if hasThresholds {
		thresholds := []struct {
			label string
			value float64
		}{
			{"FPR ≤ 10⁻²", 0.01},
			{"FPR ≤ 10⁻³", 0.001},
			{"0 FP", cfg.YFloor},
		}
		subTop := mT + plotH + 60.0
		lineSpacing := 28.0

		for ti, thr := range thresholds {
			ly := subTop + float64(ti)*lineSpacing
			// Draw axis line
			sb.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#ccc" stroke-width="1"/>`+"\n",
				mL, ly, mL+plotW, ly))
			// Label
			sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="end" style="font-size:%dpx;fill:#666">%s</text>`+"\n",
				mL-8, ly+3, t.FontHint, thr.label))
			// Find first X where each series reaches this threshold
			for _, s := range series {
				if len(s.Points) == 0 {
					continue
				}
				bpk := findThresholdBPK(s.Points, thr.value)
				if bpk < 0 {
					continue
				}
				px := xToPlot(bpk)
				// Skip markers that would land past the visible plot region.
				// Without this, e.g. BloomARE on small-FPR distributions
				// hits the threshold at BPK > XMax and the marker bleeds
				// past the right edge of the chart.
				if px < mL || px > mL+plotW {
					continue
				}
				marker := s.Marker
				if marker == "" {
					marker = "circle"
				}
				sb.WriteString(fmt.Sprintf(`<g transform="translate(%.1f,%.1f) scale(1.5) translate(%.1f,%.1f)">`, px, ly, -px, -ly))
				drawMarker(&sb, t, marker, s.Color, px, ly)
				sb.WriteString("</g>\n")
			}
		}
	}

	sb.WriteString("</svg>\n")

	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}

// findThresholdBPK returns the X (BPK) where the series crosses Y = threshold,
// using log-linear interpolation between the last point above and first point
// at/below the threshold. Points are assumed sorted by X (ascending BPK).
// Returns -1 if the threshold is never reached.
func findThresholdBPK(points []Point, threshold float64) float64 {
	for i, p := range points {
		if p.Y <= threshold {
			if i == 0 || points[i-1].Y <= threshold {
				return p.X
			}
			prev := points[i-1]
			logPrev := math.Log(prev.Y)
			logCur := math.Log(p.Y)
			logThr := math.Log(threshold)
			t := (logThr - logPrev) / (logCur - logPrev)
			return prev.X + t*(p.X-prev.X)
		}
	}
	return -1
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

func drawSeriesLines(sb *strings.Builder, t Theme, series []SeriesData, toX, toY func(float64) float64) {
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
		if !s.NoLine {
			sb.WriteString(fmt.Sprintf(`<polyline fill="none" stroke="%s" stroke-width="%.1f"%s points="%s"/>`+"\n",
				s.Color, t.StrokeSeries, dash, strings.Join(pts, " ")))
		}
		marker := s.Marker
		if marker == "" {
			marker = "circle"
		}
		if marker != "none" {
			lastIdx := len(s.Points) - 1
			for i, p := range s.Points {
				if s.EndStop && i == lastIdx {
					continue // last marker drawn separately as X (after the loop)
				}
				drawMarker(sb, t, marker, s.Color, toX(p.X), toY(p.Y))
			}
		}
	}
	// Draw EndStop X markers last so they render on top of any neighbouring lines.
	for _, s := range series {
		if !s.EndStop || len(s.Points) == 0 {
			continue
		}
		last := s.Points[len(s.Points)-1]
		arm := 3.0 * t.MarkerScale
		drawXMarker(sb, s.Color, toX(last.X), toY(last.Y), arm, t.StrokeSeries)
	}
}

// drawXMarker writes an X (cross) at (cx,cy) with given half-arm length and stroke width.
func drawXMarker(sb *strings.Builder, color string, cx, cy, arm, stroke float64) {
	sb.WriteString(fmt.Sprintf(
		`<g transform="translate(%.1f,%.1f)">`+
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f" stroke-linecap="round"/>`+
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="%.1f" stroke-linecap="round"/>`+
			`</g>`+"\n",
		cx, cy, -arm, -arm, arm, arm, color, stroke, -arm, arm, arm, -arm, color, stroke))
}

func drawLegend(sb *strings.Builder, t Theme, series []SeriesData, mL, mT, plotW float64) {
	lx := mL + plotW + 16
	ly := mT + 20.0
	ll := t.LegendLineLen
	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}
		dash := ""
		if s.Dashed {
			dash = fmt.Sprintf(` stroke-dasharray="8,5"`)
		}
		if !s.NoLine {
			sb.WriteString(fmt.Sprintf(`<line x1="%.0f" y1="%.0f" x2="%.0f" y2="%.0f" stroke="%s" stroke-width="%.1f"%s/>`+"\n",
				lx, ly, lx+ll, ly, s.Color, t.StrokeSeries, dash))
		}
		marker := s.Marker
		if marker == "" {
			marker = "circle"
		}
		drawMarker(sb, t, marker, s.Color, lx+ll/2, ly)
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.0f" y="%.0f">%s</text>`+"\n", lx+ll+6, ly+4, s.Name))
		ly += t.LegendSpacing
		// Optional secondary caption (e.g. "(library limit)") with a small
		// X marker, signalling that the series carries an EndStop terminator.
		if s.EndCaption != "" {
			capY := ly - 6
			arm := 3.0 * t.MarkerScale
			drawXMarker(sb, s.Color, lx+ll/2, capY, arm, t.StrokeSeries)
			sb.WriteString(fmt.Sprintf(
				`<text x="%.0f" y="%.0f" style="font-size:%dpx;fill:#666;font-style:italic">%s</text>`+"\n",
				lx+ll+6, capY+4, t.FontHint, s.EndCaption))
			ly += float64(t.FontHint) + 4
		}
	}
}

func drawMarker(sb *strings.Builder, t Theme, marker, color string, cx, cy float64) {
	s := t.MarkerScale
	switch marker {
	case "square":
		sz := 6 * s
		sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`+"\n",
			cx-sz/2, cy-sz/2, sz, sz, color))
	case "diamond":
		r := 4 * s
		sb.WriteString(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`+"\n",
			cx, cy-r, cx+r, cy, cx, cy+r, cx-r, cy, color))
	case "triangle":
		r := 4 * s
		sb.WriteString(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`+"\n",
			cx, cy-r, cx+r*1.1, cy+r*0.8, cx-r*1.1, cy+r*0.8, color))
	case "star":
		r1, r2 := 5*s, 2*s
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
	default: // "circle"
		r := 3 * s
		sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", cx, cy, r, color))
	}
}

// GenerateTradeoffSVG creates an SVG plot with log-scale Y axis (FPR) and linear X axis (BPK).
// Optional yFloor parameter: if provided, draws a measurement floor line at that Y value.
func GenerateTradeoffSVG(title, xLabel, yLabel string, series []SeriesData, outPath string, yFloor ...float64) error {
	var fl float64
	if len(yFloor) > 0 {
		fl = yFloor[0]
	}
	return GeneratePerformanceSVG(PlotConfig{
		Title:  title,
		XLabel: xLabel,
		YLabel: yLabel,
		XScale: Linear,
		YScale: Log10,
		YFloor: fl,
		XMax:   25,
	}, series, outPath)
}
