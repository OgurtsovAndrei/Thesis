import os
import statistics
from collections import defaultdict
from typing import Any, Dict, List, Optional, Union

def parse_key_params(name: str) -> Dict[str, Any]:
    # Parses standard param format: "BenchmarkName/KeySize=64/Keys=1024-8"
    row: Dict[str, Any] = {}
    if "/" not in name:
        return row
        
    params_part = name.split("/", 1)[1]
    # Remove GOMAXPROCS suffix if present (e.g., "-8")
    if "-" in params_part and params_part.rsplit("-", 1)[1].isdigit():
        params_part = params_part.rsplit("-", 1)[0]
        
    for chunk in params_part.split("/"):
        if "=" in chunk:
            k, v = chunk.split("=", 1)
            try:
                row[k.lower()] = float(v)
            except ValueError:
                row[k.lower()] = v
    return row

def parse_file(path: str) -> List[Dict[str, Any]]:
    rows: List[Dict[str, Any]] = []
    if not os.path.exists(path):
        return rows
        
    with open(path, "r") as f:
        for line in f:
            if not line.startswith("Benchmark"):
                continue
            parts = line.split()
            if len(parts) < 4:
                continue
                
            full_name = parts[0]
            # Standard Go fields
            row: Dict[str, Any] = {
                "full_name": full_name,
                "benchmark": full_name.split("/")[0],
            }
            # Auto-extract parameters from name
            row.update(parse_key_params(full_name))
            
            # Helper to parse Value Unit pairs
            def try_float(s: str) -> Optional[float]:
                try: return float(s)
                except ValueError: return None

            # Iterate tokens to find standard and custom metrics
            # Standard: 100 234 ns/op
            try:
                samples = int(parts[1])
                row["samples_count"] = samples
            except ValueError:
                pass

            for i, tok in enumerate(parts):
                if i < 2: continue
                
                prev = parts[i-1]
                val = try_float(prev)
                
                if tok == "ns/op":
                    if val is not None: row["ns_per_op"] = val
                elif tok == "B/op":
                    if val is not None: row["bytes_per_op"] = val
                elif tok == "allocs/op":
                    if val is not None: row["allocs_per_op"] = val
                elif val is not None:
                    # Heuristic for custom metrics: "88.0 bits/key"
                    # Only accept if the 'unit' is NOT a number
                    if try_float(tok) is not None:
                        continue
                        
                    clean_unit = tok.replace("/", "_").replace(".", "")
                    row[clean_unit] = val
            
            rows.append(row)
    return rows

def aggregate(rows: List[Dict[str, Any]], group_keys: List[str]) -> List[Dict[str, Any]]:
    # Group by specified keys (e.g., ["benchmark", "keysize", "keys"])
    grouped: Dict[tuple[Any, ...], List[Dict[str, Any]]] = defaultdict(list)
    for r in rows:
        # Create a signature for grouping
        key_vals = []
        for k in group_keys:
            key_vals.append(r.get(k))
        grouped[tuple(key_vals)].append(r)
        
    agg_rows: List[Dict[str, Any]] = []
    for g_key, group in grouped.items():
        res: Dict[str, Any] = {}
        # Fill grouping keys
        for i, k in enumerate(group_keys):
            res[k] = g_key[i]
            
        res["samples"] = len(group)
        
        # Calculate medians for all numeric fields present in the group
        all_metric_keys = set().union(*(d.keys() for d in group))
        for k in all_metric_keys:
            if k in group_keys or k in ["full_name", "samples", "samples_count"]: 
                continue
                
            vals = [r[k] for r in group if k in r and isinstance(r[k], (int, float))]
            if vals:
                res[k] = statistics.median(vals)
        
        agg_rows.append(res)
    return agg_rows
