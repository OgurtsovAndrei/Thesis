import sys
import os
from collections import defaultdict

sys.path.append(os.path.join(os.getcwd(), "scripts"))

from bench_lib.parser import parse_file
from bench_lib.plotter import draw_line_chart, ensure_dir

def main():
    results_file = "bench_results/approx_range_grid_large.txt"
    rows = parse_file(results_file)
    
    if not rows:
        print("No results found in", results_file)
        return

    # series for plots
    # We want to show performance across different N values
    memory_series = defaultdict(list) # Name -> [(L, bits)]
    query_series = defaultdict(list)  # Name -> [(L, ns)]
    throughput_series = defaultdict(list) # Name -> [(L, keys_per_sec)]
    
    for r in rows:
        name = r.get("full_name", "")
        keysize = r.get("keysize")
        keys_count = r.get("keys")
        
        if keysize is None or keys_count is None:
            continue
            
        n_label = f"N=2^{int(keys_count).bit_length()-1}"
        if keys_count == 262144: n_label = "N=262K"
        elif keys_count == 1048576: n_label = "N=1M"
        elif keys_count == 4194304: n_label = "N=4M"
        elif keys_count == 16777216: n_label = "N=16M"

        if "/Query" in name:
            query_series[n_label].append((keysize, r.get("ns_per_op", 0)))
        elif "/Build" in name:
            memory_series[n_label].append((keysize, r.get("bits_per_key", 0)))
            build_ns = r.get("ns_per_op", 0)
            if build_ns > 0:
                throughput = (1e9 / build_ns) * keys_count
                throughput_series[n_label].append((keysize, throughput))

    ensure_dir("bench_results/plots")
    
    # Plot 1: Query Latency (L vs Time)
    draw_line_chart(
        path="bench_results/plots/are_large_query_latency.svg",
        title="ARE Query Latency vs Key Size (L)",
        x_label="Key Size (bits)",
        y_label="Time (ns/op)",
        series=query_series
    )
    
    # Plot 2: Space Efficiency (L vs Bits/Key)
    draw_line_chart(
        path="bench_results/plots/are_large_bits_per_key.svg",
        title="ARE Space Efficiency vs Key Size (L)",
        x_label="Key Size (bits)",
        y_label="Bits per Key",
        series=memory_series
    )

    # Plot 3: Build Throughput
    draw_line_chart(
        path="bench_results/plots/are_large_build_throughput.svg",
        title="ARE Build Throughput vs Key Size (L)",
        x_label="Key Size (bits)",
        y_label="Keys/sec",
        series=throughput_series
    )
    
    print("Large grid plots generated in bench_results/plots/")

if __name__ == "__main__":
    main()
