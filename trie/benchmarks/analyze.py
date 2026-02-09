#!/usr/bin/env python3
import argparse
import csv
import os
import sys
from collections import defaultdict
from typing import Dict, List, Any

# Setup path to import shared library
current_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(os.path.dirname(current_dir))
scripts_dir = os.path.join(project_root, "scripts")
sys.path.append(scripts_dir)

from bench_lib import runner, parser, plotter

# --- Configuration ---

RAW_DIR = os.path.join(current_dir, "raw")
PARSED_DIR = os.path.join(current_dir, "parsed")
PLOTS_DIR = os.path.join(current_dir, "plots")

MODULES = [
    {
        "name": "trie_builder",
        "dir": os.path.join(project_root, "trie"),
        "out": os.path.join(RAW_DIR, "builder.txt"),
    },
]

# Mapping for labels
BENCH_LABELS = {
    "BenchmarkZFTBuild": "ZFT",
    "BenchmarkHZFTBuild_Heavy": "HZFT Old (Heavy)",
    "BenchmarkHZFTBuild_Streaming": "HZFT New (Streaming)",
    "BenchmarkAZFTBuild_Heavy": "AZFT Old (Heavy)",
    "BenchmarkAZFTBuild_Streaming": "AZFT New (Streaming)",
    "BenchmarkRangeLocatorBuild": "RangeLocator",
    "BenchmarkLocalExactRangeLocatorBuild": "LocalExactRangeLocator",
}

def main() -> int:
    arg_parser = argparse.ArgumentParser(description="Parse Go benchmark outputs and generate SVG plots.")
    arg_parser.add_argument("--run", action="store_true", help="Run benchmarks before parsing.")
    arg_parser.add_argument("--count", type=int, default=5, help="Benchmark repeat count (default: 5).")
    arg_parser.add_argument("--bench", default="Benchmark.*Build", help="Benchmark regex (default: Benchmark.*Build)")
    arg_parser.add_argument("--jobs", "-j", type=int, default=None, help="Number of parallel jobs (default: all cores).")
    arg_parser.add_argument("--no-benchmem", action="store_true", help="Disable -benchmem.")
    args = arg_parser.parse_args()

    plotter.ensure_dir(RAW_DIR)
    plotter.ensure_dir(PARSED_DIR)
    plotter.ensure_dir(PLOTS_DIR)

    if args.run:
        runner.run_benchmarks(MODULES, args.count, args.bench, not args.no_benchmem, args.jobs)

    # 2. Parse Results
    all_rows = []
    for mod in MODULES:
        rows = parser.parse_file(mod["out"])
        all_rows.extend(rows)
        
    if not all_rows:
        print("No data found. Did you run with --run?")
        return 1

    # Write long CSV
    all_fields = set()
    for r in all_rows:
        all_fields.update(r.keys())
    
    header = [
        "benchmark",
        "keysize",
        "keys",
        "ns_per_op",
        "bytes_per_op",
        "allocs_per_op",
        "full_name",
    ]
    extra = sorted(k for k in all_fields if k not in header)
    header = header + extra

    long_path = os.path.join(PARSED_DIR, "bench_long.csv")
    with open(long_path, "w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=header)
        writer.writeheader()
        writer.writerows(all_rows)

    # 3. Aggregate
    agg_rows = parser.aggregate(all_rows, ["benchmark", "keysize", "keys"])
    
    # Define metrics to plot
    # (MetricKey, TitleSuffix, YLabel, LogY)
    metrics_to_plot = [
        ("ns_per_op", "Time (ns/op)", "ns", True),
        ("allocs_per_op", "Allocs per Op", "allocs", True),
        ("bytes_per_op", "Bytes per Op", "bytes", True),
    ]

    # Group by KeySize
    by_keysize = defaultdict(list)
    for r in agg_rows:
        ks = r.get("keysize")
        if ks is not None:
            by_keysize[int(ks)].append(r)

    for ks, rows in by_keysize.items():
        for metric, title_suffix, ylabel, log_y in metrics_to_plot:
            series = defaultdict(list)
            for r in rows:
                bench = r["benchmark"]
                label = BENCH_LABELS.get(bench, bench)
                val = r.get(metric)
                keys = r.get("keys")
                
                if val is not None and keys is not None:
                    series[label].append((keys, val))

            if series:
                fname = f"keysize_{ks}_{metric}.svg"
                plotter.draw_line_chart(
                    os.path.join(PLOTS_DIR, fname),
                    f"Build (KeySize={ks}): {title_suffix}",
                    "Number of Keys",
                    ylabel,
                    series,
                    log_x=True,
                    log_y=log_y
                )

    print(f"\nWrote CSVs to: {PARSED_DIR}")
    print(f"SVG plots in: {PLOTS_DIR}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
