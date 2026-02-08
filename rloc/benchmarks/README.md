# RLOC Benchmarking

Automated performance testing for Range Locator (RLOC) and Learned Range Locator (LERLOC).

## Requirements

- Python 3+
- Go 1.20+
- `pipenv` (optional but recommended)

## Quick Start

Run full benchmarks (this may take time):

```bash
# From project root
pipenv run python3 rloc/benchmarks/analyze.py --run --count 5
```

## Options

- `--run`: Execute benchmarks before parsing. If omitted, parses existing files in `rloc/benchmarks/raw`.
- `--count N`: Run each benchmark N times (default 5).
- `-j N`, `--jobs N`: Number of parallel workers (default: all CPUs).
- `--bench REGEX`: Run specific benchmarks (e.g., `--bench=Build`).

## Output

Results are generated in `rloc/benchmarks/`:

- `raw/`: Raw output from `go test`.
- `parsed/`:
  - `all_runs.csv`: Full dataset.
  - `agg.csv`: Aggregated (median) results.
- `plots/`: SVG visualizations.
  - `build_time_ns.svg`: Build time vs N (Log-Log).
  - `bits_per_key.svg`: Memory usage vs N.
  - `comparison_bits_key_*.svg`: RLOC vs LERLOC memory comparison.

## Shared Library

The benchmarking logic (runner, parser, plotter) is located in `scripts/bench_lib/`. This library is shared across modules (`mmph` can be updated to use it).
