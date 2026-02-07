#!/usr/bin/env python3
import argparse
import concurrent.futures
import csv
import math
import os
import statistics
import subprocess
import sys
from collections import defaultdict

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
RAW_DIR = os.path.join(BASE_DIR, "raw")
PARSED_DIR = os.path.join(BASE_DIR, "parsed")
PLOTS_DIR = os.path.join(BASE_DIR, "plots")

MODULES = [
    {
        "name": "bucket-mmph",
        "dir": "/Users/andrei.ogurtsov/Thesis/mmph/bucket-mmph",
        "out": os.path.join(RAW_DIR, "bucket-mmph.txt"),
    },
    {
        "name": "rbtz-mmph",
        "dir": "/Users/andrei.ogurtsov/Thesis/mmph/rbtz-mmph",
        "out": os.path.join(RAW_DIR, "rbtz-mmph.txt"),
    },
    {
        "name": "bucket_with_approx_trie",
        "dir": "/Users/andrei.ogurtsov/Thesis/mmph/bucket_with_approx_trie",
        "out": os.path.join(RAW_DIR, "bucket_with_approx_trie.txt"),
    },
]

BUILD_BENCH = {
    "bucket-mmph": {"BenchmarkMonotoneHashBuild"},
    "rbtz-mmph": {"BenchmarkBuild"},
    "bucket_with_approx_trie": {"BenchmarkMonotoneHashWithTrieBuild"},
}

LOOKUP_BENCH = {
    "bucket-mmph": {"BenchmarkMonotoneHashLookup"},
    "rbtz-mmph": {"BenchmarkLookup"},
    "bucket_with_approx_trie": {"BenchmarkMonotoneHashWithTrieLookup"},
}

CORE_METRICS = [
    "build_time_ns",
    "lookup_time_ns",
    "bits_per_key_in_mem",
    "bytes_in_mem",
    "allocs_per_op",
]


def ensure_dir(path):
    os.makedirs(path, exist_ok=True)


def parse_keycount(bench_full):
    # Example: BenchmarkBuild/Keys=1024-16
    if "Keys=" not in bench_full:
        return None
    try:
        part = bench_full.split("Keys=")[1]
        num = ""
        for ch in part:
            if ch.isdigit():
                num += ch
            else:
                break
        return int(num) if num else None
    except Exception:
        return None


def parse_benchmark_line(line):
    line = line.strip()
    if not line.startswith("Benchmark"):
        return None
    parts = line.split()
    if len(parts) < 4:
        return None

    bench_full = parts[0]
    try:
        samples = int(parts[1])
    except ValueError:
        return None

    keycount = parse_keycount(bench_full)
    if keycount is None:
        return None

    row = {
        "benchmark_full": bench_full,
        "benchmark": bench_full.split("/")[0],
        "keycount": keycount,
        "samples": samples,
        "ns_per_op": None,
        "bytes_per_op": None,
        "allocs_per_op": None,
        "raw_line": line,
    }

    # Parse standard metrics and custom ReportMetric values.
    # Go benchmarks output custom metrics as "value unit"
    for i, tok in enumerate(parts[2:], start=2):
        if tok == "ns/op":
            row["ns_per_op"] = float(parts[i - 1])
        elif tok == "B/op":
            row["bytes_per_op"] = float(parts[i - 1])
        elif tok == "allocs/op":
            row["allocs_per_op"] = float(parts[i - 1])
        elif "/" in tok or "_" in tok:
            # Likely a custom metric unit like bits/key_in_mem or bytes_in_mem
            try:
                row[tok] = float(parts[i - 1])
            except (ValueError, IndexError):
                pass
        elif "=" in tok:
            name, val = tok.split("=", 1)
            try:
                row[name] = float(val)
            except ValueError:
                pass

    return row


