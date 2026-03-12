import sys
import os
import csv

# Add scripts directory to path for bench_lib
sys.path.append(os.path.join(os.getcwd(), "scripts"))
from bench_lib.plotter import draw_line_chart, ensure_dir

def main():
    csv_file = "emptiness/are_optimized/are_optimized_tradeoff.csv"
    if not os.path.exists(csv_file):
        print(f"CSV file not found: {csv_file}")
        return

    bpk_vals = []
    fpr_unif = []
    fpr_seq = []
    target_eps = []
    
    try:
        with open(csv_file, 'r') as f:
            reader = csv.DictReader(f)
            for row in reader:
                b = float(row['BitsPerKey'])
                # We use the same bpk for all series since they are measured from the same run
                bpk_vals.append(b)
                
                # Floor FPR at 1e-7 for log scale if it's 0
                fu = float(row['ActualFPR_Uniform'])
                fpr_unif.append((b, fu if fu > 0 else 1e-7))
                
                fs = float(row['ActualFPR_Sequential'])
                fpr_seq.append((b, fs if fs > 0 else 1e-7))
                
                te = float(row['TargetEpsilon'])
                target_eps.append((b, te))
    except Exception as e:
        print(f"Error reading CSV: {e}")
        return

    ensure_dir("bench_results/plots")
    output_path = "bench_results/plots/are_optimized_tradeoff.svg"
    
    series = {
        "Uniform Data (Empirical)": fpr_unif,
        "Sequential Data (Empirical)": fpr_seq,
        "Target Epsilon (Theoretical)": target_eps
    }
    
    styles = {
        "Target Epsilon (Theoretical)": "dashed"
    }
    
    draw_line_chart(
        path=output_path,
        title="Optimized Adaptive ARE: False Positive Rate vs Space",
        x_label="Bits per Key (bpk)",
        y_label="False Positive Rate (FPR)",
        series=series,
        styles=styles,
        log_y=True
    )
    
    print(f"Final tradeoff plot generated: {output_path}")
    
    # Cleanup old PNG if it exists
    old_png = "bench_results/plots/are_optimized_tradeoff.png"
    if os.path.exists(old_png):
        os.remove(old_png)
        print(f"Removed old PNG: {old_png}")

if __name__ == "__main__":
    main()
