#!/usr/bin/env python3
"""
Versionize: berechnet einen Content-Hash ueber die Worldcraft-Frontend-Dateien
und schreibt ihn als `?v=<hash>` Cache-Busting-Parameter in `index.html` sowie
in die STATIC_ASSETS-Liste und den CACHE_NAME der `sw.js`.

Aufruf:  python3 versionize.py [--web WEB]
    --web   Pfad zum web/-Verzeichnis (Default: web/ relativ zum Skript).

Jede Aenderung an einer der gehashten Dateien ergibt einen neuen Hash, sodass
Clients die neue Version beim naechsten Laden automatisch holen.
"""

import argparse
import hashlib
import os
import re
import sys

# Dateien, deren Inhalt den Hash bestimmt. index.html/sw.js werden NICHT
# gehasht (sie tragen den Hash selbst), um Selbstreferenz zu vermeiden.
HASHED = [
    "js/i18n.js",
    "js/api.js",
    "js/ui.js",
    "js/minimap.js",
    "js/audio.js",
    "js/app.js",
    "css/style.css",
    "css/minimap.css",
    "manifest.json",
]


def content_hash(web):
    h = hashlib.sha256()
    for rel in HASHED:
        path = os.path.join(web, rel)
        try:
            with open(path, "rb") as f:
                h.update(rel.encode("utf-8"))
                h.update(b"\0")
                h.update(f.read())
        except FileNotFoundError:
            print(f"WARN: {rel} fehlt, uebersprungen", file=sys.stderr)
    # Kurzform: 10 Hex-Zeichen ist ausreichend eindeutig fuer Cache-Busting.
    return h.hexdigest()[:10]


def update_index_html(web, ver):
    path = os.path.join(web, "index.html")
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()

    new = re.sub(r'(href|src)="(/[^"?]+\.(?:css|js))(?:\?v=[^"]*)?"', lambda m: f'{m.group(1)}="{m.group(2)}?v={ver}"', content)
    with open(path, "w", encoding="utf-8") as f:
        f.write(new)
    return new != content


def update_sw_js(web, ver):
    path = os.path.join(web, "sw.js")
    with open(path, "r", encoding="utf-8") as f:
        original = f.read()
    content = original

    # CACHE_NAME: worldcraft-vX -> worldcraft-v<hash>
    content = re.sub(r'(CACHE_NAME\s*=\s*"worldcraft-v)[^"]*(")', rf'\g<1>{ver}\g<2>', content)

    # STATIC_ASSETS: alle /...js?v= und /...css?v= URLs auf neuen Hash setzen
    content = re.sub(r'("/(?:js|css)/[^"?]+)(?:\?v=[^"]*)?"', lambda m: f'{m.group(1)}?v={ver}"', content)

    with open(path, "w", encoding="utf-8") as f:
        f.write(content)
    return content != original


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--web", default=None)
    args = ap.parse_args()

    web = args.web
    if web is None:
        web = os.path.join(os.path.dirname(os.path.abspath(__file__)), "web")

    ver = content_hash(web)
    print(f"Version: {ver}")

    changed_idx = update_index_html(web, ver)
    changed_sw = update_sw_js(web, ver)

    # Verify index.html + sw.js consistently use the new hash
    idx = open(os.path.join(web, "index.html"), encoding="utf-8").read()
    sw = open(os.path.join(web, "sw.js"), encoding="utf-8").read()
    if f"v={ver}" not in idx or f"v={ver}" not in sw:
        print("FEHLER: Version nicht vollstaendig geschrieben", file=sys.stderr)
        sys.exit(1)

    print("index.html:", "aktualisiert" if changed_idx else "unveraendert")
    print("sw.js:     ", "aktualisiert" if changed_sw else "unveraendert")
    return 0


if __name__ == "__main__":
    sys.exit(main())
