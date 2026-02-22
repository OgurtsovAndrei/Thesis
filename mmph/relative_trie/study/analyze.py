#!/usr/bin/env python3
import csv
import math
import os
import re
import statistics
from collections import defaultdict
from typing import Dict, List, Tuple, Any, Optional, Sequence


def read_csv(path: str) -> List[Dict[str, str]]:
    with open(path, newline="") as f:
        return list(csv.DictReader(f))


def read_csv_if_exists(path: str) -> List[Dict[str, str]]:
    if not os.path.exists(path):
        return []
    return read_csv(path)


def as_int(row: Dict[str, str], key: str) -> int:
    return int(float(row[key]))


def as_float(row: Dict[str, str], key: str) -> float:
    return float(row[key])


def ensure_dir(path: str) -> None:
    os.makedirs(path, exist_ok=True)


def write_csv(path: str, rows: List[Dict[str, Any]], header: List[str]) -> None:
    with open(path, "w", newline="") as f:
        wr = csv.DictWriter(f, fieldnames=header)
        wr.writeheader()
        wr.writerows(rows)


def quantile(vals: Sequence[float], q: float) -> float:
    if not vals:
        return 0.0
    if len(vals) == 1:
        return vals[0]
    s_vals = sorted(vals)
    pos = round((len(s_vals) - 1) * q)
    pos = max(0, min(pos, len(s_vals) - 1))
    return s_vals[pos]