def parse_raw_files():
    rows = []
    run_counts = defaultdict(int)

    if not os.path.isdir(RAW_DIR):
        return rows

    for fname in sorted(os.listdir(RAW_DIR)):
        if not fname.endswith(".txt"):
            continue
        module = fname[:-4]
        path = os.path.join(RAW_DIR, fname)
        with open(path, "r") as f:
            for line in f:
                row = parse_benchmark_line(line)
                if row is None:
                    continue
                key = (module, row["benchmark"], row["keycount"])
                run_counts[key] += 1
                row["run_index"] = run_counts[key]
                row["module"] = module
                rows.append(row)

    return rows


def write_csv(path, rows, header):
    with open(path, "w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=header)
        writer.writeheader()
        for r in rows:
            writer.writerow(r)


def median(values):
    if not values:
        return None
    return statistics.median(values)


def build_aggregates(rows):
    # Collect per-metric values
    metric_values = defaultdict(list)  # (metric, module, keycount) -> [values]
    metric_presence = defaultdict(set)  # (metric, module) -> set(keycount)

    for row in rows:
        module = row["module"]
        bench = row["benchmark"]
        keycount = row["keycount"]

        # Build metrics
        if bench in BUILD_BENCH.get(module, set()):
            if row.get("ns_per_op") is not None:
                metric_values[("build_time_ns", module, keycount)].append(row["ns_per_op"])
                metric_presence[("build_time_ns", module)].add(keycount)
            if row.get("allocs_per_op") is not None:
                metric_values[("allocs_per_op", module, keycount)].append(row["allocs_per_op"])
                metric_presence[("allocs_per_op", module)].add(keycount)

            bits_val = None
            if row.get("bits/key_in_mem") is not None:
                bits_val = row.get("bits/key_in_mem")
            elif row.get("bits_per_key") is not None:
                bits_val = row.get("bits_per_key")
            if bits_val is not None:
                metric_values[("bits_per_key_in_mem", module, keycount)].append(bits_val)
                metric_presence[("bits_per_key_in_mem", module)].add(keycount)

            if row.get("bytes_in_mem") is not None:
                metric_values[("bytes_in_mem", module, keycount)].append(row.get("bytes_in_mem"))
                metric_presence[("bytes_in_mem", module)].add(keycount)

        # Lookup metrics
        if bench in LOOKUP_BENCH.get(module, set()):
            if row.get("ns_per_op") is not None:
                metric_values[("lookup_time_ns", module, keycount)].append(row["ns_per_op"])
                metric_presence[("lookup_time_ns", module)].add(keycount)

    # Aggregate medians
    agg_rows = []
    for (metric, module, keycount), vals in sorted(metric_values.items()):
        med = median(vals)
        if med is None:
            continue
        agg_rows.append(
            {
                "metric": metric,
                "module": module,
                "keycount": keycount,
                "value": med,
            }
        )

    # Warnings for missing metrics per module (at least one point expected)
    modules = sorted({r["module"] for r in rows})
    missing = []
    for metric in CORE_METRICS:
        for module in modules:
            if not metric_presence.get((metric, module)):
                missing.append((metric, module))

    return agg_rows, missing


def svg_start(width, height):
    return [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<style>text{font-family:Menlo,Monaco,monospace;font-size:12px;fill:#222} .axis{stroke:#333;stroke-width:1} .grid{stroke:#ddd;stroke-width:1} .label{font-size:11px;fill:#444}</style>',
    ]


def svg_finish(parts, path):
    parts.append("</svg>")
    with open(path, "w") as f:
        f.write("\n".join(parts))


def draw_line_chart_logx(path, title, x_label, y_label, series, log_y=False):
    width, height = 960, 540
    left, right, top, bottom = 90, 40, 55, 75
    pw = width - left - right
    ph = height - top - bottom

    x_vals = sorted({x for pts in series.values() for x, _ in pts})
    if not x_vals:
        return

    y_values = [y for pts in series.values() for _, y in pts if y > 0]
    if not y_values:
        y_values = [1.0]
    
    y_min_val = min(y_values)
    y_max_val = max(y_values)

    if log_y:
        # Use log10 for Y axis
        y_min = math.floor(math.log10(y_min_val)) if y_min_val > 0 else 0
        y_max = math.ceil(math.log10(y_max_val * 1.1))
        if y_max == y_min:
            y_max = y_min + 1
    else:
        y_min = 0.0
        y_max = y_max_val * 1.1

    def x_pos(x):
        x_min = min(x_vals)
        x_max = max(x_vals)
        if x_max == x_min:
            return left + pw / 2
        t = (math.log2(x) - math.log2(x_min)) / (math.log2(x_max) - math.log2(x_min))
        return left + t * pw

    def y_pos(y):
        if log_y:
            if y <= 0: return top + ph
            val = math.log10(y)
            t = (val - y_min) / (y_max - y_min)
        else:
            t = (y - y_min) / (y_max - y_min)
        return top + ph - t * ph

    parts = svg_start(width, height)
    parts.append(f'<text x="{width/2}" y="26" text-anchor="middle">{title}</text>')
    parts.append(f'<line class="axis" x1="{left}" y1="{top+ph}" x2="{left+pw}" y2="{top+ph}" />')
    parts.append(f'<line class="axis" x1="{left}" y1="{top}" x2="{left}" y2="{top+ph}" />')

    # Y Grid
    if log_y:
        for p in range(int(y_min), int(y_max) + 1):
            yv = 10**p
            py = y_pos(yv)
            if top <= py <= top + ph:
                parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
                parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">10^{p}</text>')
    else:
        for i in range(6):
            yv = y_max * i / 5
            py = y_pos(yv)
            parts.append(f'<line class="grid" x1="{left}" y1="{py:.2f}" x2="{left+pw}" y2="{py:.2f}" />')
            parts.append(f'<text class="label" x="{left-8}" y="{py+4:.2f}" text-anchor="end">{yv:.2f}</text>')

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
        pts = sorted(pts, key=lambda t: t[0])
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


def build_plots(agg_rows):
    ensure_dir(PLOTS_DIR)
    by_metric = defaultdict(list)
    for r in agg_rows:
        by_metric[r["metric"]].append(r)

    plot_specs = {
        "build_time_ns": ("Build Time (ns/op)", "Key count", "ns/op", "build_time_ns.svg", True),
        "lookup_time_ns": ("Lookup Time (ns/op)", "Key count", "ns/op", "lookup_time_ns.svg", False),
        "bits_per_key_in_mem": ("Bits per Key (in-mem)", "Key count", "bits/key", "bits_per_key_in_mem.svg", False),
        "bytes_in_mem": ("Bytes (in-mem)", "Key count", "bytes", "bytes_in_mem.svg", True),
        "allocs_per_op": ("Allocs per Op (build)", "Key count", "allocs/op", "allocs_per_op.svg", True),
    }

    for metric, (title, xlab, ylab, fname, log_y) in plot_specs.items():
        rows = by_metric.get(metric, [])
        series = defaultdict(list)
        for r in rows:
            series[r["module"]].append((int(r["keycount"]), float(r["value"])))
        out_path = os.path.join(PLOTS_DIR, fname)
        draw_line_chart_logx(out_path, title, xlab, ylab, series, log_y=log_y)


def run_single_iteration(mod, iteration, bench, benchmem):
    # Run exactly one iteration per process
    args = ["go", "test", f"-bench={bench}", "-count=1"]
    if benchmem:
        args.append("-benchmem")
    args.append("./...")
    
    # Unique temp file for this iteration
    temp_out = f"{mod['out']}.{iteration}"
    
    try:
        with open(temp_out, "w") as f:
            subprocess.run(args, cwd=mod["dir"], stdout=f, stderr=subprocess.STDOUT, check=True)
        return (mod["name"], iteration, temp_out, True)
    except subprocess.CalledProcessError as e:
        print(f"Error running benchmark {mod['name']} iter {iteration}: {e}", file=sys.stderr)
        return (mod["name"], iteration, temp_out, False)


def run_benchmarks(count, bench, benchmem, jobs):
    ensure_dir(RAW_DIR)
    
    if jobs is None:
        jobs = os.cpu_count() or 4
        
    print(f"Running benchmarks with parallelism: {jobs} workers")
    print(f"Total tasks: {len(MODULES)} modules * {count} runs = {len(MODULES)*count}")

    with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as executor:
        futures = []
        for mod in MODULES:
            for i in range(count):
                futures.append(executor.submit(run_single_iteration, mod, i, bench, benchmem))
        
        # Wait for all
        results = []
        for f in concurrent.futures.as_completed(futures):
            results.append(f.result())

    # Merge results
    print("Merging output files...")
    for mod in MODULES:
        final_out = mod["out"]
        with open(final_out, "w") as outfile:
            for i in range(count):
                temp_file = f"{mod['out']}.{i}"
                if os.path.exists(temp_file):
                    with open(temp_file, "r") as infile:
                        outfile.write(infile.read())
                    os.remove(temp_file) # Clean up
    print("Done.")


def main():
    parser = argparse.ArgumentParser(description="Parse Go benchmark outputs and generate SVG plots.")
    parser.add_argument("--run", action="store_true", help="Run benchmarks before parsing.")
    parser.add_argument("--count", type=int, default=5, help="Benchmark repeat count (default: 5).")
    parser.add_argument("--bench", default=".", help="Benchmark regex (default: .)")
    parser.add_argument("--jobs", "-j", type=int, default=None, help="Number of parallel jobs (default: all cores).")
    parser.add_argument("--no-benchmem", action="store_true", help="Disable -benchmem.")
    args = parser.parse_args()

    ensure_dir(PARSED_DIR)
    ensure_dir(PLOTS_DIR)

    if args.run:
        run_benchmarks(args.count, args.bench, not args.no_benchmem, args.jobs)

    rows = parse_raw_files()
    if not rows:
        print("No benchmark rows parsed. Did you run with --run or provide raw files?", file=sys.stderr)
        return 1

    # Write long CSV
    all_fields = set()
    for r in rows:
        all_fields.update(r.keys())
    header = [
        "module",
        "benchmark",
        "benchmark_full",
        "keycount",
        "samples",
        "run_index",
        "ns_per_op",
        "bytes_per_op",
        "allocs_per_op",
        "raw_line",
    ]
    extra = sorted(k for k in all_fields if k not in header)
    header = header + extra

    long_path = os.path.join(PARSED_DIR, "bench_long.csv")
    write_csv(long_path, rows, header)

    # Aggregate
    agg_rows, missing = build_aggregates(rows)
    agg_path = os.path.join(PARSED_DIR, "bench_agg.csv")
    write_csv(agg_path, agg_rows, ["metric", "module", "keycount", "value"])

    # Plots
    build_plots(agg_rows)

    # Print a summary table for bits_per_key_in_mem
    print("\nSummary: Bits per Key (in-mem)")
    bits_data = [r for r in agg_rows if r["metric"] == "bits_per_key_in_mem"]
    if bits_data:
        # Get all modules and keycounts for the table
        modules = sorted({r["module"] for r in bits_data})
        keycounts = sorted({r["keycount"] for r in bits_data})
        
        # Header
        header_str = f"{'Key Count':>12}"
        for mod in modules:
            header_str += f" | {mod:>25}"
        print(header_str)
        print("-" * len(header_str))
        
        for kc in keycounts:
            row_str = f"{kc:12d}"
            for mod in modules:
                val = next((r["value"] for r in bits_data if r["module"] == mod and r["keycount"] == kc), None)
                if val is not None:
                    row_str += f" | {val:25.2f}"
                else:
                    row_str += f" | {'-':>25}"
            print(row_str)

    if missing:
        print("Warnings: missing metrics for some modules:", file=sys.stderr)
        for metric, module in missing:
            print(f"  missing {metric} for {module}", file=sys.stderr)

    print(f"Wrote: {long_path}")
    print(f"Wrote: {agg_path}")
    print(f"SVG plots in: {PLOTS_DIR}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
