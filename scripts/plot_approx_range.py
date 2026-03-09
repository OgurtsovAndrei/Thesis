import sys
import os

sys.path.append(os.path.join(os.getcwd(), "scripts"))

from bench_lib.parser import parse_file
from bench_lib.plotter import draw_line_chart, ensure_dir

def main():
    results_file = "bench_results/approx_range_emptiness.txt"
    rows = parse_file(results_file)
    
    if not rows:
        print("No results found in", results_file)
        return

    # Filter Query and Build results
    query_pts_01 = []
    bits_pts_01 = []
    query_pts_001 = []
    bits_pts_001 = []
    
    for r in rows:
        name = r.get("full_name", "")
        keysize = r.get("keysize")
        eps = r.get("eps")
        if keysize is None or eps is None:
            continue
            
        if "/Query" in name:
            if eps == 0.01:
                query_pts_01.append((keysize, r.get("ns_per_op", 0)))
            elif eps == 0.001:
                query_pts_001.append((keysize, r.get("ns_per_op", 0)))
        elif "/Build" in name:
            if eps == 0.01:
                bits_pts_01.append((keysize, r.get("bits_per_key", 0)))
            elif eps == 0.001:
                bits_pts_001.append((keysize, r.get("bits_per_key", 0)))

    ensure_dir("bench_results/plots")
    
    # Plot 1: Query Latency
    draw_line_chart(
        path="bench_results/plots/approx_range_query_latency.svg",
        title="ARE: Query Latency (1M keys)",
        x_label="Original Bit Length (L)",
        y_label="Time (ns/op)",
        series={"Eps=0.01": query_pts_01, "Eps=0.001": query_pts_001}
    )
    
    # Plot 2: Space Efficiency
    draw_line_chart(
        path="bench_results/plots/approx_range_bits_per_key.svg",
        title="ARE: Bits per Key (1M keys)",
        x_label="Original Bit Length (L)",
        y_label="Bits per Key",
        series={"Eps=0.01": bits_pts_01, "Eps=0.001": bits_pts_001}
    )
    
    print("Plots generated in bench_results/plots/")

if __name__ == "__main__":
    main()
