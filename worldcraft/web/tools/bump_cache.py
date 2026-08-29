#!/usr/bin/env python3
"""
tools/bump_cache.py — Worldcraft Frontend Build-Hash.

Ersetzt den Cache-Versions-Hash in sw.js durch einen Inhalts-Hash aller Assets
und bringt die  ?v=<hash>  in index.html auf den gleichen Stand. So greifen
Service-Worker-Updates und Asset-Busting automatisch mit — kein manuelles
Versionierung mehr nötig.

Idempotent: egal ob in sw.js noch der Platzhalter  %%CACHE_VERSION%%  oder
bereits ein Literal-Hash steht, der nächste Lauf ersetzt ihn immer wieder.
Auch die STATIC_ASSETS-Liste im Service Worker wird aus dem Dateisystem
regeneriert, damit neue JS/CSS/Img/Audio-Dateien nie vergessen werden.

Aufruf:  python3 tools/bump_cache.py        (im worldcraft/web-Verzeichnis)
"""
import hashlib
import os
import re
import sys

WEB = os.path.dirname(os.path.abspath(__file__))
WEB = os.path.join(WEB, "..") if os.path.basename(WEB) == "tools" else WEB

ASSET_DIRS = ["css", "js", "img", "audio"]
ASSET_EXTS = (".css", ".js", ".png", ".jpg", ".jpeg", ".svg", ".mp3", ".ogg", ".webp", ".wav")
FIXED_ASSETS = ["/", "/index.html", "/manifest.json"]

CACHE_VERSION_RE = re.compile(r'const CACHE_VERSION\s*=\s*"[a-f0-9]*";')
ASSETS_BLOCK_RE = re.compile(r"(const STATIC_ASSETS\s*=\s*\[)[^\]]*(\];)", re.DOTALL)


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


def quoted(s):
    return '"' + s.replace("\\", "\\\\").replace('"', '\\"') + '"'


def static_assets_block():
    entries = list(FIXED_ASSETS)
    for d in ASSET_DIRS:
        p = os.path.join(WEB, d)
        if not os.path.isdir(p):
            continue
        for root, _dirs, files in os.walk(p):
            for name in sorted(files):
                if name.endswith(ASSET_EXTS):
                    rel = os.path.relpath(os.path.join(root, name), WEB)
                    entries.append("/" + rel.replace(os.sep, "/"))
    body = "".join("\n  " + quoted(e) + "," for e in entries)
    return "const STATIC_ASSETS = [" + body + "\n];"


def main():
    ver = hash_of_assets()
    sw_path = os.path.join(WEB, "sw.js")
    with open(sw_path, "r", encoding="utf-8") as f:
        sw = f.read()
    new_ver_line = 'const CACHE_VERSION = "%s";' % ver
    if not CACHE_VERSION_RE.search(sw):
        print("E: sw.js enthält keine CACHE_VERSION-Zeile", file=sys.stderr)
        return 1
    sw = CACHE_VERSION_RE.sub(new_ver_line, sw, count=1)
    if not ASSETS_BLOCK_RE.search(sw):
        print("E: sw.js enthält keinen STATIC_ASSETS-Block", file=sys.stderr)
        return 1
    new_block = static_assets_block()
    sw = ASSETS_BLOCK_RE.sub(lambda m: m.group(1) + new_block[new_block.index("[") + 1:new_block.rindex("]")] + m.group(2), sw, count=1)
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