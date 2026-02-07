# PSig / Memory Study Summary

## Inputs
- `mmph/paramselect/study/data/grid_main_v2.csv`
- `mmph/paramselect/study/data/grid_focus_v2.csv`
- `mmph/paramselect/study/data/grid_focus_extra_s16.csv`
- `mmph/paramselect/study/data/grid_focus_extra_s8.csv`
- `mmph/paramselect/study/data/grid_focus_extra_s32.csv`
- `mmph/paramselect/study/data/grid_focus_big_s32.csv`
- `mmph/paramselect/study/data/grid_focus_small_s8.csv`
- `/Users/andrei.ogurtsov/Thesis/mmph/paramselect/study/memory_bench_v2.txt`

## Main Findings
- On `grid_main_v2` (144 scenarios, 64 trials each): mean success by S is S=8: 0.166, S=16: 0.893, S=32: 1.000.
- `S=32` is fully stable in this grid (all scenarios succeeded in all trials); S=8 fails for most medium/large settings.
- `S=16` is mixed: stable up to moderate `n`, but for `n=131072` several `w` values show severe degradation.
- This confirms that theorem-based `S` from per-query bound (`epsilon_query = m/n`) is necessary but not sufficient for high probability of full-structure build success.

## Margin Analysis
- `summary_by_margin.csv` shows behavior grouped by `s_margin_bits = S - S_required`.
- Positive margin improves success rate but does not guarantee `~1.0` success for largest `n`.

## Memory Snapshot (from `/Users/andrei.ogurtsov/Thesis/mmph/paramselect/study/memory_bench_v2.txt`)
- RLOC bits/key: min=51.27, median=56.04, max=106.00.
- LERLOC bits/key: min=115.30, median=212.40, max=320.00.
- Stable regime (`keys>=8192`): RLOC avg=54.33 bits/key, LERLOC avg=182.43 bits/key.
- MMPH baseline from paper chart: ~14 bits/key (for large n).

## Generated Artifacts
- `mmph/paramselect/study/data/summary_by_s.csv`
- `mmph/paramselect/study/data/summary_by_margin.csv`
- `mmph/paramselect/study/data/worst_cases.csv`
- `mmph/paramselect/study/data/memory_points.csv`
- `mmph/paramselect/study/plots/success_rate_by_s.svg`
- `mmph/paramselect/study/plots/s16_success_vs_n.svg`
- `mmph/paramselect/study/plots/s8_success_vs_n.svg`
- `mmph/paramselect/study/plots/s32_success_vs_n.svg`
- `mmph/paramselect/study/plots/grid_size_report_bpk_vs_n.svg`
- `mmph/paramselect/study/plots/memory_bits_per_key_keys32768.svg`
