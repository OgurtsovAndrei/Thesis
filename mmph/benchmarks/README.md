# MMPH Benchmarks

This folder contains a small pipeline to collect Go benchmark outputs, parse them, and generate SVG plots for cross-module
comparison.

## How to run

Single command (runs benchmarks, parses, and generates plots):

```bash
python3 /Users/andrei.ogurtsov/Thesis/mmph/benchmarks/analyze.py --run
```

If you already have raw outputs in `benchmarks/raw/`, you can skip running benches:

```bash
python3 /Users/andrei.ogurtsov/Thesis/mmph/benchmarks/analyze.py
```

## Outputs

- Raw benchmark logs:
  - `raw/bucket_mmph.txt`
  - `raw/rbtz_mmph.txt`
  - `raw/relative_trie.txt`
- Parsed CSVs:
  - `parsed/bench_long.csv`
  - `parsed/bench_agg.csv`
- Plots (SVG):
  - `plots/build_time_ns.svg`
  - `plots/lookup_time_ns.svg`
  - `plots/bits_per_key_in_mem.svg`
  - `plots/bytes_in_mem.svg`
  - `plots/allocs_per_op.svg`

## Metrics and normalization

- **Build time**: `ns/op` from build benchmarks only.
- **Lookup time**: `ns/op` from lookup benchmarks only.
- **Bits/key (in-mem)**: `bits/key_in_mem` where available (falls back to `bits_per_key`).
- **Bytes (in-mem)**: `bytes_in_mem`.
- **Allocs/op**: `allocs/op` from build benchmarks.

Aggregation uses the median across runs for each `(module, metric, keycount)`.

## Caveats

- Key generation differs by module (as-implemented). The pipeline does not enforce a shared keyset.
- Results are machine-dependent; use the same environment to compare changes over time.
