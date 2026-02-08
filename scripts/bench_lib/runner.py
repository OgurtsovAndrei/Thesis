import concurrent.futures
import os
import re
import subprocess
import sys

def ensure_dir(path):
    os.makedirs(path, exist_ok=True)

def list_benchmarks(cwd, pattern="."):
    """Lists all benchmarks in the given directory matching the pattern."""
    try:
        args = ["go", "test", "-list", pattern]
        result = subprocess.run(args, cwd=cwd, capture_output=True, text=True, check=True)
        benchmarks = []
        for line in result.stdout.splitlines():
            if line.startswith("Benchmark"):
                benchmarks.append(line.strip())
        return benchmarks
    except subprocess.CalledProcessError as e:
        print(f"Error listing benchmarks in {cwd}: {e}", file=sys.stderr)
        return []

def run_single_iteration(mod, iteration, bench, benchmem, bench_suffix=""):
    # If bench_suffix is provided (e.g. specific benchmark name), 
    # we use it for file naming and -bench flag.
    # Otherwise we use the provided bench pattern.
    actual_bench = f"^{bench}$" if bench_suffix else bench
    
    args = ["go", "test", f"-bench={actual_bench}", "-count=1"]
    if benchmem:
        args.append("-benchmem")
    args.append("./...")
    
    # Ensure raw directory exists
    raw_dir = os.path.dirname(mod["out"])
    ensure_dir(raw_dir)

    # Unique temp file name
    file_tag = f"{bench_suffix}.{iteration}" if bench_suffix else str(iteration)
    temp_out = f"{mod['out']}.{file_tag}"
    
    try:
        with open(temp_out, "w") as f:
            subprocess.run(args, cwd=mod["dir"], stdout=f, stderr=subprocess.STDOUT, check=True)
        return (mod["name"], iteration, temp_out, True)
    except subprocess.CalledProcessError as e:
        print(f"Error running benchmark {mod['name']} ({actual_bench}) iter {iteration}: {e}", file=sys.stderr)
        return (mod["name"], iteration, temp_out, False)

def run_benchmarks(modules, count, bench, benchmem, jobs, split=False):
    if jobs is None:
        jobs = os.cpu_count() or 4
    
    print(f"Running benchmarks with {jobs} workers (split={split})...")
    
    tasks = []
    temp_files_map = {mod["name"]: [] for mod in modules}
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as executor:
        for mod in modules:
            if split:
                # Find all individual benchmarks
                all_benchmarks = list_benchmarks(mod["dir"], pattern=bench)
                if not all_benchmarks:
                    print(f"No benchmarks found in {mod['dir']} matching {bench}")
                    continue
                
                print(f"Module {mod['name']}: splitting into {len(all_benchmarks)} benchmarks * {count} iterations")
                for b_name in all_benchmarks:
                    for i in range(count):
                        tasks.append(executor.submit(run_single_iteration, mod, i, b_name, benchmem, bench_suffix=b_name))
                        # Keep track of temp file for merging
                        temp_files_map[mod["name"]].append(f"{mod['out']}.{b_name}.{i}")
            else:
                for i in range(count):
                    tasks.append(executor.submit(run_single_iteration, mod, i, bench, benchmem))
                    temp_files_map[mod["name"]].append(f"{mod['out']}.{i}")
        
        concurrent.futures.wait(tasks)
        
    print("Merging outputs...")
    for mod in modules:
        with open(mod["out"], "w") as outfile:
            for temp in temp_files_map[mod["name"]]:
                if os.path.exists(temp):
                    with open(temp, "r") as infile:
                        outfile.write(infile.read())
                    os.remove(temp)
    print("Done.")
