import json
import re
import collections
import os
import sys

# Setup path to import shared library
current_dir = os.path.dirname(os.path.abspath(__file__))
# trie/benchmarks/shzft/ -> trie/benchmarks/ -> trie/ -> root/
project_root = os.path.dirname(os.path.dirname(os.path.dirname(current_dir)))
scripts_dir = os.path.join(project_root, "scripts")
sys.path.append(scripts_dir)

from bench_lib import plotter

def parse_bench(path):
    # Use dict to deduplicate points: (L, N, type_name) -> Value
    bits_per_key_map = {}
    query_latency_map = {}
    
    if not os.path.exists(path):
        return {}, {}

    with open(path, 'r') as f:
        current_bench = None
        for line in f:
            m = re.match(r'BenchmarkMemory/(HZFT|SHZFT)/L=(\d+)/N=(\d+)', line)
            if m:
                type_name, l, n = m.groups()
                l, n = int(l), int(n)
                if n < 1024: continue # Filter N < 1024
                current_bench = (type_name, l, n)
            
            if "JSON_MEM_REPORT:" in line and current_bench:
                json_str = line.split("JSON_MEM_REPORT:")[1].strip()
                try:
                    report = json.loads(json_str)
                    type_name, l, n = current_bench
                    bits_per_key_map[(l, n, type_name)] = report['total_bytes'] * 8 / n
                except:
                    pass

    # Parse Queries
    with open(path, 'r') as f:
        for line in f:
            m = re.match(r'BenchmarkQuery/(HZFT|SHZFT)/Query/L=(\d+)/N=(\d+)', line)
            if m:
                type_name, l, n = m.groups()
                l, n = int(l), int(n)
                if n < 1024: continue
                ns_op_match = re.search(r'(\d+)\s+ns/op', line)
                if ns_op_match:
                    query_latency_map[(l, n, type_name)] = float(ns_op_match.group(1))

    # Convert back to series for plotter
    bits_per_key = collections.defaultdict(lambda: collections.defaultdict(list))
    for (l, n, type_name), val in bits_per_key_map.items():
        bits_per_key[l][type_name].append((float(n), val))

    query_latency = collections.defaultdict(lambda: collections.defaultdict(list))
    for (l, n, type_name), val in query_latency_map.items():
        query_latency[l][type_name].append((float(n), val))

    return bits_per_key, query_latency

def main():
    bench_file = os.path.join(current_dir, 'bench_results_detailed.txt')
    bits_per_key, query_latency = parse_bench(bench_file)
    plot_dir = os.path.join(current_dir, 'plots')
    plotter.ensure_dir(plot_dir)

    # Color mapping for key lengths
    # Blue, Green, Orange, Purple, Red, Cyan
    L_COLORS = {
        64: "#2a7fff",
        256: "#22a06b",
        1024: "#e4572e",
        4096: "#7c3aed"
    }

    # 1. Consolidated Bits Per Key Chart (by N)
    all_bpk_series = {}
    styles = {}
    colors = {}
    for l in sorted(bits_per_key.keys()):
        if l not in L_COLORS: continue
        hzft_name = f"HZFT (L={l})"
        shzft_name = f"SHZFT (L={l})"
        color = L_COLORS[l]
        
        if "HZFT" in bits_per_key[l]:
            all_bpk_series[hzft_name] = bits_per_key[l]["HZFT"]
            styles[hzft_name] = "dashed"
            colors[hzft_name] = color
        if "SHZFT" in bits_per_key[l]:
            all_bpk_series[shzft_name] = bits_per_key[l]["SHZFT"]
            colors[shzft_name] = color
    
    path = os.path.join(plot_dir, 'bits_per_key_all.svg')
    plotter.draw_line_chart(
        path, 
        "Trie Memory Efficiency: Bits/Key vs N", 
        "Number of Keys (N)", 
        "Bits Per Key", 
        all_bpk_series, 
        log_x=True,
        styles=styles,
        colors=colors
    )
    print(f"Generated: {path}")

    # 1b. Bits Per Key by L (New Chart)
    # Regroup data: N -> Type -> [(L, Val)]
    bits_by_n = collections.defaultdict(lambda: collections.defaultdict(list))
    for l, types in bits_per_key.items():
        for t_name, pts in types.items():
            for n, val in pts:
                bits_by_n[int(n)][t_name].append((float(l), val))

    N_COLORS = {
        1024: "#2a7fff",
        8192: "#22a06b",
        32768: "#e4572e",
        262144: "#7c3aed"
    }
    
    bpk_l_series = {}
    bpk_l_styles = {}
    bpk_l_colors = {}
    for n in sorted(bits_by_n.keys()):
        if n not in N_COLORS: continue
        hzft_name = f"HZFT (N={n})"
        shzft_name = f"SHZFT (N={n})"
        color = N_COLORS[n]
        
        if "HZFT" in bits_by_n[n]:
            bpk_l_series[hzft_name] = bits_by_n[n]["HZFT"]
            bpk_l_styles[hzft_name] = "dashed"
            bpk_l_colors[hzft_name] = color
        if "SHZFT" in bits_by_n[n]:
            bpk_l_series[shzft_name] = bits_by_n[n]["SHZFT"]
            bpk_l_colors[shzft_name] = color

    path_l = os.path.join(plot_dir, 'bits_per_key_by_L.svg')
    plotter.draw_line_chart(
        path_l,
        "Memory Scaling: Bits/Key vs Key Length (L)",
        "Key Length (L, bits)",
        "Bits Per Key",
        bpk_l_series,
        log_x=True, # L values are 64, 256, 1024, 4096 (powers of 4/2)
        styles=bpk_l_styles,
        colors=bpk_l_colors
    )
    print(f"Generated: {path_l}")

    # 2. Consolidated Query Latency Chart
    all_query_series = {}
    q_styles = {}
    q_colors = {}
    for l in sorted(query_latency.keys()):
        if l not in L_COLORS: continue
        hzft_name = f"HZFT (L={l})"
        shzft_name = f"SHZFT (L={l})"
        color = L_COLORS[l]
        
        if "HZFT" in query_latency[l]:
            all_query_series[hzft_name] = query_latency[l]["HZFT"]
            q_styles[hzft_name] = "dashed"
            q_colors[hzft_name] = color
        if "SHZFT" in query_latency[l]:
            all_query_series[shzft_name] = query_latency[l]["SHZFT"]
            q_colors[shzft_name] = color
    
    path = os.path.join(plot_dir, 'query_latency_all.svg')
    plotter.draw_line_chart(
        path, 
        "Query Performance: HZFT vs SHZFT", 
        "Number of Keys (N)", 
        "Latency (ns/op)", 
        all_query_series, 
        log_x=True,
        log_y=True,
        styles=q_styles,
        colors=q_colors
    )
    print(f"Generated: {path}")

if __name__ == "__main__":
    main()
