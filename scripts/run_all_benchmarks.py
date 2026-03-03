#!/usr/bin/env python3
import concurrent.futures
import json
import os
import subprocess
import sys
import time
import queue
from pathlib import Path

# Добавляем путь к bench_lib
sys.path.append(str(Path(__file__).parent))
from bench_lib import parser

# Настройка пакетов для тестирования
TARGET_PACKAGES = [
    "locators/rloc",
    "locators/lerloc",
    "trie/zft",
    "trie/hzft",
    "trie/shzft",
    "trie/azft",
    "mmph/relative_trie",
]

RAW_DIR = Path("bench_results/raw")
SUMMARY_DIR = Path("bench_results/summary")

# Настройки производительности
MAX_WORKERS = 16
BENCH_TIME = "3s"
P_CORES = list(range(16)) # Ядра 0-7 для i9-13900 (P-cores)

# Очередь для управления ядрами
core_queue = queue.Queue()
for c in P_CORES:
    core_queue.put(c)

def ensure_dirs():
    RAW_DIR.mkdir(parents=True, exist_ok=True)
    SUMMARY_DIR.mkdir(parents=True, exist_ok=True)

def list_benchmarks(package_path):
    """Возвращает список имен бенчмарков в пакете."""
    try:
        cmd = ["go", "test", "-list", "Benchmark", f"./{package_path}"]
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        return [line.strip() for line in result.stdout.splitlines() if line.startswith("Benchmark")]
    except Exception as e:
        return []

def run_benchmark(package_path, bench_name):
    """Запускает один бенчмарк на конкретном ядре."""
    core_id = core_queue.get()
    
    safe_name = bench_name.replace("/", "_")
    output_file = RAW_DIR / f"{package_path.replace('/', '_')}_{safe_name}.txt"
    
    # Используем taskset для привязки к ядру
    cmd = [
        "taskset", "-c", str(core_id),
        "go", "test", 
        "-bench", f"^{bench_name}$",
        "-benchmem",
        f"-benchtime={BENCH_TIME}",
        "-run=^$",
        f"./{package_path}"
    ]
    
    try:
        with open(output_file, "w") as f:
            subprocess.run(cmd, stdout=f, stderr=subprocess.STDOUT, check=True)
        return True
    except Exception:
        return False
    finally:
        core_queue.put(core_id)

def main():
    ensure_dirs()
    
    all_tasks = []
    for pkg in TARGET_PACKAGES:
        benchmarks = list_benchmarks(pkg)
        for b in benchmarks:
            all_tasks.append((pkg, b))
    
    total = len(all_tasks)
    if total == 0:
        print("No benchmarks found.")
        return

    print(f"Starting {total} benchmarks on cores {P_CORES} (benchtime={BENCH_TIME})...")
    
    start_total = time.time()
    completed = 0
    success_count = 0
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        future_to_bench = {executor.submit(run_benchmark, pkg, b): (pkg, b) for pkg, b in all_tasks}
        
        for future in concurrent.futures.as_completed(future_to_bench):
            completed += 1
            if future.result():
                success_count += 1
            
            # Обновляем прогресс в стиле docker/tqdm
            elapsed = time.time() - start_total
            percent = (completed / total) * 100
            bar_len = 20
            filled_len = int(bar_len * completed // total)
            bar = '█' * filled_len + '-' * (bar_len - filled_len)
            
            sys.stdout.write(f"\r[{bar}] {completed}/{total} [{elapsed:.2f}s] {percent:.1f}% | Success: {success_count}")
            sys.stdout.flush()

    print("\n\nParsing results...")
    all_rows = []
    for raw_file in RAW_DIR.glob("*.txt"):
        all_rows.extend(parser.parse_file(str(raw_file)))
    
    if all_rows:
        import csv
        summary_csv = SUMMARY_DIR / "all_benchmarks.csv"
        summary_json = SUMMARY_DIR / "all_benchmarks.json"
        
        all_keys = set()
        for row in all_rows:
            all_keys.update(row.keys())
        fieldnames = sorted(list(all_keys))
        
        with open(summary_csv, "w", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(all_rows)
            
        with open(summary_json, "w") as f:
            json.dump(all_rows, f, indent=2)

    print(f"Done! Summary saved to {SUMMARY_DIR}")

if __name__ == "__main__":
    main()
