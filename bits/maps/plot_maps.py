import re
import collections
import os
import sys

# Setup path to import shared library
current_dir = os.path.dirname(os.path.abspath(__file__))
# bits/maps/ -> bits/ -> root/
project_root = os.path.dirname(os.path.dirname(current_dir))
scripts_dir = os.path.join(project_root, "scripts")
sys.path.append(scripts_dir)

from bench_lib import plotter

def parse_bench(path):
    get_latency = collections.defaultdict(lambda: collections.defaultdict(list))
    put_latency = collections.defaultdict(lambda: collections.defaultdict(list))
    
    if not os.path.exists(path):
        return {}, {}

    with open(path, 'r') as f:
        for line in f:
            # BenchmarkMaps/BitMap_Get_N1000_L64-20
            m = re.match(r'BenchmarkMaps/(BitMap|ArrayBitMap)_(Get|Put)_N(\d+)_L(\d+)', line)
            if m:
                type_name, op, n, l = m.groups()
                n, l = int(n), int(l)
                
                ns_op_match = re.search(r'(\d+(?:\.\d+)?)\s+ns/op', line)
                if ns_op_match:
                    val = float(ns_op_match.group(1))
                    if op == "Get":
                        get_latency[l][type_name].append((float(n), val))
                    else:
                        # For Put, convert to ns per key (it's currently ns per full map build in the benchmark)
                        put_latency[l][type_name].append((float(n), val / n))

    return get_latency, put_latency

def main():
    bench_file = os.path.join(current_dir, 'bench_results.txt')
    get_latency, put_latency = parse_bench(bench_file)
    plot_dir = os.path.join(current_dir, 'plots')
    plotter.ensure_dir(plot_dir)

    # Colors for types
    COLORS = {
        "BitMap": "#2a7fff",      # Blue
        "ArrayBitMap": "#e4572e", # Orange/Red
    }

    # 1. Get Latency Charts (one per L)
    for l in sorted(get_latency.keys()):
        series = get_latency[l]
        path = os.path.join(plot_dir, f'get_latency_L{l}.svg')
        plotter.draw_line_chart(
            path,
            f"Lookup Performance (L={l} bits)",
            "Number of Keys (N)",
            "Latency (ns/op)",
            series,
            log_x=True,
            colors=COLORS
        )
        print(f"Generated: {path}")

    # 2. Put Latency Charts (one per L)
    for l in sorted(put_latency.keys()):
        series = put_latency[l]
        path = os.path.join(plot_dir, f'put_latency_L{l}.svg')
        plotter.draw_line_chart(
            path,
            f"Insertion Performance (L={l} bits)",
            "Number of Keys (N)",
            "Latency (ns/key)",
            series,
            log_x=True,
            colors=COLORS
        )
        print(f"Generated: {path}")

    # 3. Scaling by L (N=100,000)
    n_fixed = 100000
    scaling_series = collections.defaultdict(list)
    for l in sorted(get_latency.keys()):
        for type_name in get_latency[l]:
            for n, val in get_latency[l][type_name]:
                if int(n) == n_fixed:
                    scaling_series[type_name].append((float(l), val))
    
    path = os.path.join(plot_dir, f'scaling_by_L_N{n_fixed}.svg')
    plotter.draw_line_chart(
        path,
        f"Lookup Scaling vs Key Length (N={n_fixed})",
        "Key Length (L, bits)",
        "Latency (ns/op)",
        scaling_series,
        log_x=True,
        colors=COLORS
    )
    print(f"Generated: {path}")

if __name__ == "__main__":
    main()
