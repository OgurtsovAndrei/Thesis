# AI Agent Conventions & Project Standards

This document outlines the coding standards, benchmarking protocols, and workflow conventions established for this repository. Adhering to these rules ensures consistency and performance across modules.

## 1. Python Coding Standards

- **Strict Typing**: All Python code must be fully type-annotated. Use `mypy` for static analysis.
- **Mypy Configuration**: Always use `explicit_package_bases = true` in `pyproject.toml` to handle multiple files with the same name (e.g., `analyze.py` in different modules).
- **Dependency Management**: Use `Pipfile` and `pipenv` for all Python dependencies.
- **Shared Logic**: Do not duplicate logic for common tasks (running benchmarks, parsing, plotting). Use the centralized library in `scripts/bench_lib/`.
- **Formatting**: Use standard Python idioms. Prefer `pathlib` or `os.path` for robust cross-platform path handling.

## 2. Benchmarking Infrastructure

- **Infrastructure**: Use `scripts/bench_lib/runner.py` for executing Go benchmarks.
- **Max Parallelism**: Always utilize all available CPU cores. Use `concurrent.futures.ThreadPoolExecutor` with `os.cpu_count()`.
- **Granular Splitting**: Support a `--split` flag. When enabled, use `go test -list` to extract individual benchmark functions and run each (benchmark, iteration) pair as a separate process to maximize CPU utilization.
- **Parsing**: The parser in `scripts/bench_lib/parser.py` is designed to handle Go's `ReportMetric` output in the `value unit` format (e.g., `88.0 bits/key`).
- **Data Integrity**: Merge temporary iteration files (`*.txt.0`, `*.txt.1`, etc.) into a final report only after all tasks are completed. Handle potential timeouts gracefully.

## 3. Visualization Standards

- **Format**: Always generate plots in **SVG** format for infinite scalability and small file size.
- **Scalability Analysis**: Use logarithmic scales for the X-axis (key counts) and Y-axis (build time, allocations) to clearly show asymptotic behavior ($O(N)$ vs $O(\log N)$).
- **Styling**: Maintain a consistent color palette (defined in `plotter.py`). Include legends and descriptive labels.

## 4. Git & Commit Conventions

- **Incremental Commits**: Commit changes immediately after each successfully completed and verified logical step. This ensures a clean history and easier rollback if needed.
- **Atomic Commits**: Split infrastructure changes (e.g., `bench_lib` updates) from module-specific logic or data updates.
- **Commit Messages**: 
  - Format: `module: short description` (e.g., `rloc: add performance report`).
  - Use a detailed body with bullet points for complex changes.
- **Ignored Files**: Ensure `*.txt.*` (temporary benchmark files) and `.DS_Store` are always in `.gitignore`.

## 5. Documentation

- **Automated Reports**: Analysis scripts should ideally update or generate `README.md` files within their respective modules to provide immediate insights from benchmark data.
- **Architecture Context**: Document how low-level components (like MMPH or RLOC) fit into the overall project hierarchy (e.g., Range Filter architecture).

## 6. AI Workflow Tips

- **Plan Mode**: Before heavy implementation, use a "Planning Mode" to describe the architecture and get approval.
- **Safety**: Do not assume directory existence; use `ensure_dir` (available in `bench_lib`).
- **Resource Awareness**: This environment may run on high-performance hardware (e.g., Apple M4 Max or 32-core VMs). Designing for high concurrency is a priority.
