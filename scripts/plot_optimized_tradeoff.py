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

    pts_opt_unif = []
    pts_opt_seq = []
    pts_soda_seq = []
    pts_trunc_seq = []
    pts_target = []
    
    try:
        with open(csv_file, 'r') as f:
            reader = csv.DictReader(f)
            for row in reader:
                eps = float(row['TargetEpsilon'])
                
                # Optimized Adaptive ARE (Unif)
                b_opt = float(row['BPK_Opt'])
                f_opt_u = float(row['FPR_Opt_Unif'])
                pts_opt_unif.append((b_opt, f_opt_u if f_opt_u > 0 else 1e-7))
                
                # Optimized Adaptive ARE (Seq)
                f_opt_s = float(row['FPR_Opt_Seq'])
                pts_opt_seq.append((b_opt, f_opt_s if f_opt_s > 0 else 1e-7))
                
                # Original SODA ARE
                b_soda = float(row['BPK_Soda'])
                f_soda = float(row['FPR_Soda_Seq'])
                pts_soda_seq.append((b_soda, f_soda if f_soda > 0 else 1e-7))
                
                # Truncation ARE
                b_trunc = float(row['BPK_Trunc'])
                f_trunc = float(row['FPR_Trunc_Seq'])
                pts_trunc_seq.append((b_trunc, f_trunc if f_trunc > 0 else 1e-7))
                
                pts_target.append((b_opt, eps))
                
    except Exception as e:
        print(f"Error reading CSV: {e}")
        return

    ensure_dir("bench_results/plots")
    output_path = "bench_results/plots/are_comparison_tradeoff.svg"
    
    series = {
        "Adaptive ARE (Uniform)": pts_opt_unif,
        "Adaptive ARE (Sequential)": pts_opt_seq,
        "Original SODA (Sequential)": pts_soda_seq,
        "Truncation ARE (Sequential)": pts_trunc_seq,
        "Theoretical Target": pts_target
    }
    
    styles = {
        "Theoretical Target": "dashed"
    }
    
    draw_line_chart(
        path=output_path,
        title="Range Emptiness Filters Comparison: FPR vs Space",
        x_label="Bits per Key (bpk)",
        y_label="False Positive Rate (FPR)",
        series=series,
        styles=styles,
        log_y=True
    )
    
    print(f"Comparison plot generated: {output_path}")

if __name__ == "__main__":
    main()
