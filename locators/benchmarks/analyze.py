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

LOCATORS_DIR = os.path.dirname(current_dir)
MMPH_DIR = os.path.join(project_root, "mmph", "relative_trie")
RAW_DIR = os.path.join(current_dir, "raw")
PARSED_DIR = os.path.join(current_dir, "parsed")
PLOTS_DIR = os.path.join(current_dir, "plots")

MODULES = [
    {
        "name": "rloc",
        "dir": os.path.join(LOCATORS_DIR, "rloc"),
        "out": os.path.join(RAW_DIR, "rloc.txt"),
    },
    {
        "name": "lerloc",
        "dir": os.path.join(LOCATORS_DIR, "lerloc"),
        "out": os.path.join(RAW_DIR, "lerloc.txt"),
    },
    {
        "name": "mmph",
        "dir": MMPH_DIR,
        "out": os.path.join(RAW_DIR, "mmph.txt"),
    }
]

# --- Helpers ---

def parse_mem_reports(path: str) -> Dict[str, Dict[int, Dict[str, int]]]:
    """
    Parses JSON_MEM_REPORT lines from benchmark output.
    Returns mapping: mode -> n_keys -> {component_name: bytes}
    """
    reports = defaultdict(dict)
    if not os.path.exists(path):
        return reports

    current_keys = 0
    current_mode = "default"
    with open(path, "r") as f:
        for line in f:
            # Detect mode and key count from Benchmark line
            if line.startswith("BenchmarkMemoryDetailed/"):
                # Remove suffix like -20 or benchmark values
                bench_part = line.split()[0]
                parts = bench_part.split("/")
                
                # Reset current_mode for every new benchmark line
                current_mode = "default"
                
                # Check for explicit mode like Fast or Compact
                if len(parts) >= 3:
                    # The part after BenchmarkMemoryDetailed is the mode
                    # if it's not the Keys part.
                    if not parts[1].startswith("Keys="):
                        current_mode = parts[1]
                
                for p in parts:
                    if p.startswith("Keys="):
                        try:
                            current_keys = int(p.split("=")[1].split("-")[0])
                        except: pass
            
            if "JSON_MEM_REPORT:" in line:
                json_str = line.split("JSON_MEM_REPORT:")[1].strip()
                try:
                    data = json.loads(json_str)
                    reports[current_mode][current_keys] = flatten_report(data)
                except Exception as e:
                    print(f"Failed to parse JSON report for {current_mode}/N={current_keys}: {e}")
    return reports

