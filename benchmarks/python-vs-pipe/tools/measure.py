#!/usr/bin/env python3
"""Measure LOC (non-blank, non-comment) for all benchmark pairs and write JSON.

Usage: python tools/measure.py [--run]
  Without --run: only counts LOC.
  With --run: also executes each pair (needs DEEPSEEK_API_KEY) and times it.
"""
import json
import os
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).parent.parent
PIPE_BIN = Path("/home/harry/CODE/pipe/bin/pipe")
PYTHON_BIN = ROOT / "python/.venv/bin/python"

PAIRS = [
    ("rag", "pipe/rag.pipe", "python/rag.py"),
    ("parallel", "pipe/parallel.pipe", "python/parallel.py"),
    ("agent", "pipe/agent.pipe", "python/agent.py"),
    ("summarize_api", "pipe/summarize_api.pipe", "python/summarize_api.py"),
    ("log_analysis", "pipe/log_analysis.pipe", "python/log_analysis.py"),
    ("batch", "pipe/batch.pipe", "python/batch.py"),
]


def count_loc(path: Path) -> int:
    """Count non-blank, non-comment lines."""
    loc = 0
    text = path.read_text(errors="replace")
    for line in text.splitlines():
        s = line.strip()
        if not s:
            continue
        if s.startswith("#"):  # Python comment or shebang
            continue
        if s.startswith("--") or s.startswith("//"):  # Pipe comment
            continue
        loc += 1
    return loc


def run_cmd(cmd, cwd, timeout=180):
    t = time.monotonic()
    try:
        proc = subprocess.run(
            cmd, cwd=str(cwd), capture_output=True, text=True, timeout=timeout
        )
        ok = proc.returncode == 0
        return round(time.monotonic() - t, 2), ok, (proc.stderr or proc.stdout)[-300:], proc.stdout
    except subprocess.TimeoutExpired:
        return round(time.monotonic() - t, 2), False, "timeout", ""


SERVER_CMDS = {
    "pipe": [str(PIPE_BIN), "-vm", "-q", "pipe/summarize_api.pipe"],
    "python": [str(PYTHON_BIN), "python/summarize_api.py"],
}
SERVER_PORTS = {"pipe": 8787, "python": 8788}
SERVER_PAYLOAD = '{"text":"Caching improves performance by storing frequently accessed data. Our system uses Redis with a 5 minute TTL for config values."}'


def bench_server(which, cwd, url="/summarize", timeout=60):
    """Start a server in background, fire one request, measure, kill."""
    t = time.monotonic()
    proc = subprocess.Popen(
        SERVER_CMDS[which], cwd=str(cwd), stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )
    port = SERVER_PORTS[which]
    try:
        import urllib.request

        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            try:
                req = urllib.request.Request(
                    f"http://127.0.0.1:{port}" + url,
                    data=SERVER_PAYLOAD.encode(),
                    headers={"Content-Type": "application/json"},
                )
                with urllib.request.urlopen(req, timeout=10) as resp:
                    resp.read()
                return round(time.monotonic() - t, 2), True, ""
            except Exception:
                time.sleep(0.5)
        return round(time.monotonic() - t, 2), False, "server never answered"
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()


def main():
    do_run = "--run" in sys.argv
    results = {"pairs": [], "sizes": {}}

    for name, pipe_rel, py_rel in PAIRS:
        pipe_path = ROOT / pipe_rel
        py_path = ROOT / py_rel
        entry = {
            "name": name,
            "pipe_loc": count_loc(pipe_path),
            "python_loc": count_loc(py_path),
        }
        if do_run:
            if name == "summarize_api":
                # Server-based benchmark: run each server, hit it via curl, kill it
                sp, sok, serr = bench_server("pipe", ROOT)
                print(f"{name:15s} pipe={sp}s ok={sok}")
                entry["pipe_sec"] = sp
                entry["pipe_ok"] = sok
                if not sok:
                    entry["pipe_err"] = serr
                # Separate ports to avoid conflicts; re-use same script (FastAPI runs on 8787 too)
                # Quick re-check that python server also responds (same helper, sequential)
                sp2, sok2, serr2 = bench_server("python", ROOT)
                print(f"{name:15s} python={sp2}s ok={sok2}")
                entry["python_sec"] = sp2
                entry["python_ok"] = sok2
                if not sok2:
                    entry["python_err"] = serr2
            else:
                rp, pok, perr, pout = run_cmd([str(PIPE_BIN), "-vm", "-q", pipe_rel], ROOT)
                rp2, pok2, perr2, pout2 = run_cmd([str(PYTHON_BIN), py_rel], ROOT)
                entry["pipe_sec"] = rp
                entry["python_sec"] = rp2
                entry["pipe_ok"] = pok
                entry["python_ok"] = pok2
                if not pok:
                    entry["pipe_err"] = perr
                if not pok2:
                    entry["python_err"] = perr2
                if name == "parallel":
                    # Capture the internal sequential/parallel split measured inside
                    # each script (pure LLM call time, excluding interpreter startup).
                    import re
                    m = re.search(r"Sequenziell: ([\d.]+)s \| Parallel: ([\d.]+)s", pout2 or "")
                    if m:
                        entry["python_seq_sec"] = float(m.group(1))
                        entry["python_par_sec"] = float(m.group(2))
                    m2 = re.search(r"Fertig nach ([\d.]+)s", pout or "")
                    if m2:
                        entry["pipe_par_sec"] = float(m2.group(1))
                print(f"{name:15s} pipe={rp}s ok={pok}  python={rp2}s ok={pok2}")
        results["pairs"].append(entry)

    # Size comparison
    pipe_size = os.path.getsize(PIPE_BIN)
    venv_size = 0
    for root, dirs, files in os.walk(ROOT / "python/.venv"):
        for f in files:
            venv_size += os.path.getsize(os.path.join(root, f))
    results["sizes"] = {
        "pipe_binary_bytes": pipe_size,
        "pipe_binary_human": f"{pipe_size/1e6:.1f} MB",
        "python_venv_bytes": venv_size,
        "python_venv_human": f"{venv_size/1e6:.0f} MB",
    }

    out = ROOT / "results/results.json"
    out.write_text(json.dumps(results, indent=2, ensure_ascii=False))
    print(f"\nWrote {out}")
    print(json.dumps(results["sizes"], indent=2))
    for e in results["pairs"]:
        print(f"  {e['name']:15s} pipe={e['pipe_loc']} LOC  python={e['python_loc']} LOC")


if __name__ == "__main__":
    main()
