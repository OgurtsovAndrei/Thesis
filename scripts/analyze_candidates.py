import sys
from collections import defaultdict

def analyze(logfile):
    stats = defaultdict(lambda: None)
    total_counts = None
    
    try:
        with open(logfile, 'r') as f:
            for line in f:
                parts = line.strip().split(',')
                if len(parts) < 2:
                    continue
                test_name = parts[0]
                counts = [int(x) for x in parts[1:]]
                n = len(counts)
                
                if stats[test_name] is None:
                    stats[test_name] = [0] * n
                for i in range(min(n, len(stats[test_name]))):
                    stats[test_name][i] += counts[i]
                
                if total_counts is None:
                    total_counts = [0] * n
                for i in range(min(n, len(total_counts))):
                    total_counts[i] += counts[i]
    except FileNotFoundError:
        print(f"File {logfile} not found")
        return

    num_cand = len(total_counts) if total_counts else 0
    header = " | ".join(f"C{i+1:<7}" for i in range(num_cand))
    print(f"{'Test Name':<40} | {header}")
    print("-" * (43 + 10 * num_cand))
    for test_name, counts in stats.items():
        row = " | ".join(f"{c:<8}" for c in counts)
        print(f"{test_name[:40]:<40} | {row}")
    print("-" * (43 + 10 * num_cand))
    if total_counts:
        row = " | ".join(f"{c:<8}" for c in total_counts)
        print(f"{'TOTAL':<40} | {row}")

if __name__ == "__main__":
    analyze(sys.argv[1] if len(sys.argv) > 1 else "candidate_stats.log")