def flatten_report(data: Dict[str, Any]) -> Dict[str, int]:
    """
    Flattens hierarchical MemReport into key components.
    Handles both LERLOC and standalone MMPH.
    """
    out = {
        "HZFastTrie": 0,
        "Leaf_BitVector": 0,
        "MMPH_Trie": 0,
        "MMPH_Buckets": 0,
        "Other": 0
    }
    
    total = data.get("total_bytes", 0)
    
    def walk(node):
        name = node["name"]
        if name == "hzft":
            out["HZFastTrie"] += node["total_bytes"]
            return # Stop walking sub-components of HZFT
        elif name == "rsdic_bv":
            out["Leaf_BitVector"] += node["total_bytes"]
            return # Stop walking sub-components of RSDic
        elif name == "ApproxZFastTrie":
            out["MMPH_Trie"] += node["total_bytes"]
            return
        elif name == "buckets":
            out["MMPH_Buckets"] += node["total_bytes"]
            return
        elif name in ["header", "Top_Level_Header"]:
            out["Other"] += node["total_bytes"]
        
        for child in node.get("children", []):
            walk(child)

    walk(data)

    # Calculate final 'Other' by ensuring sum matches total (handling any missed pieces)
    known_sum = sum(v for k, v in out.items() if k != "Other") + out["Other"]
    if known_sum < total:
        out["Other"] += (total - known_sum)
    
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
    series_build_time: Dict[str, List[Tuple[float, float]]] = defaultdict(list)
    for r in build_rows:
        if r.get("keysize") and r.get("keys"):
            mode = ""
            if "Fast" in str(r["benchmark"]): mode = " (Fast)"
            if "Compact" in str(r["benchmark"]): mode = " (Compact)"
            series_build_time[f"{r['module'].upper()}{mode} KeySize={int(r['keysize'])}"].append((float(r["keys"]), float(r.get("ns_per_op", 0))))
            
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
        bpk = r.get("bits_per_key") or r.get("bits_key_in_mem")
        if bpk and r.get("keys"):
             mode = ""
             if "Fast" in str(r["benchmark"]): mode = " (Fast)"
             if "Compact" in str(r["benchmark"]): mode = " (Compact)"
             series_bits[f"{r['module'].upper()}{mode}"].append((float(r["keys"]), float(bpk)))

    if series_bits:
        plotter.draw_line_chart(
            os.path.join(PLOTS_DIR, "bits_per_key.svg"),
            "Bits per Key",
            "Keys (N)",
            "bits/key",
            series_bits,
            log_x=True,
            log_y=False
        )

    # --- Plot 3: RLOC vs LERLOC Comparison (Bits/Key) ---
    for ksize in [64, 128, 256]:
        series_k: Dict[str, List[Tuple[float, float]]] = defaultdict(list)
        rows_k = [r for r in mem_rows if int(r.get("keysize", 0)) == ksize]
        if not rows_k: continue
        
        for r in rows_k:
            if r.get("rl_bits_per_key"):
                series_k["RLOC"].append((float(r["keys"]), float(r.get("rl_bits_per_key"))))
            if r.get("lerl_bits_per_key"):
                series_k["LERLOC (Default)"].append((float(r["keys"]), float(r.get("lerl_bits_per_key"))))
            if r.get("fast_bits_key"):
                series_k["LERLOC (Fast)"].append((float(r["keys"]), float(r.get("fast_bits_key"))))
            if r.get("compact_bits_key"):
                series_k["LERLOC (Compact)"].append((float(r["keys"]), float(r.get("compact_bits_key"))))
        
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

    # --- Plot 4 & 5: Detailed Memory Breakdown (Consolidated) ---
    detailed_components = ["Other", "HZFastTrie", "Leaf_BitVector", "MMPH_Trie", "MMPH_Buckets"]
    
    all_breakdown_rows = []
    efficiency_series = defaultdict(list)

    for mod in MODULES:
        reports_by_mode = parse_mem_reports(mod["out"])
        if not reports_by_mode:
            continue
            
        for mode, detailed_reports in reports_by_mode.items():
            m_name = mod["name"].upper()
            if mode != "default":
                m_name += f" ({mode})"
            
            sorted_n = sorted(detailed_reports.keys())
            
            for n in sorted_n:
                comp_data = detailed_reports[n]
                row = {"Module": mod["name"], "Mode": mode, "Keys": n}
                total_bpk = 0.0
                for comp in detailed_components:
                    bpk = (float(comp_data.get(comp, 0)) * 8.0) / float(n)
                    row[comp] = round(bpk, 4)
                    total_bpk += bpk
                row["Total_Bits_Per_Key"] = round(total_bpk, 4)
                all_breakdown_rows.append(row)
                
                efficiency_series[m_name].append((float(n), total_bpk))

    # Export Consolidated CSV
    if all_breakdown_rows:
        csv_path = os.path.join(PARSED_DIR, "mem_breakdown.csv")
        with open(csv_path, "w", newline="") as f:
            fieldnames = ["Module", "Mode", "Keys"] + detailed_components + ["Total_Bits_Per_Key"]
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(all_breakdown_rows)
        print(f"Consolidated memory breakdown saved to {csv_path}")

    # Export Consolidated Efficiency Plot (Line Chart)
    if efficiency_series:
        plotter.draw_line_chart(
            os.path.join(PLOTS_DIR, "mem_efficiency_all.svg"),
            "Memory Efficiency Comparison",
            "Keys (N)",
            "bits/key",
            efficiency_series,
            log_x=True,
            log_y=False
        )
        print(f"Consolidated efficiency plot saved to {PLOTS_DIR}/mem_efficiency_all.svg")

    print("Consolidated reports generated in locators/benchmarks/")
    return 0

if __name__ == "__main__":
    main()
