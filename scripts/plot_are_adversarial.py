#!/usr/bin/env python3
"""Plot ARE adversarial tradeoff: FPR vs BPK for multiple scenarios."""

import math
import sys
import os

sys.path.append(os.path.join(os.path.dirname(__file__), ".."))
sys.path.append(os.path.join(os.getcwd(), "scripts"))
from bench_lib.plotter import draw_line_chart, ensure_dir

CSV_PATH = "bench_results/range_emptiness/adversarial/data.csv"
OUT_DIR = "bench_results/range_emptiness/adversarial"

SCENARIO_LABELS = {
    "uniform_point":    "Uniform Keys + Point Queries",
    "uniform_wide":     "Uniform Keys + Wide Range (w≤10K)",
    "spread_point":     "Spread Keys + Point Queries",
    "spread_gap":       "Spread Keys + Gap Queries",
    "clustered_point":  "Clustered Keys + Point Queries",
}

def load_csv(path):
    rows = []
    with open(path) as f:
        header = None
        for line in f:
            parts = line.strip().split(",")
            if parts[0] == "Scenario":
                header = parts
                continue
            if header and len(parts) == len(header):
                rows.append(dict(zip(header, parts)))
    return rows


def add_theoretical_bound(series):
    """Add the theoretical bound: FPR = 2*N / 2^K, BPK = K * N_unique/N ≈ K."""
    N = 200000
    pts = []
    for K in range(18, 29):
        fpr = 2.0 * N / (2 ** K)
        bpk = float(K)  # worst case: all prefixes distinct
        if fpr > 0:
            pts.append((bpk, fpr))
    series["Theoretical Bound (2N/2^K)"] = pts


def main():
    if not os.path.exists(CSV_PATH):
        print(f"CSV not found: {CSV_PATH}")
        return

    rows = load_csv(CSV_PATH)

    # --- Plot 1: FPR vs BPK (all scenarios, log Y) ---
    series = {}
    for row in rows:
        scenario = row["Scenario"]
        bpk = float(row["BitsPerKey"])
        fpr = float(row["ActualFPR"])
        if fpr <= 0:
            fpr = 1e-8
        label = SCENARIO_LABELS.get(scenario, scenario)
        series.setdefault(label, []).append((bpk, fpr))

    add_theoretical_bound(series)

    for name in series:
        series[name].sort()

    ensure_dir(OUT_DIR)
    draw_line_chart(
        path=os.path.join(OUT_DIR, "adversarial_fpr_vs_bpk.svg"),
        title="ARE: FPR vs Space Under Different Workloads (N=200K)",
        x_label="Bits per Key",
        y_label="False Positive Rate",
        series=series,
        log_y=True,
        styles={"Theoretical Bound (2N/2^K)": "dashed"},
    )
    print(f"Plot 1 saved: {OUT_DIR}/adversarial_fpr_vs_bpk.svg")

    # --- Plot 2: Only range/gap queries (zoomed, practical region) ---
    range_series = {}
    for row in rows:
        scenario = row["Scenario"]
        if scenario not in ("uniform_wide", "spread_gap", "uniform_point"):
            continue
        bpk = float(row["BitsPerKey"])
        fpr = float(row["ActualFPR"])
        if fpr <= 0:
            fpr = 1e-8
        label = SCENARIO_LABELS.get(scenario, scenario)
        range_series.setdefault(label, []).append((bpk, fpr))

    add_theoretical_bound(range_series)

    for name in range_series:
        range_series[name].sort()

    draw_line_chart(
        path=os.path.join(OUT_DIR, "adversarial_range_queries.svg"),
        title="ARE: Range Query FPR Under Adversarial Conditions (N=200K)",
        x_label="Bits per Key",
        y_label="False Positive Rate",
        series=range_series,
        log_y=True,
        styles={"Theoretical Bound (2N/2^K)": "dashed"},
    )
    print(f"Plot 2 saved: {OUT_DIR}/adversarial_range_queries.svg")

    # --- Plot 3: Clustered vs Uniform (BPK comparison) ---
    bpk_series = {}
    for row in rows:
        scenario = row["Scenario"]
        if scenario not in ("uniform_point", "clustered_point", "spread_point"):
            continue
        k = int(row["K"])
        bpk = float(row["BitsPerKey"])
        label = SCENARIO_LABELS.get(scenario, scenario)
        bpk_series.setdefault(label, []).append((float(k), bpk))

    for name in bpk_series:
        bpk_series[name].sort()

    draw_line_chart(
        path=os.path.join(OUT_DIR, "adversarial_bpk_by_distribution.svg"),
        title="ARE: Space Consumption by Key Distribution (N=200K)",
        x_label="Prefix Length K (bits)",
        y_label="Bits per Key",
        series=bpk_series,
    )
    print(f"Plot 3 saved: {OUT_DIR}/adversarial_bpk_by_distribution.svg")


if __name__ == "__main__":
    main()
