import sys
import os

sys.path.append(os.path.join(os.getcwd(), "scripts"))
from bench_lib.plotter import draw_line_chart, ensure_dir

def main():
    csv_file = "bench_results/are_final_smooth_data.csv"
    if not os.path.exists(csv_file):
        print("CSV not found")
        return

    all_pts = []
    with open(csv_file, "r") as f:
        for line in f:
            parts = line.strip().split(",")
            if len(parts) == 4:
                try:
                    # parts: N, K, BitsPerKey, ActualFPR
                    bits = float(parts[2])
                    fpr = float(parts[3])
                    if fpr <= 0: fpr = 1e-7 # Even smaller floor for 1M queries
                    all_pts.append((bits, fpr))
                except:
                    continue

    all_pts.sort()

    ensure_dir("bench_results/plots")
    draw_line_chart(
        path="bench_results/plots/are_tradeoff_final.svg",
        title="Approximate Range Emptiness: FPR vs Space (1M Queries/point)",
        x_label="Bits per Key",
        y_label="False Positive Rate",
        series={"Empirical Tradeoff": all_pts},
        log_y=True
    )
    print("Final tradeoff plot generated: bench_results/plots/are_tradeoff_final.svg")

if __name__ == "__main__":
    main()
