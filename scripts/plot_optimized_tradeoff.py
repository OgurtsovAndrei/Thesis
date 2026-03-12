import csv
import matplotlib.pyplot as plt
import os

def main():
    csv_file = "emptiness/are_optimized/are_optimized_tradeoff.csv"
    if not os.path.exists(csv_file):
        print(f"CSV file not found: {csv_file}")
        return

    bpk = []
    fpr_unif = []
    fpr_seq = []
    target_eps = []
    
    with open(csv_file, 'r') as f:
        reader = csv.DictReader(f)
        for row in reader:
            bpk.append(float(row['BitsPerKey']))
            fpr_unif.append(float(row['ActualFPR_Uniform']) if float(row['ActualFPR_Uniform']) > 0 else 1e-6)
            fpr_seq.append(float(row['ActualFPR_Sequential']) if float(row['ActualFPR_Sequential']) > 0 else 1e-6)
            target_eps.append(float(row['TargetEpsilon']))
    
    plt.figure(figsize=(10, 6))
    
    plt.plot(bpk, fpr_unif, marker='o', label='Uniform Data (Random Queries)')
    plt.plot(bpk, fpr_seq, marker='s', label='Sequential Data (Gap Queries)')
    plt.plot(bpk, target_eps, linestyle='--', color='red', label='Target Epsilon (Theory)')
    
    plt.yscale('log')
    plt.xlabel('Bits per Key (bpk)')
    plt.ylabel('False Positive Rate (FPR)')
    plt.title('Optimized Adaptive ARE: FPR vs Space Tradeoff')
    plt.grid(True, which="both", ls="-", alpha=0.5)
    plt.legend()
    
    output_dir = "bench_results/plots"
    os.makedirs(output_dir, exist_ok=True)
    plt.savefig(os.path.join(output_dir, "are_optimized_tradeoff.png"))
    print(f"Plot saved to {output_dir}/are_optimized_tradeoff.png")

if __name__ == "__main__":
    main()
