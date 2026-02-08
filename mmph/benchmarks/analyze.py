#!/usr/bin/env python3
import argparse
import csv
import os
import sys
from collections import defaultdict

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
        "name": "bucket-mmph",
        "dir": os.path.join(project_root, "mmph/bucket-mmph"),
        "out": os.path.join(RAW_DIR, "bucket-mmph.txt"),
    },
    {
        "name": "rbtz-mmph",
        "dir": os.path.join(project_root, "mmph/rbtz-mmph"),
        "out": os.path.join(RAW_DIR, "rbtz-mmph.txt"),
    },
    {
        "name": "bucket_with_approx_trie",
        "dir": os.path.join(project_root, "mmph/bucket_with_approx_trie"),
        "out": os.path.join(RAW_DIR, "bucket_with_approx_trie.txt"),
    },
]

# Mapping for aggregation
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

def main():
    arg_parser = argparse.ArgumentParser(description="Parse Go benchmark outputs and generate SVG plots.")
    arg_parser.add_argument("--run", action="store_true", help="Run benchmarks before parsing.")
    arg_parser.add_argument("--count", type=int, default=5, help="Benchmark repeat count (default: 5).")
    arg_parser.add_argument("--bench", default=".", help="Benchmark regex (default: .)")
    arg_parser.add_argument("--jobs", "-j", type=int, default=None, help="Number of parallel jobs (default: all cores).")
    arg_parser.add_argument("--split", action="store_true", help="Split benchmarks into individual tasks for max parallelism")
    arg_parser.add_argument("--no-benchmem", action="store_true", help="Disable -benchmem.")
    args = arg_parser.parse_args()

    plotter.ensure_dir(RAW_DIR)
    plotter.ensure_dir(PARSED_DIR)
    plotter.ensure_dir(PLOTS_DIR)

    if args.run:
        runner.run_benchmarks(MODULES, args.count, args.bench, not args.no_benchmem, args.jobs, split=args.split)

    # 2. Parse Results
    all_rows = []
    for mod in MODULES:
        rows = parser.parse_file(mod["out"])
        for r in rows:
            r["module"] = mod["name"]
        all_rows.extend(rows)
        
    if not all_rows:
        print("No data found. Did you run with --run?")
        return

    # Write long CSV
    all_fields = set()
    for r in all_rows:
        all_fields.update(r.keys())
    
    header = [
        "module",
        "benchmark",
        "full_name",
        "keys",
        "ns_per_op",
        "bytes_per_op",
        "allocs_per_op",
    ]
    extra = sorted(k for k in all_fields if k not in header)
    header = header + extra

    long_path = os.path.join(PARSED_DIR, "bench_long.csv")
    with open(long_path, "w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=header)
        writer.writeheader()
        writer.writerows(all_rows)

    # 3. Aggregate
    # MMPH benchmarks use "keys" parameter (extracted as "keys" by the parser)
    agg_rows = parser.aggregate(all_rows, ["benchmark", "module", "keys"])
    
    # Map back to the metrics expected by the plotting logic if needed, 
    # but we'll use a more flexible plotting approach.
    
    # Define metrics to plot
    # (MetricKey, Title, YLabel, LogY)
    metrics_to_plot = [
        ("ns_per_op", "Time (ns/op)", "ns", True),
        ("allocs_per_op", "Allocs per Op", "allocs", True),
        ("bits_per_key_in_mem", "Bits per Key (in-mem)", "bits/key", False),
        ("bits_per_key", "Bits per Key", "bits/key", False),
        ("bytes_in_mem", "Bytes (in-mem)", "bytes", True),
    ]

    # Process metrics for plots
    # We need to separate build and lookup for some plots
    for metric, title_prefix, ylabel, log_y in metrics_to_plot:
        # Build plot
        build_series = defaultdict(list)
        lookup_series = defaultdict(list)
        
        for r in agg_rows:
            mod = r["module"]
            bench = r["benchmark"]
            val = r.get(metric)
            keys = r.get("keys")
            
            if val is None or keys is None:
                continue
                
            if bench in BUILD_BENCH.get(mod, set()):
                build_series[mod].append((keys, val))
            elif bench in LOOKUP_BENCH.get(mod, set()):
                lookup_series[mod].append((keys, val))

        if build_series:
            fname = f"build_{metric}.svg" if metric != "ns_per_op" else "build_time_ns.svg"
            plotter.draw_line_chart(
                os.path.join(PLOTS_DIR, fname),
                f"Build: {title_prefix}",
                "Key count",
                ylabel,
                build_series,
                log_x=True,
                log_y=log_y
            )
            
        if lookup_series:
            fname = f"lookup_{metric}.svg" if metric != "ns_per_op" else "lookup_time_ns.svg"
            plotter.draw_line_chart(
                os.path.join(PLOTS_DIR, fname),
                f"Lookup: {title_prefix}",
                "Key count",
                ylabel,
                lookup_series,
                log_x=True,
                log_y=log_y
            )

    # Print summary table for bits/key
    print("\nSummary: Bits per Key (in-mem)")
    # Collect bits per key from all possible keys (bits_per_key_in_mem or bits_per_key)
    bits_table = defaultdict(dict)
    all_keys = set()
    all_mods = set()
    
    for r in agg_rows:
        mod = r["module"]
        bench = r["benchmark"]
        keys = r.get("keys")
        if keys is None: continue
        
        val = r.get("bits_per_key_in_mem") or r.get("bits_per_key")
        if val is not None and bench in BUILD_BENCH.get(mod, set()):
            bits_table[keys][mod] = val
            all_keys.add(keys)
            all_mods.add(mod)
            
    if all_keys:
        sorted_keys = sorted(all_keys)
        sorted_mods = sorted(all_mods)
        
        header_str = f"{'Key Count':>12}"
        for mod in sorted_mods:
            header_str += f" | {mod:>25}"
        print(header_str)
        print("-" * len(header_str))
        
        for kc in sorted_keys:
            row_str = f"{kc:12d}"
            for mod in sorted_mods:
                val = bits_table[kc].get(mod)
                if val is not None:
                    row_str += f" | {val:25.2f}"
                else:
                    row_str += f" | {'-':>25}"
            print(row_str)

    print(f"\nWrote CSVs to: {PARSED_DIR}")
    print(f"SVG plots in: {PLOTS_DIR}")
    return 0

if __name__ == "__main__":
    sys.exit(main())