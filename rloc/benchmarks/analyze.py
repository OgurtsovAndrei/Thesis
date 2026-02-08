#!/usr/bin/env python3
import argparse
import csv
import os
import sys
from collections import defaultdict
from typing import Dict, List, Tuple

# Setup path to import shared library
current_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(os.path.dirname(current_dir))
scripts_dir = os.path.join(project_root, "scripts")
sys.path.append(scripts_dir)

from bench_lib import runner, parser, plotter

# --- Configuration ---

RLOC_DIR = os.path.dirname(current_dir)
RAW_DIR = os.path.join(current_dir, "raw")
PARSED_DIR = os.path.join(current_dir, "parsed")
PLOTS_DIR = os.path.join(current_dir, "plots")

MODULES = [
    {
        "name": "rloc",
        "dir": RLOC_DIR,
        "out": os.path.join(RAW_DIR, "rloc.txt"),
    }
]

# --- Main Logic ---

def main() -> int:
    arg_parser = argparse.ArgumentParser()
    arg_parser.add_argument("--run", action="store_true", help="Run benchmarks")
    arg_parser.add_argument("--count", type=int, default=5, help="Number of iterations per benchmark")
    arg_parser.add_argument("--jobs", "-j", type=int, default=None, help="Parallel jobs")
    arg_parser.add_argument("--bench", default=".", help="Benchmark regex")
    arg_parser.add_argument("--split", action="store_true", help="Split benchmarks into individual tasks for max parallelism")
    arg_parser.add_argument("--no-benchmem", action="store_true", help="Disable -benchmem")
    args = arg_parser.parse_args()
    
    plotter.ensure_dir(RAW_DIR)
    plotter.ensure_dir(PARSED_DIR)
    plotter.ensure_dir(PLOTS_DIR)
    
    # 1. Run Benchmarks
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
        return 1

    # Write full raw csv
    all_fieldnames = set()
    for r in all_rows:
        all_fieldnames.update(r.keys())

    priority_fields = ["benchmark", "module", "keysize", "keys", "prefixlen"]
    fieldnames = priority_fields + [f for f in sorted(all_fieldnames) if f not in priority_fields]
    
    with open(os.path.join(PARSED_DIR, "all_runs.csv"), "w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(all_rows)

    # 3. Aggregate
    # Grouping by Benchmark Name, KeySize, Keys, PrefixLen
    agg_rows = parser.aggregate(all_rows, ["benchmark", "keysize", "keys", "prefixlen"])
    
    agg_fieldnames = set()
    for r in agg_rows:
        agg_fieldnames.update(r.keys())

    with open(os.path.join(PARSED_DIR, "agg.csv"), "w", newline="") as f:
        if agg_rows:
            agg_fieldnames = set()
            for r in agg_rows:
                agg_fieldnames.update(r.keys())
            writer = csv.DictWriter(f, fieldnames=sorted(agg_fieldnames))
            writer.writeheader()
            writer.writerows(agg_rows)
        
    print(f"Parsed {len(all_rows)} rows, aggregated into {len(agg_rows)} datapoints.")

    # 4. Generate Plots
    
    # Filter datasets
    build_rows = [r for r in agg_rows if "Build" in str(r["benchmark"])]
    query_rows = [r for r in agg_rows if "Query" in str(r["benchmark"]) or "Search" in str(r["benchmark"])]
    mem_rows = [r for r in agg_rows if "MemoryComparison" in str(r["benchmark"])]

    # --- Plot 1: Build Time vs N (Log-Log) ---
    # Series by KeySize
    series_build_time: Dict[str, List[Tuple[float, float]]] = defaultdict(list)
    for r in build_rows:
        if r.get("keysize") and r.get("keys"):
            series_build_time[f"KeySize={int(r['keysize'])}"].append((float(r["keys"]), float(r.get("ns_per_op", 0))))
            
    plotter.draw_line_chart(
        os.path.join(PLOTS_DIR, "build_time_ns.svg"),
        "Build Time (ns/op)",
        "Keys (N)",
        "ns/op",
        series_build_time,
        log_x=True,
        log_y=True
    )

    # --- Plot 2: Bits/Key vs N (Log-X) ---
    series_bits: Dict[str, List[Tuple[float, float]]] = defaultdict(list)
    for r in build_rows:
        # Check for bits_per_key metric
        bpk = r.get("bits_per_key")
        if bpk and r.get("keysize") and r.get("keys"):
             series_bits[f"KeySize={int(r['keysize'])}"].append((float(r["keys"]), float(bpk)))

    if series_bits:
        plotter.draw_line_chart(
            os.path.join(PLOTS_DIR, "bits_per_key.svg"),
            "Bits per Key",
            "Keys (N)",
            "bits/key",
            series_bits,
            log_x=True,
            log_y=False # Usually linear is fine for bits/key
        )

    # --- Plot 3: RLOC vs LERLOC Comparison (Bits/Key) ---
    # From BenchmarkMemoryComparison, we have rl_bits_per_key and lerl_bits_per_key in the same row
    # We will pick one KeySize (e.g., 64 or 128) or plot all if not too messy.
    # Let's plot for KeySize=64 and KeySize=128 separately if available.
    
    for ksize in [64, 128, 256]:
        series_k: Dict[str, List[Tuple[float, float]]] = defaultdict(list)
        rows_k = [r for r in mem_rows if int(r.get("keysize", 0)) == ksize]
        if not rows_k: continue
        
        for r in rows_k:
            if r.get("rl_bits_per_key"):
                series_k["RLOC"].append((float(r["keys"]), float(r["rl_bits_per_key"])))
            if r.get("lerl_bits_per_key"):
                series_k["LERLOC"].append((float(r["keys"]), float(r["lerl_bits_per_key"])))
        
        if series_k:
            plotter.draw_line_chart(
                os.path.join(PLOTS_DIR, f"comparison_bits_key_{ksize}.svg"),
                f"RLOC vs LERLOC Memory (KeySize={ksize})",
                "Keys (N)",
                "bits/key",
                series_k,
                log_x=True,
                log_y=False
            )

    print("Plots generated in rloc/benchmarks/plots/")
    return 0

if __name__ == "__main__":
    main()
