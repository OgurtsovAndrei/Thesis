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

    series_data = {
        "Adaptive ARE (Unif)": [],
        "Adaptive ARE (Seq)": [],
        "Original SODA (Unif)": [],
        "Original SODA (Seq)": [],
        "Truncation ARE (Unif)": [],
        "Truncation ARE (Seq)": [],
        "Theoretical Target": []
    }
    
    try:
        with open(csv_file, 'r') as f:
            reader = csv.DictReader(f)
            for row in reader:
                eps = float(row['TargetEpsilon'])
                
                # Adaptive
                b_opt = float(row['BPK_Opt'])
                series_data["Adaptive ARE (Unif)"].append((b_opt, float(row['FPR_Opt_Unif']) or 1e-7))
                series_data["Adaptive ARE (Seq)"].append((b_opt, float(row['FPR_Opt_Seq']) or 1e-7))
                series_data["Theoretical Target"].append((b_opt, eps))
                
                # SODA
                b_soda = float(row['BPK_Soda'])
                series_data["Original SODA (Unif)"].append((b_soda, float(row['FPR_Soda_Unif']) or 1e-7))
                series_data["Original SODA (Seq)"].append((b_soda, float(row['FPR_Soda_Seq']) or 1e-7))
                
                # Truncation
                b_trunc = float(row['BPK_Trunc'])
                series_data["Truncation ARE (Unif)"].append((b_trunc, float(row['FPR_Trunc_Unif']) or 1e-7))
                series_data["Truncation ARE (Seq)"].append((b_trunc, float(row['FPR_Trunc_Seq']) or 1e-7))
                
    except Exception as e:
        print(f"Error reading CSV: {e}")
        return

    ensure_dir("bench_results/plots")
    output_path = "bench_results/plots/are_full_comparison.svg"
    
    final_series = {}
    for name, pts in series_data.items():
        if pts:
            pts.sort()
            final_series[name] = pts

    colors = {
        "Adaptive ARE (Unif)": "#2a7fff",   # Blue
        "Adaptive ARE (Seq)": "#0044aa",    # Dark Blue
        "Original SODA (Unif)": "#22a06b", # Green
        "Original SODA (Seq)": "#005522",  # Dark Green
        "Truncation ARE (Unif)": "#ffcc00",# Yellow
        "Truncation ARE (Seq)": "#e4572e", # Orange
        "Theoretical Target": "#ef4444"    # Red
    }
    
    styles = {
        "Theoretical Target": "dashed",
        "Adaptive ARE (Seq)": "dashed",
        "Original SODA (Seq)": "dashed",
        "Truncation ARE (Seq)": "dashed"
    }
    
    draw_line_chart(
        path=output_path,
        title="Range Emptiness Filters: 3 Architectures x 2 Data Types",
        x_label="Bits per Key (bpk)",
        y_label="False Positive Rate (FPR)",
        series=final_series,
        styles=styles,
        colors=colors,
        log_y=True
    )
    
    print(f"Full comparison plot generated: {output_path}")

if __name__ == "__main__":
    main()
