import concurrent.futures
import os
import subprocess
import sys

def ensure_dir(path):
    os.makedirs(path, exist_ok=True)

def run_single_iteration(mod, iteration, bench, benchmem):
    args = ["go", "test", f"-bench={bench}", "-count=1"]
    if benchmem:
        args.append("-benchmem")
    args.append("./...")
    
    # Ensure raw directory exists
    raw_dir = os.path.dirname(mod["out"])
    ensure_dir(raw_dir)

    temp_out = f"{mod['out']}.{iteration}"
    try:
        with open(temp_out, "w") as f:
            subprocess.run(args, cwd=mod["dir"], stdout=f, stderr=subprocess.STDOUT, check=True)
        return (mod["name"], iteration, temp_out, True)
    except subprocess.CalledProcessError as e:
        print(f"Error running benchmark {mod['name']} iter {iteration}: {e}", file=sys.stderr)
        return (mod["name"], iteration, temp_out, False)

def run_benchmarks(modules, count, bench, benchmem, jobs):
    if jobs is None:
        jobs = os.cpu_count() or 4
    
    print(f"Running benchmarks with {jobs} workers...")
    
    tasks = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as executor:
        for mod in modules:
            for i in range(count):
                tasks.append(executor.submit(run_single_iteration, mod, i, bench, benchmem))
        
        concurrent.futures.wait(tasks)
        
    print("Merging outputs...")
    for mod in modules:
        with open(mod["out"], "w") as outfile:
            for i in range(count):
                temp = f"{mod['out']}.{i}"
                if os.path.exists(temp):
                    with open(temp, "r") as infile:
                        outfile.write(infile.read())
                    os.remove(temp)
    print("Done.")
