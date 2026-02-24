import math
import os
from typing import Dict, List, Tuple

def ensure_dir(path: str) -> None:
    os.makedirs(path, exist_ok=True)

def svg_start(width: float, height: float) -> List[str]:
    return [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<style>text{font-family:Menlo,Monaco,monospace;font-size:12px;fill:#222} .axis{stroke:#333;stroke-width:1} .grid{stroke:#ddd;stroke-width:1} .label{font-size:11px;fill:#444}</style>',
    ]

def svg_finish(parts: List[str], path: str) -> None:
    parts.append("</svg>")
    ensure_dir(os.path.dirname(path))
    with open(path, "w") as f:
        f.write("\n".join(parts))

def draw_line_chart(path: str, title: str, x_label: str, y_label: str, series: Dict[str, List[Tuple[float, float]]], log_x: bool = False, log_y: bool = False) -> None:
    width, height = 960.0, 540.0
    left, right, top, bottom = 90.0, 40.0, 55.0, 75.0
    pw = width - left - right
    ph = height - top - bottom

    x_vals = sorted({x for pts in series.values() for x, _ in pts})
    if not x_vals:
        return

    y_values = [y for pts in series.values() for _, y in pts]
    if not y_values:
        y_values = [1.0]

    # Handle Y range
    if log_y:
        # Filter non-positive values for log scale
        pos_y = [y for y in y_values if y > 0]
        if not pos_y: pos_y = [1.0]
        y_min_val = min(pos_y)
        y_max_val = max(pos_y)
        
        y_min = float(math.floor(math.log10(y_min_val))) if y_min_val > 0 else 0.0
        y_max = float(math.ceil(math.log10(y_max_val * 1.1)))
        if y_max <= y_min: y_max = y_min + 1.0
    else:
        y_min = 0.0
        y_max = max(1.0, max(y_values) * 1.1)

    # Coordinate mappers
    def x_pos(x: float) -> float:
        x_min_val = min(x_vals)
        x_max_val = max(x_vals)
        if x_max_val == x_min_val:
            return left + pw / 2
        
        if log_x:
            if x <= 0: x = 1.0 # Avoid log(0)
            t = (math.log2(x) - math.log2(x_min_val)) / (math.log2(x_max_val) - math.log2(x_min_val))
        else:
            t = (x - x_min_val) / (x_max_val - x_min_val)
        return left + t * pw

    def y_pos(y: float) -> float:
        if log_y:
            if y <= 0: return top + ph
            val = math.log10(y)
            t = (val - y_min) / (y_max - y_min)
        else:
            t = (y - y_min) / (y_max - y_min)
        return top + ph - t * ph

    parts = svg_start(width, height)
    parts.append(f'<text x="{width/2}" y="26" text-anchor="middle">{title}</text>')
    
    # Axes
    parts.append(f'<line class="axis" x1="{left}" y1="{top+ph}" x2="{left+pw}" y2="{top+ph}" />')
    parts.append(f'<line class="axis" x1="{left}" y1="{top}" x2="{left}" y2="{top+ph}" />')

    # Y Grid
    if log_y:
        for p in range(int(y_min), int(y_max) + 1):
            yv = 10.0**p
            py = y_pos(yv)
            if top <= py <= top + ph + 0.1:
                parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
                parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">10^{p}</text>')
    else:
        for i in range(6):
            yv = y_max * i / 5
            py = y_pos(yv)
            parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
            parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">{yv:.2f}</text>')

    # X Grid
    for x in x_vals:
        px = x_pos(x)
        parts.append(f'<line class="grid" x1="{px:.2f}" y1="{top}" x2="{px:.2f}" y2="{top+ph}" />')
        label = f"{x:,.0f}" if x < 1000000 else f"{x:.1e}"
        parts.append(f'<text class="label" x="{px:.2f}" y="{top+ph+20}" text-anchor="middle">{label}</text>')

    # Data
    palette = ["#2a7fff", "#e4572e", "#22a06b", "#7c3aed", "#a16207", "#d946ef", "#0ea5e9"]
    legend_x = left + 10
    legend_y = top + 12
    
    for idx, (name, pts) in enumerate(series.items()):
        color = palette[idx % len(palette)]
        if not pts:
            continue
        pts = sorted(pts, key=lambda t: t[0])
        coords = " ".join(f"{x_pos(x):.2f},{y_pos(y):.2f}" for x, y in pts)
        parts.append(f'<polyline fill="none" stroke="{color}" stroke-width="2.5" points="{coords}" />')
        for x, y in pts:
            parts.append(f'<circle cx="{x_pos(x):.2f}" cy="{y_pos(y):.2f}" r="3.5" fill="{color}" />')
        
        # Legend
        ly = legend_y + idx * 18
        parts.append(f'<line x1="{legend_x}" y1="{ly}" x2="{legend_x+16}" y2="{ly}" stroke="{color}" stroke-width="2.5" />')
        parts.append(f'<text class="label" x="{legend_x+22}" y="{ly+4}">{name}</text>')

    parts.append(f'<text class="label" x="{width/2}" y="{height-18}" text-anchor="middle">{x_label}</text>')
    parts.append(f'<text class="label" transform="translate(20,{height/2}) rotate(-90)" text-anchor="middle">{y_label}</text>')
    svg_finish(parts, path)

def draw_stacked_area_chart(path: str, title: str, x_label: str, y_label: str, series: Dict[str, List[Tuple[float, float]]], log_x: bool = False) -> None:
    """Draws a stacked area chart. Series order in dictionary determines stacking order (bottom to top)."""
    width, height = 960.0, 540.0
    left, right, top, bottom = 90.0, 180.0, 55.0, 75.0 # More space on right for legend
    pw = width - left - right
    ph = height - top - bottom

    all_x = sorted({x for pts in series.values() for x, _ in pts})
    if not all_x:
        return

    # Prepare stacked data
    stacked_y: Dict[float, List[float]] = {x: [0.0] * len(series) for x in all_x}
    series_names = list(series.keys())
    
    for i, name in enumerate(series_names):
        points = {x: y for x, y in series[name]}
        for x in all_x:
            y_val = points.get(x, 0.0)
            # Each point is cumulative
            prev_y = stacked_y[x][i-1] if i > 0 else 0.0
            stacked_y[x][i] = prev_y + y_val

    max_y = max(vals[-1] for vals in stacked_y.values()) if stacked_y else 1.0
    y_max = max_y * 1.1

    # Coordinate mappers
    def x_pos(x: float) -> float:
        x_min_val, x_max_val = min(all_x), max(all_x)
        if x_max_val == x_min_val: return left + pw / 2
        if log_x:
            t = (math.log2(max(1, x)) - math.log2(max(1, x_min_val))) / (math.log2(max(1, x_max_val)) - math.log2(max(1, x_min_val)))
        else:
            t = (x - x_min_val) / (x_max_val - x_min_val)
        return left + t * pw

    def y_pos(y: float) -> float:
        t = y / y_max
        return top + ph - t * ph

    parts = svg_start(width, height)
    parts.append(f'<text x="{width/2}" y="26" text-anchor="middle">{title}</text>')
    
    # Axes
    parts.append(f'<line class="axis" x1="{left}" y1="{top+ph}" x2="{left+pw}" y2="{top+ph}" />')
    parts.append(f'<line class="axis" x1="{left}" y1="{top}" x2="{left}" y2="{top+ph}" />')

    # Y Grid
    for i in range(6):
        yv = y_max * i / 5
        py = y_pos(yv)
        parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
        label = f"{yv:,.1f}" if y_max < 100 else f"{yv:,.0f}"
        parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">{label}</text>')

    # X Grid
    for x in all_x:
        px = x_pos(x)
        parts.append(f'<line class="grid" x1="{px:.2f}" y1="{top}" x2="{px:.2f}" y2="{top+ph}" />')
        label = f"{x:,.0f}" if x < 1000000 else f"{x:.1e}"
        parts.append(f'<text class="label" x="{px:.2f}" y="{top+ph+20}" text-anchor="middle">{label}</text>')

    # Data - Draw areas (reverse order to get correct stacking visual if transparency used, 
    # but here we draw full polygons from top to bottom or bottom to top)
    palette = ["#2a7fff", "#e4572e", "#22a06b", "#7c3aed", "#a16207", "#d946ef", "#0ea5e9"]
    
    for i in range(len(series_names) - 1, -1, -1):
        color = palette[i % len(palette)]
        name = series_names[i]
        
        # Create polygon points for the area
        # Top boundary
        points = []
        for x in all_x:
            points.append((x_pos(x), y_pos(stacked_y[x][i])))
        
        # Bottom boundary (top of previous series)
        if i > 0:
            for x in reversed(all_x):
                points.append((x_pos(x), y_pos(stacked_y[x][i-1])))
        else:
            # Bottom of chart
            points.append((x_pos(all_x[-1]), top + ph))
            points.append((x_pos(all_x[0]), top + ph))
            
        poly_str = " ".join(f"{px:.2f},{py:.2f}" for px, py in points)
        parts.append(f'<polygon fill="{color}" fill-opacity="0.8" points="{poly_str}" />')

    # Legend (reversed to match stack order visually)
    legend_x = left + pw + 10
    for i, name in enumerate(reversed(series_names)):
        idx = len(series_names) - 1 - i
        color = palette[idx % len(palette)]
        ly = top + 12 + i * 20
        parts.append(f'<rect x="{legend_x}" y="{ly-8}" width="12" height="12" fill="{color}" />')
        parts.append(f'<text class="label" x="{legend_x+18}" y="{ly+2}">{name}</text>')

    parts.append(f'<text class="label" x="{width/2}" y="{height-18}" text-anchor="middle">{x_label}</text>')
    parts.append(f'<text class="label" transform="translate(20,{height/2}) rotate(-90)" text-anchor="middle">{y_label}</text>')
    svg_finish(parts, path)