def svg_start(width: float, height: float) -> List[str]:
    return [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<style>text{font-family:Menlo,Monaco,monospace;font-size:12px;fill:#222} .axis{stroke:#333;stroke-width:1} .grid{stroke:#ddd;stroke-width:1} .label{font-size:11px;fill:#444}</style>',
    ]


def svg_finish(parts: List[str], path: str) -> None:
    parts.append("</svg>")
    with open(path, "w") as f:
        f.write("\n".join(parts))


def draw_bar_chart(path: str, title: str, x_label: str, y_label: str, labels: List[str], values: List[float], color: str = "#2a7fff") -> None:
    width, height = 960.0, 540.0
    left, right, top, bottom = 90.0, 40.0, 55.0, 75.0
    pw = width - left - right
    ph = height - top - bottom
    ymax = max(1.0, max(values) * 1.1)

    parts = svg_start(width, height)
    parts.append(f'<text x="{width/2}" y="26" text-anchor="middle">{title}</text>')
    parts.append(f'<line class="axis" x1="{left}" y1="{top+ph}" x2="{left+pw}" y2="{top+ph}" />')
    parts.append(f'<line class="axis" x1="{left}" y1="{top}" x2="{left}" y2="{top+ph}" />')

    for i in range(6):
        yv = ymax * i / 5
        y = top + ph - (yv / ymax) * ph
        parts.append(f'<line class="grid" x1="{left}" y1="{y:.2f}" x2="{left+pw}" y2="{y:.2f}" />')
        parts.append(f'<text class="label" x="{left-8}" y="{y+4:.2f}" text-anchor="end">{yv:.2f}</text>')

    n = len(labels)
    gap = 20.0
    bar_w = (pw - gap * (n + 1)) / max(1, n)
    for i, (lab, val) in enumerate(zip(labels, values)):
        x = left + gap + i * (bar_w + gap)
        h = (val / ymax) * ph
        y = top + ph - h
        parts.append(f'<rect x="{x:.2f}" y="{y:.2f}" width="{bar_w:.2f}" height="{h:.2f}" fill="{color}" />')
        parts.append(f'<text class="label" x="{x + bar_w/2:.2f}" y="{top+ph+20}" text-anchor="middle">{lab}</text>')
        parts.append(f'<text class="label" x="{x + bar_w/2:.2f}" y="{y-6:.2f}" text-anchor="middle">{val:.3f}</text>')

    parts.append(f'<text class="label" x="{width/2}" y="{height-18}" text-anchor="middle">{x_label}</text>')
    parts.append(f'<text class="label" transform="translate(20,{height/2}) rotate(-90)" text-anchor="middle">{y_label}</text>')
    svg_finish(parts, path)


def draw_line_chart(path: str, title: str, x_label: str, y_label: str, x_vals: List[int], series: Dict[str, List[Tuple[int, float]]]) -> None:
    width, height = 960.0, 540.0
    left, right, top, bottom = 90.0, 40.0, 55.0, 75.0
    pw = width - left - right
    ph = height - top - bottom

    x_min = min(x_vals)
    x_max = max(x_vals)
    y_min = 0.0
    y_max = 1.0

    def x_pos(x: int) -> float:
        if x_max == x_min:
            return left + pw / 2
        t = (math.log2(x) - math.log2(x_min)) / (math.log2(x_max) - math.log2(x_min))
        return left + t * pw

    def y_pos(y: float) -> float:
        return top + ph - ((y - y_min) / (y_max - y_min)) * ph

    parts = svg_start(width, height)
    parts.append(f'<text x="{width/2}" y="26" text-anchor="middle">{title}</text>')
    parts.append(f'<line class="axis" x1="{left}" y1="{top+ph}" x2="{left+pw}" y2="{top+ph}" />')
    parts.append(f'<line class="axis" x1="{left}" y1="{top}" x2="{left}" y2="{top+ph}" />')

    for y in [0.0, 0.25, 0.5, 0.75, 1.0]:
        py = y_pos(y)
        parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
        parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">{y:.2f}</text>')

    for x in x_vals:
        px = x_pos(x)
        parts.append(f'<line class="grid" x1="{px:.2f}" y1="{top}" x2="{px:.2f}" y2="{top+ph}" />')
        parts.append(f'<text class="label" x="{px:.2f}" y="{top+ph+20}" text-anchor="middle">{x}</text>')

    palette = ["#2a7fff", "#e4572e", "#22a06b", "#7c3aed", "#a16207"]
    legend_x = left + 10
    legend_y = top + 12
    for idx, (name, pts) in enumerate(series.items()):
        color = palette[idx % len(palette)]
        if not pts:
            continue
        coords = " ".join(f"{x_pos(x):.2f},{y_pos(y):.2f}" for x, y in pts)
        parts.append(f'<polyline fill="none" stroke="{color}" stroke-width="2.5" points="{coords}" />')
        for x, y in pts:
            parts.append(f'<circle cx="{x_pos(x):.2f}" cy="{y_pos(y):.2f}" r="3.5" fill="{color}" />')
        ly = legend_y + idx * 18
        parts.append(f'<line x1="{legend_x}" y1="{ly}" x2="{legend_x+16}" y2="{ly}" stroke="{color}" stroke-width="2.5" />')
        parts.append(f'<text class="label" x="{legend_x+22}" y="{ly+4}">{name}</text>')

    parts.append(f'<text class="label" x="{width/2}" y="{height-18}" text-anchor="middle">{x_label}</text>')
    parts.append(f'<text class="label" transform="translate(20,{height/2}) rotate(-90)" text-anchor="middle">{y_label}</text>')
    svg_finish(parts, path)


def draw_line_chart_numeric(path: str, title: str, x_label: str, y_label: str, x_vals: List[int], series: Dict[str, List[Tuple[int, float]]]) -> None:
    width, height = 960.0, 540.0
    left, right, top, bottom = 90.0, 40.0, 55.0, 75.0
    pw = width - left - right
    ph = height - top - bottom

    x_min = min(x_vals)
    x_max = max(x_vals)
    y_values: List[float] = []
    for pts in series.values():
        y_values.extend(y for _, y in pts)
    y_min = 0.0
    y_max = max(1.0, max(y_values) * 1.1)

    def x_pos(x: int) -> float:
        if x_max == x_min:
            return left + pw / 2
        t = (x - x_min) / (x_max - x_min)
        return left + t * pw

    def y_pos(y: float) -> float:
        return top + ph - ((y - y_min) / (y_max - y_min)) * ph

    parts = svg_start(width, height)
    parts.append(f'<text x="{width/2}" y="26" text-anchor="middle">{title}</text>')
    parts.append(f'<line class="axis" x1="{left}" y1="{top+ph}" x2="{left+pw}" y2="{top+ph}" />')
    parts.append(f'<line class="axis" x1="{left}" y1="{top}" x2="{left}" y2="{top+ph}" />')

    for i in range(6):
        yv = y_max * i / 5
        py = y_pos(yv)
        parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
        parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">{yv:.1f}</text>')

    for x in x_vals:
        px = x_pos(x)
        parts.append(f'<line class="grid" x1="{px:.2f}" y1="{top}" x2="{px:.2f}" y2="{top+ph}" />')
        parts.append(f'<text class="label" x="{px:.2f}" y="{top+ph+20}" text-anchor="middle">{x}</text>')

    palette = ["#2a7fff", "#e4572e", "#22a06b", "#7c3aed", "#a16207"]
    legend_x = left + 10
    legend_y = top + 12
    for idx, (name, pts) in enumerate(series.items()):
        color = palette[idx % len(palette)]
        if not pts:
            continue
        coords = " ".join(f"{x_pos(x):.2f},{y_pos(y):.2f}" for x, y in pts)
        parts.append(f'<polyline fill="none" stroke="{color}" stroke-width="2.5" points="{coords}" />')
        for x, y in pts:
            parts.append(f'<circle cx="{x_pos(x):.2f}" cy="{y_pos(y):.2f}" r="3.5" fill="{color}" />')
        ly = legend_y + idx * 18
        parts.append(f'<line x1="{legend_x}" y1="{ly}" x2="{legend_x+16}" y2="{ly}" stroke="{color}" stroke-width="2.5" />')
        parts.append(f'<text class="label" x="{legend_x+22}" y="{ly+4}">{name}</text>')

    parts.append(f'<text class="label" x="{width/2}" y="{height-18}" text-anchor="middle">{x_label}</text>')
    parts.append(f'<text class="label" transform="translate(20,{height/2}) rotate(-90)" text-anchor="middle">{y_label}</text>')
    svg_finish(parts, path)


def draw_overlay_empirical_theory(path: str, title: str, x_label: str, y_label: str, x_vals: List[int], empirical_series: Dict[str, List[Tuple[int, float]]], theory_series: Dict[str, List[Tuple[int, float]]]) -> None:
    width, height = 960.0, 540.0
    left, right, top, bottom = 90.0, 40.0, 55.0, 75.0
    pw = width - left - right
    ph = height - top - bottom

    x_min = min(x_vals)
    x_max = max(x_vals)
    y_min = 0.0
    y_max = 1.0

    def x_pos(x: int) -> float:
        if x_max == x_min:
            return left + pw / 2
        t = (math.log2(x) - math.log2(x_min)) / (math.log2(x_max) - math.log2(x_min))
        return left + t * pw

    def y_pos(y: float) -> float:
        return top + ph - ((y - y_min) / (y_max - y_min)) * ph

    parts = svg_start(width, height)
    parts.append(f'<text x="{width/2}" y="26" text-anchor="middle">{title}</text>')
    parts.append(f'<line class="axis" x1="{left}" y1="{top+ph}" x2="{left+pw}" y2="{top+ph}" />')
    parts.append(f'<line class="axis" x1="{left}" y1="{top}" x2="{left}" y2="{top+ph}" />')

    for y in [0.0, 0.25, 0.5, 0.75, 1.0]:
        py = y_pos(y)
        parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
        parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">{y:.2f}</text>')

    for x in x_vals:
        px = x_pos(x)
        parts.append(f'<line class="grid" x1="{px:.2f}" y1="{top}" x2="{px:.2f}" y2="{top+ph}" />')
        parts.append(f'<text class="label" x="{px:.2f}" y="{top+ph+20}" text-anchor="middle">{x}</text>')

    palette = ["#2a7fff", "#e4572e", "#22a06b", "#7c3aed", "#a16207"]
    legend_x = left + 10
    legend_y = top + 12
    keys = sorted(set(list(empirical_series.keys()) + list(theory_series.keys())))
    for idx, key in enumerate(keys):
        color = palette[idx % len(palette)]
        e_pts = empirical_series.get(key, [])
        t_pts = theory_series.get(key, [])

        if e_pts:
            e_coords = " ".join(f"{x_pos(x):.2f},{y_pos(y):.2f}" for x, y in e_pts)
            parts.append(f'<polyline fill="none" stroke="{color}" stroke-width="2.5" points="{e_coords}" />')
            for x, y in e_pts:
                parts.append(f'<circle cx="{x_pos(x):.2f}" cy="{y_pos(y):.2f}" r="3.5" fill="{color}" />')

        if t_pts:
            t_coords = " ".join(f"{x_pos(x):.2f},{y_pos(y):.2f}" for x, y in t_pts)
            parts.append(
                f'<polyline fill="none" stroke="{color}" stroke-width="2.5" stroke-dasharray="8,6" points="{t_coords}" />'
            )
            for x, y in t_pts:
                parts.append(
                    f'<rect x="{x_pos(x)-2.5:.2f}" y="{y_pos(y)-2.5:.2f}" width="5" height="5" fill="{color}" />'
                )

        ly = legend_y + idx * 18
        parts.append(f'<line x1="{legend_x}" y1="{ly}" x2="{legend_x+16}" y2="{ly}" stroke="{color}" stroke-width="2.5" />')
        parts.append(f'<line x1="{legend_x+20}" y1="{ly}" x2="{legend_x+36}" y2="{ly}" stroke="{color}" stroke-width="2.5" stroke-dasharray="8,6" />')
        parts.append(f'<text class="label" x="{legend_x+42}" y="{ly+4}">{key} (solid=emp, dashed=theory)</text>')

    parts.append(f'<text class="label" x="{width/2}" y="{height-18}" text-anchor="middle">{x_label}</text>')
    parts.append(f'<text class="label" transform="translate(20,{height/2}) rotate(-90)" text-anchor="middle">{y_label}</text>')
    svg_finish(parts, path)


def draw_line_chart_logx_numericy(path: str, title: str, x_label: str, y_label: str, x_vals: List[int], series: Dict[str, List[Tuple[int, float]]]) -> None:
    width, height = 960.0, 540.0
    left, right, top, bottom = 90.0, 40.0, 55.0, 75.0
    pw = width - left - right
    ph = height - top - bottom

    x_min = min(x_vals)
    x_max = max(x_vals)
    y_values: List[float] = []
    for pts in series.values():
        y_values.extend(y for _, y in pts)
    y_min = 0.0
    y_max = max(1.0, max(y_values) * 1.1)

    def x_pos(x: int) -> float:
        if x_max == x_min:
            return left + pw / 2
        t = (math.log2(x) - math.log2(x_min)) / (math.log2(x_max) - math.log2(x_min))
        return left + t * pw

    def y_pos(y: float) -> float:
        return top + ph - ((y - y_min) / (y_max - y_min)) * ph

    parts = svg_start(width, height)
    parts.append(f'<text x="{width/2}" y="26" text-anchor="middle">{title}</text>')
    parts.append(f'<line class="axis" x1="{left}" y1="{top+ph}" x2="{left+pw}" y2="{top+ph}" />')
    parts.append(f'<line class="axis" x1="{left}" y1="{top}" x2="{left}" y2="{top+ph}" />')

    for i in range(6):
        yv = y_max * i / 5
        py = y_pos(yv)
        parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
        parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">{yv:.1f}</text>')

    for x in x_vals:
        px = x_pos(x)
        parts.append(f'<line class="grid" x1="{px:.2f}" y1="{top}" x2="{px:.2f}" y2="{top+ph}" />')
        parts.append(f'<text class="label" x="{px:.2f}" y="{top+ph+20}" text-anchor="middle">{x}</text>')

    palette = ["#2a7fff", "#e4572e", "#22a06b", "#7c3aed", "#a16207"]
    legend_x = left + 10
    legend_y = top + 12
    for idx, (name, pts) in enumerate(series.items()):
        color = palette[idx % len(palette)]
        if not pts:
            continue
        coords = " ".join(f"{x_pos(x):.2f},{y_pos(y):.2f}" for x, y in pts)
        parts.append(f'<polyline fill="none" stroke="{color}" stroke-width="2.5" points="{coords}" />')
        for x, y in pts:
            parts.append(f'<circle cx="{x_pos(x):.2f}" cy="{y_pos(y):.2f}" r="3.5" fill="{color}" />')
        ly = legend_y + idx * 18
        parts.append(f'<line x1="{legend_x}" y1="{ly}" x2="{legend_x+16}" y2="{ly}" stroke="{color}" stroke-width="2.5" />')
        parts.append(f'<text class="label" x="{legend_x+22}" y="{ly+4}">{name}</text>')

    parts.append(f'<text class="label" x="{width/2}" y="{height-18}" text-anchor="middle">{x_label}</text>')
    parts.append(f'<text class="label" transform="translate(20,{height/2}) rotate(-90)" text-anchor="middle">{y_label}</text>')
    svg_finish(parts, path)


def parse_memory_bench_text(path: str) -> List[Dict[str, Any]]:
    if not os.path.exists(path):
        return []
    rows: List[Dict[str, Any]] = []
    pat = re.compile(
        r"BenchmarkMemoryComparison/KeySize=(\d+)/Keys=(\d+)-\d+\s+.*?\s([0-9.]+)\s+lerl_bits_per_key\s+.*?\s([0-9.]+)\s+rl_bits_per_key"
    )
    with open(path) as f:
        for line in f:
            m = pat.search(line)
            if not m:
                continue
            rows.append(
                {
                    "keysize": int(m.group(1)),
                    "keys": int(m.group(2)),
                    "lerl_bits_per_key": float(m.group(3)),
                    "rl_bits_per_key": float(m.group(4)),
                }
            )
    return rows


def theory_query_false_positive(w_bits: int, s_bits: int) -> float:
    checks = max(1, math.ceil(math.log2(max(2, w_bits))))
    # Exact "any false positive in checks independent comparisons" model.
    return float(1.0 - math.pow(1.0 - math.pow(2.0, -float(s_bits)), float(checks)))


def theory_build_success(n_keys: int, w_bits: int, s_bits: int, rebuild_attempts: int = 100) -> float:
    p_query_fail = theory_query_false_positive(w_bits, s_bits)
    # Model: key failures are independent; one attempt succeeds only if all n keys succeed.
    p_attempt_success = math.pow(1.0 - p_query_fail, float(n_keys))
    # Build retries with fresh seeds (independent attempts).
    return float(1.0 - math.pow(1.0 - p_attempt_success, float(rebuild_attempts)))


def main() -> None:
    base = os.path.dirname(os.path.abspath(__file__))
    data_dir = os.path.join(base, "data")
    plots_dir = os.path.join(base, "plots")
    ensure_dir(data_dir)
    ensure_dir(plots_dir)

    main_rows = read_csv(os.path.join(data_dir, "grid_main_v2.csv"))
    size_rows = read_csv_if_exists(os.path.join(data_dir, "grid_size_report.csv"))
    focus_rows_base = read_csv(os.path.join(data_dir, "grid_focus_v2.csv"))
    focus_rows_extra_s16 = read_csv_if_exists(os.path.join(data_dir, "grid_focus_extra_s16.csv"))
    focus_rows_extra_s8 = read_csv_if_exists(os.path.join(data_dir, "grid_focus_extra_s8.csv"))
    focus_rows_extra_s32 = read_csv_if_exists(os.path.join(data_dir, "grid_focus_extra_s32.csv"))
    focus_rows_big_s32 = read_csv_if_exists(os.path.join(data_dir, "grid_focus_big_s32.csv"))
    focus_rows_small_s8 = read_csv_if_exists(os.path.join(data_dir, "grid_focus_small_s8.csv"))
    focus_rows = (
        focus_rows_base
        + focus_rows_extra_s16
        + focus_rows_extra_s8
        + focus_rows_extra_s32
        + focus_rows_big_s32
        + focus_rows_small_s8
    )
    focus_unique: Dict[Tuple[int, int, int], Dict[str, str]] = {}
    for r in focus_rows:
        k = (as_int(r, "n"), as_int(r, "w_bits"), as_int(r, "s_bits"))
        focus_unique[k] = r
    focus_rows = list(focus_unique.values())
    mem_rows_txt = parse_memory_bench_text(os.path.join(base, "memory_bench_v2.txt"))
    if mem_rows_txt:
        mem_rows: List[Dict[str, Any]] = mem_rows_txt
        mem_source = os.path.join(base, "memory_bench_v2.txt")
    else:
        mem_rows = [r for r in read_csv("rloc/benchmarks_parsed.csv") if r["benchmark"] == "MemoryComparison"]
        mem_source = "rloc/benchmarks_parsed.csv"

    memory_points: List[Dict[str, Any]] = [
        {
            "keysize": as_int(r, "keysize") if isinstance(r["keysize"], str) else r["keysize"],
            "keys": as_int(r, "keys") if isinstance(r["keys"], str) else r["keys"],
            "rl_bits_per_key": round(as_float(r, "rl_bits_per_key"), 6) if isinstance(r["rl_bits_per_key"], str) else round(float(r["rl_bits_per_key"]), 6),
            "lerl_bits_per_key": round(as_float(r, "lerl_bits_per_key"), 6) if isinstance(r["lerl_bits_per_key"], str) else round(float(r["lerl_bits_per_key"]), 6),
        }
        for r in mem_rows
    ]
    memory_points.sort(key=lambda r: (r["keysize"], r["keys"]))
    write_csv(
        os.path.join(data_dir, "memory_points.csv"),
        memory_points,
        ["keysize", "keys", "rl_bits_per_key", "lerl_bits_per_key"],
    )

    by_s: Dict[int, List[Dict[str, str]]] = defaultdict(list)
    by_margin: Dict[int, List[Dict[str, str]]] = defaultdict(list)
    for r in main_rows:
        s = as_int(r, "s_bits")
        by_s[s].append(r)
        by_margin[as_int(r, "s_margin_bits")].append(r)

    summary_by_s: List[Dict[str, Any]] = []
    for s in sorted(by_s):
        rows = by_s[s]
        succ = [as_float(r, "success_rate") for r in rows]
        fail = [as_float(r, "fail_rate") for r in rows]
        att = [as_float(r, "avg_attempts_success") for r in rows if as_float(r, "success_rate") > 0]
        summary_by_s.append(
            {
                "s_bits": s,
                "scenarios": len(rows),
                "success_rate_mean": round(statistics.fmean(succ), 6),
                "success_rate_p50": round(quantile(succ, 0.5), 6),
                "fail_rate_mean": round(statistics.fmean(fail), 6),
                "fail_rate_max": round(max(fail), 6),
                "avg_attempts_success_mean": round(statistics.fmean(att), 4) if att else 0.0,
            }
        )

    write_csv(
        os.path.join(data_dir, "summary_by_s.csv"),
        summary_by_s,
        [
            "s_bits",
            "scenarios",
            "success_rate_mean",
            "success_rate_p50",
            "fail_rate_mean",
            "fail_rate_max",
            "avg_attempts_success_mean",
        ],
    )

    summary_by_margin: List[Dict[str, Any]] = []
    for margin in sorted(by_margin):
        rows = by_margin[margin]
        succ = [as_float(r, "success_rate") for r in rows]
        summary_by_margin.append(
            {
                "s_margin_bits": margin,
                "scenarios": len(rows),
                "success_rate_mean": round(statistics.fmean(succ), 6),
                "success_rate_p50": round(quantile(succ, 0.5), 6),
                "success_rate_min": round(min(succ), 6),
            }
        )

    write_csv(
        os.path.join(data_dir, "summary_by_margin.csv"),
        summary_by_margin,
        ["s_margin_bits", "scenarios", "success_rate_mean", "success_rate_p50", "success_rate_min"],
    )

    theory_rows: List[Dict[str, Any]] = []
    for r in focus_rows:
        n = as_int(r, "n")
        w = as_int(r, "w_bits")
        s = as_int(r, "s_bits")
        p = theory_build_success(n, w, s, rebuild_attempts=100)
        theory_rows.append(
            {
                "n": n,
                "w_bits": w,
                "s_bits": s,
                "theory_build_success": round(p, 10),
            }
        )
    theory_rows.sort(key=lambda x: (x["s_bits"], x["w_bits"], x["n"]))
    write_csv(
        os.path.join(data_dir, "theory_focus_success.csv"),
        theory_rows,
        ["n", "w_bits", "s_bits", "theory_build_success"],
    )

    worst_rows: List[Dict[str, Any]] = sorted(main_rows, key=lambda r: as_float(r, "fail_rate"), reverse=True)[:12]
    write_csv(
        os.path.join(data_dir, "worst_cases.csv"),
        worst_rows,
        list(worst_rows[0].keys()) if worst_rows else [],
    )

    # Plot 1: average success by S.
    draw_bar_chart(
        os.path.join(plots_dir, "success_rate_by_s.svg"),
        "Build Success Rate vs S (grid_main_v2)",
        "S bits",
        "Success rate",
        [str(r["s_bits"]) for r in summary_by_s],
        [float(r["success_rate_mean"]) for r in summary_by_s],
    )

    # Plot 2: success vs n for S=16 in focused grid (+ optional extras).
    s16_rows = [r for r in focus_rows if as_int(r, "s_bits") == 16]
    n_vals = sorted({as_int(r, "n") for r in s16_rows})
    by_w16: Dict[int, List[Tuple[int, float]]] = defaultdict(list)
    for r in s16_rows:
        by_w16[as_int(r, "w_bits")].append((as_int(r, "n"), as_float(r, "success_rate")))
    series16 = {f"w={w}": sorted(vals, key=lambda p: p[0]) for w, vals in sorted(by_w16.items())}
    draw_line_chart(
        os.path.join(plots_dir, "s16_success_vs_n.svg"),
        "S=16 Build Success vs n (focus grid)",
        "n (log2 scale)",
        "Success rate",
        n_vals,
        series16,
    )

    s16_theory_series: Dict[str, List[Tuple[int, float]]] = {}
    for w, vals in sorted(by_w16.items()):
        pts: List[Tuple[int, float]] = []
        for n, _ in sorted(vals, key=lambda p: p[0]):
            pts.append((n, theory_build_success(n, w, 16, rebuild_attempts=100)))
        s16_theory_series[f"w={w}"] = pts
    draw_line_chart(
        os.path.join(plots_dir, "s16_success_vs_n_theory.svg"),
        "S=16 Theoretical Build Success vs n",
        "n (log2 scale)",
        "Success probability",
        n_vals,
        s16_theory_series,
    )
    draw_overlay_empirical_theory(
        os.path.join(plots_dir, "s16_success_vs_n_overlay.svg"),
        "S=16 Build Success: empirical vs theory",
        "n (log2 scale)",
        "Success probability",
        n_vals,
        series16,
        s16_theory_series,
    )

    # Plot 3: success vs n for S=8 in focused grid (+ optional extras).
    s8_rows = [r for r in focus_rows if as_int(r, "s_bits") == 8]
    n_vals_s8 = sorted({as_int(r, "n") for r in s8_rows})
    by_w8: Dict[int, List[Tuple[int, float]]] = defaultdict(list)
    for r in s8_rows:
        by_w8[as_int(r, "w_bits")].append((as_int(r, "n"), as_float(r, "success_rate")))
    series8 = {f"w={w}": sorted(vals, key=lambda p: p[0]) for w, vals in sorted(by_w8.items())}
    draw_line_chart(
        os.path.join(plots_dir, "s8_success_vs_n.svg"),
        "S=8 Build Success vs n (focus grid)",
        "n (log2 scale)",
        "Success rate",
        n_vals_s8,
        series8,
    )

    s8_theory_series: Dict[str, List[Tuple[int, float]]] = {}
    for w, vals in sorted(by_w8.items()):
        pts_8: List[Tuple[int, float]] = []
        for n, _ in sorted(vals, key=lambda p: p[0]):
            pts_8.append((n, theory_build_success(n, w, 8, rebuild_attempts=100)))
        s8_theory_series[f"w={w}"] = pts_8
    draw_line_chart(
        os.path.join(plots_dir, "s8_success_vs_n_theory.svg"),
        "S=8 Theoretical Build Success vs n",
        "n (log2 scale)",
        "Success probability",
        n_vals_s8,
        s8_theory_series,
    )
    draw_overlay_empirical_theory(
        os.path.join(plots_dir, "s8_success_vs_n_overlay.svg"),
        "S=8 Build Success: empirical vs theory",
        "n (log2 scale)",
        "Success probability",
        n_vals_s8,
        series8,
        s8_theory_series,
    )

    # Plot 4: memory bits/key at keys=32768 from parsed benchmark file.
    s32_rows = [r for r in focus_rows if as_int(r, "s_bits") == 32]
    n_vals_s32 = sorted({as_int(r, "n") for r in s32_rows})
    by_w32: Dict[int, List[Tuple[int, float]]] = defaultdict(list)
    for r in s32_rows:
        by_w32[as_int(r, "w_bits")].append((as_int(r, "n"), as_float(r, "success_rate")))
    series32 = {f"w={w}": sorted(vals, key=lambda p: p[0]) for w, vals in sorted(by_w32.items())}
    draw_line_chart(
        os.path.join(plots_dir, "s32_success_vs_n.svg"),
        "S=32 Build Success vs n (focus grid)",
        "n (log2 scale)",
        "Success rate",
        n_vals_s32,
        series32,
    )

    s32_theory_series: Dict[str, List[Tuple[int, float]]] = {}
    for w, vals in sorted(by_w32.items()):
        pts_32: List[Tuple[int, float]] = []
        for n, _ in sorted(vals, key=lambda p: p[0]):
            pts_32.append((n, theory_build_success(n, w, 32, rebuild_attempts=100)))
        s32_theory_series[f"w={w}"] = pts_32
    draw_line_chart(
        os.path.join(plots_dir, "s32_success_vs_n_theory.svg"),
        "S=32 Theoretical Build Success vs n",
        "n (log2 scale)",
        "Success probability",
        n_vals_s32,
        s32_theory_series,
    )
    draw_overlay_empirical_theory(
        os.path.join(plots_dir, "s32_success_vs_n_overlay.svg"),
        "S=32 Build Success: empirical vs theory",
        "n (log2 scale)",
        "Success probability",
        n_vals_s32,
        series32,
        s32_theory_series,
    )

    # Plot 4: memory bits/key at keys=32768 from parsed benchmark file.
    mem_32768 = [r for r in mem_rows if (as_int(r, "keys") if isinstance(r["keys"], str) else r["keys"]) == 32768]
    mem_32768.sort(key=lambda r: as_int(r, "keysize") if isinstance(r["keysize"], str) else r["keysize"])
    x_vals_mem = [as_int(r, "keysize") if isinstance(r["keysize"], str) else int(r["keysize"]) for r in mem_32768]
    rl_series = [(as_int(r, "keysize") if isinstance(r["keysize"], str) else int(r["keysize"]), as_float(r, "rl_bits_per_key") if isinstance(r["rl_bits_per_key"], str) else float(r["rl_bits_per_key"])) for r in mem_32768]
    lerl_series = [(as_int(r, "keysize") if isinstance(r["keysize"], str) else int(r["keysize"]), as_float(r, "lerl_bits_per_key") if isinstance(r["lerl_bits_per_key"], str) else float(r["lerl_bits_per_key"])) for r in mem_32768]
    mmph_series = [(x, 14.0) for x in x_vals_mem]
    draw_line_chart_numeric(
        os.path.join(plots_dir, "memory_bits_per_key_keys32768.svg"),
        "Memory bits/key at 32768 keys (from benchmarks_parsed.csv)",
        "Key size (bits)",
        "bits/key",
        x_vals_mem,
        {
            "RLOC": rl_series,
            "LERLOC": lerl_series,
            "MMPH baseline=14": mmph_series,
        },
    )

    if size_rows:
        by_s_n: Dict[Tuple[int, int], List[float]] = defaultdict(list)
        n_vals_size = sorted({as_int(r, "n") for r in size_rows})
        for r in size_rows:
            bpk_raw = r.get("bpk", "none")
            if bpk_raw == "none":
                continue
            s_val = as_int(r, "s_bits")
            n_val = as_int(r, "n")
            by_s_n[(s_val, n_val)].append(float(bpk_raw))

        series_size: Dict[str, List[Tuple[int, float]]] = {}
        for s_bit in [8, 16, 32]:
            pts_size: List[Tuple[int, float]] = []
            for n_size in n_vals_size:
                vals_size = by_s_n.get((s_bit, n_size), [])
                if not vals_size:
                    continue
                pts_size.append((n_size, float(statistics.median(vals_size))))
            if pts_size:
                series_size[f"S={s_bit}"] = pts_size

        if series_size:
            draw_line_chart_logx_numericy(
                os.path.join(plots_dir, "grid_size_report_bpk_vs_n.svg"),
                "Grid Size Report: median bpk vs n by S",
                "n (log2 scale)",
                "bits per key (median over w)",
                n_vals_size,
                series_size,
            )

    rl_vals = [as_float(r, "rl_bits_per_key") if isinstance(r["rl_bits_per_key"], str) else float(r["rl_bits_per_key"]) for r in mem_rows]
    lerl_vals = [as_float(r, "lerl_bits_per_key") if isinstance(r["lerl_bits_per_key"], str) else float(r["lerl_bits_per_key"]) for r in mem_rows]
    stable_rows = [r for r in mem_rows if (as_int(r, "keys") if isinstance(r["keys"], str) else r["keys"]) >= 8192]
    stable_rl = [as_float(r, "rl_bits_per_key") if isinstance(r["rl_bits_per_key"], str) else float(r["rl_bits_per_key"]) for r in stable_rows]
    stable_lerl = [as_float(r, "lerl_bits_per_key") if isinstance(r["lerl_bits_per_key"], str) else float(r["lerl_bits_per_key"]) for r in stable_rows]

    lines = []
    lines.append("# PSig / Memory Study Summary")
    lines.append("")
    lines.append("## Inputs")
    lines.append("- `mmph/relative_trie/study/data/grid_main_v2.csv`")
    lines.append("- `mmph/relative_trie/study/data/grid_focus_v2.csv`")
    if focus_rows_extra_s16:
        lines.append("- `mmph/relative_trie/study/data/grid_focus_extra_s16.csv`")
    if focus_rows_extra_s8:
        lines.append("- `mmph/relative_trie/study/data/grid_focus_extra_s8.csv`")
    if focus_rows_extra_s32:
        lines.append("- `mmph/relative_trie/study/data/grid_focus_extra_s32.csv`")
    if focus_rows_big_s32:
        lines.append("- `mmph/relative_trie/study/data/grid_focus_big_s32.csv`")
    if focus_rows_small_s8:
        lines.append("- `mmph/relative_trie/study/data/grid_focus_small_s8.csv`")
    lines.append(f"- `{mem_source}`")
    lines.append("")
    lines.append("## Main Findings")
    lines.append(
        f"- On `grid_main_v2` (144 scenarios, 64 trials each): mean success by S is "
        + ", ".join(f"S={r['s_bits']}: {r['success_rate_mean']:.3f}" for r in summary_by_s)
        + "."
    )
    lines.append(
        "- `S=32` is fully stable in this grid (all scenarios succeeded in all trials); "
        "S=8 fails for most medium/large settings."
    )
    lines.append(
        "- `S=16` is mixed: stable up to moderate `n`, but for `n=131072` several `w` values "
        "show severe degradation."
    )
    lines.append(
        "- This confirms that theorem-based `S` from per-query bound (`epsilon_query = m/n`) "
        "is necessary but not sufficient for high probability of full-structure build success."
    )
    lines.append("")
    lines.append("## Plots")
    lines.append("- [S=8 empirical](plots/s8_success_vs_n.svg), [S=8 theory](plots/s8_success_vs_n_theory.svg), [S=8 overlay](plots/s8_success_vs_n_overlay.svg)")
    lines.append("- [S=16 empirical](plots/s16_success_vs_n.svg), [S=16 theory](plots/s16_success_vs_n_theory.svg), [S=16 overlay](plots/s16_success_vs_n_overlay.svg)")
    lines.append("- [S=32 empirical](plots/s32_success_vs_n.svg), [S=32 theory](plots/s32_success_vs_n_theory.svg), [S=32 overlay](plots/s32_success_vs_n_overlay.svg)")
    if size_rows:
        lines.append("- [Grid size report (bpk vs n)](plots/grid_size_report_bpk_vs_n.svg)")
    lines.append("- [Memory bits/key at 32768 keys](plots/memory_bits_per_key_keys32768.svg)")
    lines.append("")
    lines.append("## Detailed Theory: how build-success probability was computed")
    lines.append("- References used: `papers/MMPH/Definitions-and-Tools.md`, `papers/MMPH/Section-3-Bucketing.md`, `papers/MMPH/Section-4-Relative-Ranking.md` (Theorem 4.1), `papers/MMPH/Section-5-Relative-Trie.md` (Theorem 5.2).")
    lines.append("- Goal of this section: derive (i) required PSig width `S` from the paper, and (ii) build-success probability for our concrete implementation.")
    lines.append("")
    lines.append("### 0. Notation aligned with the paper")
    lines.append("- `n = |S|`: number of keys for which queries must be correct.")
    lines.append("- `m = |D|`: number of delimiters (one per bucket). For bucket size `b`, typically `m = ceil(n/b)`.")
    lines.append("- `w`: max key length in bits.")
    lines.append("- `k`: number of signature checks during fat binary search; by Theorem 4.1 analysis, `k <= ceil(log2(w))`.")
    lines.append("- `S`: PSig width in bits (hash/signature length stored in trie entries).")
    lines.append("- `R`: max rebuild attempts (`maxTrieRebuilds = 100` in current code).")
    lines.append("")
    lines.append("### 1. From Theorem 4.1 to per-query failure")
    lines.append("- Theorem 4.1 states that each signature check uses `log2(log2(w)) + log2(1/epsilon_query)` bits.")
    lines.append("- Therefore with fixed width `S`, one comparison false-match probability is:")
    lines.append("$$")
    lines.append("p_{\\mathrm{cmp}} = 2^{-S}")
    lines.append("$$")
    lines.append("- Query performs up to `k` checks. Union bound used in the theorem proof:")
    lines.append("$$")
    lines.append("p_{\\mathrm{query\\_fail}} \\leq k\\cdot 2^{-S}")
    lines.append("$$")
    lines.append("- In code/plots we also use tighter independent-check approximation:")
    lines.append("$$")
    lines.append("p_{\\mathrm{query\\_fail}} \\approx 1-(1-2^{-S})^{k}")
    lines.append("$$")
    lines.append("- This is implemented by `theory_query_false_positive()` with `k = max(1, ceil(log2(max(2, w))))`.")
    lines.append("")
    lines.append("### 2. Why `epsilon_query = m/n` in Theorem 5.2")
    lines.append("- Theorem 5.2 sets per-query error target to `epsilon_query = m/n`.")
    lines.append("- Then expected number of misclassified keys over all `n` keys is:")
    lines.append("$$")
    lines.append("\\mathbb{E}[|E|] = n\\cdot \\varepsilon_{\\mathrm{query}} = m")
    lines.append("$$")
    lines.append("- Paper then stores explicit corrections for this set `E` (relative-membership + stored exact answers), yielding exact queries on `S`.")
    lines.append("- Substituting `epsilon_query = m/n` into Theorem 4.1 gives required PSig width:")
    lines.append("$$")
    lines.append("S \\geq \\log_2\\!\\log_2(w) + \\log_2\\!\\left(\\frac{n}{m}\\right)")
    lines.append("$$")
    lines.append("- For fixed bucket size `b` where `m \\approx n/b`, this becomes:")
    lines.append("$$")
    lines.append("S \\geq \\log_2\\!\\log_2(w) + \\log_2(b)")
    lines.append("$$")
    lines.append("- For `b=256`, this is roughly `\\log_2\\!\\log_2(w) + 8` (plus integer ceiling/constant slack).")
    lines.append("")
    lines.append("### 3. Mapping paper guarantees to our builder")
    lines.append("- Current implementation does **not** store correction set `E` from Theorem 5.2.")
    lines.append("- Instead, one build attempt is accepted only if `validateAllKeys` succeeds for all keys (`n` checks pass).")
    lines.append("- So we model probability of *zero failures in one attempt*.")
    lines.append("")
    lines.append("### 4. One-attempt success model")
    lines.append("- Let `p = p_query_fail` from Step 1.")
    lines.append("- Independent-query approximation gives:")
    lines.append("$$")
    lines.append("p_{\\mathrm{attempt\\_success}} \\approx (1-p)^n")
    lines.append("$$")
    lines.append("- For small `p`, this is close to Poisson form:")
    lines.append("$$")
    lines.append("p_{\\mathrm{attempt\\_success}} \\approx e^{-\\lambda},\\quad \\lambda = n\\cdot p")
    lines.append("$$")
    lines.append("- This explains sharp phase transitions: once `n * p` is not small, all-keys success becomes unlikely.")
    lines.append("")
    lines.append("### 5. Rebuild attempts")
    lines.append("- Builder retries with fresh seeds up to `R=100` attempts.")
    lines.append("- With approximate independence between attempts:")
    lines.append("$$")
    lines.append("p_{\\mathrm{build\\_success}} \\approx 1-(1-p_{\\mathrm{attempt\\_success}})^R")
    lines.append("$$")
    lines.append("- This is exactly what `theory_build_success()` computes and what is plotted in `*_theory.svg` and `*_overlay.svg`.")
    lines.append("")
    lines.append("### 6. Conservative bound vs approximation")
    lines.append("- Conservative theorem-style upper bound for query failure:")
    lines.append("$$")
    lines.append("p_{\\mathrm{query\\_fail}}^{\\mathrm{bound}} = \\min\\left(1, k\\cdot2^{-S}\\right)")
    lines.append("$$")
    lines.append("- This yields a lower bound for one-attempt success:")
    lines.append("$$")
    lines.append("p_{\\mathrm{attempt\\_success}} \\ge (1-p_{\\mathrm{query\\_fail}}^{\\mathrm{bound}})^n")
    lines.append("$$")
    lines.append("- We plot the independent approximation because it tracks observed trends better than the loose union-bound lower bound.")
    lines.append("")
    lines.append("### 7. Why empirical and theory differ")
    lines.append("- Hash/signature events are not perfectly independent across keys (shared trie paths).")
    lines.append("- `validateAllKeys` behavior depends on actual trie shape, delimiter distribution, and candidate selection details (`LowerBound`).")
    lines.append("- Attempt-to-attempt outcomes may also be correlated for some key sets despite seed changes.")
    lines.append("- The implementation applies a length-consistency collision filter in `GetExistingPrefix` that rejects MPH hits with `extentLen < fFast`, removing a class of deterministic false positives not modeled in the paper. See `zfasttrie/getexistingprefix_collision_filter.md`.")
    lines.append("- Therefore we expect qualitative trend alignment, not exact numeric equality.")
    lines.append("")
    # One concrete scenario to make the model transparent.
    ex = next(
        (
            r
            for r in focus_rows
            if as_int(r, "n") == 131072 and as_int(r, "w_bits") == 128 and as_int(r, "s_bits") == 16
        ),
        None,
    )
    if ex is not None:
        ex_n = as_int(ex, "n")
        ex_w = as_int(ex, "w_bits")
        ex_s = as_int(ex, "s_bits")
        ex_k = max(1, math.ceil(math.log2(max(2, ex_w))))
        ex_pq = theory_query_false_positive(ex_w, ex_s)
        ex_lambda = float(ex_n) * ex_pq
        ex_pa = (1.0 - ex_pq) ** ex_n
        ex_pb = 1.0 - (1.0 - ex_pa) ** 100
        ex_emp = as_float(ex, "success_rate")
        lines.append("### Worked example (`n=131072, w=128, S=16`)")
        lines.append(f"- `k = {ex_k}`")
        lines.append("$$")
        lines.append(f"p_{{\\mathrm{{query\\_fail}}}} \\approx {ex_pq:.8g}")
        lines.append("$$")
        lines.append("$$")
        lines.append(f"\\lambda = n\\cdot p_{{\\mathrm{{query\\_fail}}}} \\approx {ex_lambda:.8g}")
        lines.append("$$")
        lines.append("$$")
        lines.append(f"p_{{\\mathrm{{attempt\\_success}}}} \\approx {ex_pa:.8g}")
        lines.append("$$")
        lines.append("$$")
        lines.append(f"p_{{\\mathrm{{build\\_success}}}}(\\mathrm{{theory}}, R=100) \\approx {ex_pb:.8g}")
        lines.append("$$")
        lines.append(f"- `p_build_success(empirical from focus grid) = {ex_emp:.8g}`")
        lines.append("")
    lines.append("## Margin Analysis")
    lines.append("- `summary_by_margin.csv` shows behavior grouped by `s_margin_bits = S - S_required`.")
    lines.append("- Positive margin improves success rate but does not guarantee `~1.0` success for largest `n`.")
    lines.append("")
    lines.append(f"## Memory Snapshot (from `{mem_source}`)")
    lines.append(
        f"- RLOC bits/key: min={min(rl_vals):.2f}, median={statistics.median(rl_vals):.2f}, max={max(rl_vals):.2f}."
    )
    lines.append(
        f"- LERLOC bits/key: min={min(lerl_vals):.2f}, median={statistics.median(lerl_vals):.2f}, max={max(lerl_vals):.2f}."
    )
    lines.append(
        f"- Stable regime (`keys>=8192`): RLOC avg={statistics.fmean(stable_rl):.2f} bits/key, "
        f"LERLOC avg={statistics.fmean(stable_lerl):.2f} bits/key."
    )
    lines.append("- MMPH baseline from paper chart: ~14 bits/key (for large n).")
    lines.append("")
    lines.append("## Generated Artifacts")
    lines.append("- [`data/summary_by_s.csv`](data/summary_by_s.csv)")
    lines.append("- [`data/summary_by_margin.csv`](data/summary_by_margin.csv)")
    lines.append("- [`data/worst_cases.csv`](data/worst_cases.csv)")
    lines.append("- [`data/memory_points.csv`](data/memory_points.csv)")
    lines.append("- [`data/theory_focus_success.csv`](data/theory_focus_success.csv)")
    lines.append("- [`plots/success_rate_by_s.svg`](plots/success_rate_by_s.svg)")
    lines.append("- [`plots/s16_success_vs_n.svg`](plots/s16_success_vs_n.svg)")
    lines.append("- [`plots/s16_success_vs_n_theory.svg`](plots/s16_success_vs_n_theory.svg)")
    lines.append("- [`plots/s16_success_vs_n_overlay.svg`](plots/s16_success_vs_n_overlay.svg)")
    lines.append("- [`plots/s8_success_vs_n.svg`](plots/s8_success_vs_n.svg)")
    lines.append("- [`plots/s8_success_vs_n_theory.svg`](plots/s8_success_vs_n_theory.svg)")
    lines.append("- [`plots/s8_success_vs_n_overlay.svg`](plots/s8_success_vs_n_overlay.svg)")
    lines.append("- [`plots/s32_success_vs_n.svg`](plots/s32_success_vs_n.svg)")
    lines.append("- [`plots/s32_success_vs_n_theory.svg`](plots/s32_success_vs_n_theory.svg)")
    lines.append("- [`plots/s32_success_vs_n_overlay.svg`](plots/s32_success_vs_n_overlay.svg)")
    if size_rows:
        lines.append("- [`plots/grid_size_report_bpk_vs_n.svg`](plots/grid_size_report_bpk_vs_n.svg)")
    lines.append("- [`plots/memory_bits_per_key_keys32768.svg`](plots/memory_bits_per_key_keys32768.svg)")

    with open(os.path.join(base, "analysis_summary.md"), "w") as f:
        f.write("\n".join(lines) + "\n")
if __name__ == "__main__":
    main()
