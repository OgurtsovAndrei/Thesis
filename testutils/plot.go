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

// GenerateTradeoffSVG creates an SVG plot with log-scale Y axis (FPR) and linear X axis (BPK).
func GenerateTradeoffSVG(title, xLabel, yLabel string, series []SeriesData, outPath string) error {
	w, h := 960.0, 600.0
	mL, mR, mT, mB := 90.0, 40.0, 40.0, 50.0
	plotW := w - mL - mR
	plotH := h - mT - mB

	const fprFloor = 1e-6
	for i := range series {
		for j := range series[i].Points {
			if series[i].Points[j].Y <= 0 {
				series[i].Points[j].Y = fprFloor
			}
		}
	}

	minX, maxX := math.Inf(1), math.Inf(-1)
	minLogY, maxLogY := math.Inf(1), math.Inf(-1)
	for _, s := range series {
		for _, p := range s.Points {
			if p.X < minX {
				minX = p.X
			}
			if p.X > maxX {
				maxX = p.X
			}
			ly := math.Log10(p.Y)
			if ly < minLogY {
				minLogY = ly
			}
			if ly > maxLogY {
				maxLogY = ly
			}
		}
	}

	minX = 0
	maxX = math.Ceil(maxX/2) * 2
	minLogY = math.Floor(minLogY) - 0.5
	maxLogY = math.Ceil(maxLogY) + 0.5

	toX := func(x float64) float64 { return mL + plotW*(x-minX)/(maxX-minX) }
	toY := func(y float64) float64 {
		ly := math.Log10(y)
		return mT + plotH*(1-(ly-minLogY)/(maxLogY-minLogY))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`+"\n", w, h, w, h))
	sb.WriteString(`<style>text{font-family:Menlo,Monaco,monospace;font-size:12px;fill:#222} .axis{stroke:#333;stroke-width:1} .grid{stroke:#eee;stroke-width:0.5} .label{font-size:11px;fill:#444}</style>` + "\n")

	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="28" text-anchor="middle" style="font-size:14px;font-weight:bold">%s</text>`+"\n", w/2, title))

	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT+plotH, mL+plotW, mT+plotH))
	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT, mL, mT+plotH))

	for e := int(math.Ceil(minLogY)); e <= int(math.Floor(maxLogY)); e++ {
		py := mT + plotH*(1-(float64(e)-minLogY)/(maxLogY-minLogY))
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, py, mL+plotW, py))
		var label string
		switch {
		case e == 0:
			label = "1"
		case e == -1:
			label = "0.1"
		case e == -2:
			label = "0.01"
		case e == -3:
			label = "10^-3"
		default:
			label = fmt.Sprintf("10^%d", e)
		}
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="end">%s</text>`+"\n", mL-8, py+4, label))
	}

	xStep := 2.0
	for x := math.Ceil(minX/xStep) * xStep; x <= maxX; x += xStep {
		px := toX(x)
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", px, mT, px, mT+plotH))
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="middle">%.0f</text>`+"\n", px, mT+plotH+16, x))
	}

	drawMarker := func(sb *strings.Builder, marker, color string, cx, cy float64) {
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

	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}
		var validPts []Point
		for _, p := range s.Points {
			if p.Y <= 0 || p.X < 0.1 {
				continue
			}
			validPts = append(validPts, p)
		}
		if len(validPts) == 0 {
			continue
		}
		var pts []string
		for _, p := range validPts {
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
		for _, p := range validPts {
			drawMarker(&sb, marker, s.Color, toX(p.X), toY(p.Y))
		}
	}

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
		drawMarker(&sb, marker, s.Color, lx+8, ly)
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.0f" y="%.0f">%s</text>`+"\n", lx+22, ly+4, s.Name))
		ly += 18
	}

	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" text-anchor="middle">%s</text>`+"\n", mL+plotW/2, h-10, xLabel))
	sb.WriteString(fmt.Sprintf(`<text transform="translate(16,%.0f) rotate(-90)" text-anchor="middle">%s</text>`+"\n", mT+plotH/2, yLabel))
	sb.WriteString("</svg>\n")

	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}
