#!/usr/bin/env python3
"""
tools/bump_cache.py — Worldcraft Frontend Build-Hash.

Ersetzt  %%CACHE_VERSION%%  in sw.js durch einen Inhalts-Hash aller Assets
und bringt die  ?v=<hash>  in index.html auf den gleichen Stand. So greifen
Service-Worker-Updates und Asset-Busting automatisch mit — kein manuelles
Versionierung mehr nötig.

Aufruf:  python3 tools/bump_cache.py        (im worldcraft/web-Verzeichnis)
"""
import hashlib
import os
import re
import sys

WEB = os.path.dirname(os.path.abspath(__file__))
WEB = os.path.join(WEB, "..") if os.path.basename(WEB) == "tools" else WEB

ASSET_DIRS = ["js", "css", "img", "audio"]
ASSET_EXTS = (".js", ".css", ".png", ".jpg", ".jpeg", ".svg", ".mp3", ".json")

def hash_of_assets():
    h = hashlib.md5()
    for d in ASSET_DIRS:
        p = os.path.join(WEB, d)
        if not os.path.isdir(p):
            continue
        for root, _dirs, files in os.walk(p):
            for name in sorted(files):
                if name.endswith(ASSET_EXTS):
                    with open(os.path.join(root, name), "rb") as f:
                        h.update(f.read())
    return h.hexdigest()[:10]

def main():
    ver = hash_of_assets()
    sw_path = os.path.join(WEB, "sw.js")
    with open(sw_path, "r", encoding="utf-8") as f:
        sw = f.read()
    sw = re.sub(r'%%CACHE_VERSION%%', ver, sw, count=1)
    with open(sw_path, "w", encoding="utf-8") as f:
        f.write(sw)

        idx_path = os.path.join(WEB, "index.html")
    with open(idx_path, "r", encoding="utf-8") as f:
        idx = f.read()
    idx = re.sub(r"\?v=[a-f0-9]+", "?v=" + ver, idx)
    with open(idx_path, "w", encoding="utf-8") as f:
        f.write(idx)


    print("CACHE_VERSION:", ver)
    return 0

if __name__ == "__main__":
    sys.exit(main())
