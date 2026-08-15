#!/usr/bin/env python3
"""Measure runtime performance: tree-walker vs VM (A) and Pipe vs Python CPU microbenchmarks (C).

Usage: python tools/measure_perf.py
  No API key needed. Writes results/performance.json.
"""
import json
import statistics
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).parent.parent
PIPE_BIN = Path("/home/harry/CODE/pipe/bin/pipe")
PYTHON_BIN = ROOT / "python-vs-pipe/python/.venv/bin/python"
if not PYTHON_BIN.exists():
    PYTHON_BIN = Path(sys.executable)

PIPE_DIR = ROOT / "pipe"
PY_DIR = ROOT / "python"
RUNS = 5

PAIRS = [
    ("fib(20)", "fib20.pipe", "fib20.py"),
    ("loop sum 1M", "loop_sum.pipe", "loop_sum.py"),
    ("str concat 20k", "str_concat.pipe", "str_concat.py"),
    ("list push 20k", "list_push.pipe", "list_push.py"),
]


def median_times(cmd, cwd, runs=RUNS, timeout=120):
    """Run cmd `runs` times, return list of wall times in seconds."""
    times = []
    for _ in range(runs):
        t = time.monotonic()
        try:
            subprocess.run(cmd, cwd=str(cwd), capture_output=True, text=True, timeout=timeout, check=True)
            times.append(time.monotonic() - t)
        except (subprocess.TimeoutExpired, subprocess.CalledProcessError):
            return None
    return times


def main():
    results = {"a_vm_vs_tree": [], "c_pipe_vs_python": [], "sizes": {}}

    # --- A: VM vs Tree-Walker (internal bench, pure exec time) ---
    bench_out = subprocess.run([str(PIPE_BIN), "-bench"], capture_output=True, text=True).stdout
    a_rows = []
    current = None
    for line in bench_out.splitlines():
        s = line.strip()
        if not s or s.startswith("Pipe Benchmark") or s.startswith("-"):
            continue
        if s.endswith(":") and not s.startswith(("Tree-Walker", "VM")):
            current = {"name": s[:-1]}
            a_rows.append(current)
            continue
        if current is None:
            continue
        key, _, val = s.partition(":")
        key = key.strip()
        val = val.strip()
        if key == "Tree-Walker":
            current["tree"] = val
        elif key == "VM":
            current["vm"] = val
        elif key == "Speedup":
            current["speedup"] = float(val.replace("x", ""))
    results["a_vm_vs_tree"] = a_rows
    print("A: VM vs Tree-Walker")
    for r in a_rows:
        print(f"  {r['name']:22s} tree={r['tree']:>12s}  vm={r['vm']:>12s}  speedup={r.get('speedup')}x")

    # --- C: Pipe vs Python CPU microbenchmarks (wall time incl. startup, median) ---
    print("\nC: Pipe vs Python (median over %d runs, wall time incl. startup)" % RUNS)
    for name, pipe_rel, py_rel in PAIRS:
        pipe_times = median_times([str(PIPE_BIN), "-vm", "-q", pipe_rel], PIPE_DIR)
        py_times = median_times([str(PYTHON_BIN), py_rel], PY_DIR)
        entry = {"name": name}
        if pipe_times:
            entry["pipe_sec"] = round(statistics.median(pipe_times), 4)
        if py_times:
            entry["python_sec"] = round(statistics.median(py_times), 4)
        if pipe_times and py_times:
            if pipe_times >= py_times:
                entry["faster"] = "python"
                entry["faster_x"] = round(entry["pipe_sec"] / entry["python_sec"], 1)
            else:
                entry["faster"] = "pipe"
                entry["faster_x"] = round(entry["python_sec"] / entry["pipe_sec"], 1)
        results["c_pipe_vs_python"].append(entry)
        print(f"  {name:20s} pipe={entry.get('pipe_sec')}s  python={entry.get('python_sec')}s"
              f"  faster={entry.get('faster')} x{entry.get('faster_x')}")

    # --- Sizes ---
    results["sizes"] = {"pipe_binary_bytes": Path(PIPE_BIN).stat().st_size}
    results["sizes"]["pipe_binary_human"] = f"{results['sizes']['pipe_binary_bytes']/1e6:.1f} MB"
    venv = ROOT.parent / "python-vs-pipe/python/.venv"
    results["sizes"]["python_venv_bytes"] = sum(
        f.stat().st_size for f in venv.rglob("*") if f.is_file()
    )
    results["sizes"]["python_venv_human"] = f"{results['sizes']['python_venv_bytes']/1e6:.0f} MB"

    out = ROOT / "results/performance.json"
    out.write_text(json.dumps(results, indent=2, ensure_ascii=False))
    print(f"\nWrote {out}")


if __name__ == "__main__":
    main()
