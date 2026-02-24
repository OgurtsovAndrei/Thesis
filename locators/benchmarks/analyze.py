#!/usr/bin/env python3
import argparse
import csv
import json
import os
import sys
from collections import defaultdict
from typing import Any, Dict, List, Tuple

# Setup path to import shared library
current_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(os.path.dirname(current_dir))
scripts_dir = os.path.join(project_root, "scripts")
sys.path.append(scripts_dir)

from bench_lib import runner, parser, plotter

# --- Configuration ---

RLOC_DIR = os.path.join(os.path.dirname(current_dir), "rloc")
LERLOC_DIR = os.path.join(os.path.dirname(current_dir), "lerloc")
RAW_DIR = os.path.join(current_dir, "raw")
PARSED_DIR = os.path.join(current_dir, "parsed")
PLOTS_DIR = os.path.join(current_dir, "plots")

MODULES = [
    {
        "name": "rloc",
        "dir": RLOC_DIR,
        "out": os.path.join(RAW_DIR, "rloc.txt"),
    },
    {
        "name": "lerloc",
        "dir": LERLOC_DIR,
        "out": os.path.join(RAW_DIR, "lerloc.txt"),
    }
]

# --- Helpers ---

def parse_mem_reports(path: str) -> Dict[int, Dict[str, int]]:
    """
    Parses JSON_MEM_REPORT lines from benchmark output.
    Returns mapping: n_keys -> {component_name: bytes}
    """
    reports = {}
    if not os.path.exists(path):
        return reports

    current_keys = 0
    with open(path, "r") as f:
        for line in f:
            # Detect key count from Benchmark line
            if "BenchmarkMemoryDetailed/Keys=" in line:
                parts = line.split("/")
                for p in parts:
                    if p.startswith("Keys="):
                        current_keys = int(p.split("=")[1].split("-")[0])
            
            if "JSON_MEM_REPORT:" in line:
                json_str = line.split("JSON_MEM_REPORT:")[1].strip()
                try:
                    data = json.loads(json_str)
                    reports[current_keys] = flatten_report(data)
                except Exception as e:
                    print(f"Failed to parse JSON report for N={current_keys}: {e}")
    return reports

def flatten_report(data: Dict[str, Any]) -> Dict[str, int]:
    """Flattens hierarchical MemReport into key components for LERLOC."""
    out = {}
    
    # Root level is usually autoLocalExactRangeLocator
    out["header"] = data.get("total_bytes", 0)
    
    # We want to identify specific sub-components
    children = data.get("children", [])
    for child in children:
        name = child["name"]
        if name == "hzft":
            out["HZFastTrie"] = child["total_bytes"]
        elif name == "GenericRangeLocator":
            # Breakdown RangeLocator
            rl_children = child.get("children", [])
            for rlc in rl_children:
                rlc_name = rlc["name"]
                if rlc_name == "MonotoneHashWithTrie":
                    # Breakdown MMPH
                    mmph_children = rlc.get("children", [])
                    for mc in mmph_children:
                        mc_name = mc["name"]
                        if mc_name == "ApproxZFastTrie":
                            out["MMPH_Trie"] = mc["total_bytes"]
                        elif mc_name == "buckets":
                            out["MMPH_Buckets"] = mc["total_bytes"]
                elif rlc_name == "rsdic_bv":
                    out["Leaf_BitVector"] = rlc["total_bytes"]
        elif name == "header":
            out["Top_Level_Header"] = child["total_bytes"]

    # Calculate 'other' if needed to match total_bytes
    known_sum = sum(v for k, v in out.items() if k != "header")
    out["Other"] = max(0, data.get("total_bytes", 0) - known_sum)
    # Remove the root 'header' which was total_bytes
    if "header" in out: del out["header"]
    
    return out

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

    # --- Plot 4: Detailed Memory Breakdown (Stacked Area) ---
    for mod in MODULES:
        if mod["name"] == "lerloc":
            detailed_reports = parse_mem_reports(mod["out"])
            if detailed_reports:
                series_breakdown: Dict[str, List[Tuple[float, float]]] = defaultdict(list)
                # Sort components for consistent stacking (bottom to top)
                components = ["Top_Level_Header", "HZFastTrie", "Leaf_BitVector", "MMPH_Trie", "MMPH_Buckets", "Other"]
                
                sorted_n = sorted(detailed_reports.keys())
                for n in sorted_n:
                    comp_data = detailed_reports[n]
                    for comp in components:
                        series_breakdown[comp].append((float(n), float(comp_data.get(comp, 0))))
                
                plotter.draw_stacked_area_chart(
                    os.path.join(PLOTS_DIR, "lerloc_memory_breakdown.svg"),
                    "LERLOC Memory Usage Breakdown (KeySize=64)",
                    "Keys (N)",
                    "Bytes",
                    series_breakdown,
                    log_x=True
                )

                # --- Plot 5: Detailed Memory Breakdown (Bits/Key Stacked Area) ---
                series_bits_breakdown: Dict[str, List[Tuple[float, float]]] = defaultdict(list)
                for n in sorted_n:
                    comp_data = detailed_reports[n]
                    for comp in components:
                        # Convert bytes to bits and divide by n
                        bits_per_key = (float(comp_data.get(comp, 0)) * 8.0) / float(n)
                        series_bits_breakdown[comp].append((float(n), bits_per_key))
                
                plotter.draw_stacked_area_chart(
                    os.path.join(PLOTS_DIR, "lerloc_memory_bits_breakdown.svg"),
                    "LERLOC Memory Efficiency Breakdown (KeySize=64)",
                    "Keys (N)",
                    "bits/key",
                    series_bits_breakdown,
                    log_x=True
                )

                # --- Export CSV: Detailed Memory Breakdown (Bits/Key) ---
                csv_path = os.path.join(PARSED_DIR, "lerloc_mem_breakdown.csv")
                with open(csv_path, "w", newline="") as f:
                    fieldnames = ["Keys"] + components + ["Total_Bits_Per_Key"]
                    writer = csv.DictWriter(f, fieldnames=fieldnames)
                    writer.writeheader()
                    for n in sorted_n:
                        row = {"Keys": n}
                        total_bpk = 0.0
                        for comp in components:
                            bpk = (float(detailed_reports[n].get(comp, 0)) * 8.0) / float(n)
                            row[comp] = round(bpk, 4)
                            total_bpk += bpk
                        row["Total_Bits_Per_Key"] = round(total_bpk, 4)
                        writer.writerow(row)
                print(f"Detailed CSV breakdown saved to {csv_path}")

    print("Plots generated in locators/benchmarks/plots/")
    return 0

if __name__ == "__main__":
    main()
