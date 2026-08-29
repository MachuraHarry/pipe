# Red-Team-Sandbox-Audit — Runde 7

**Datum:** 2026-08-29
**Methode:** statische Egress-Inventur aller Datei-Schreib-Builtins + Verifikation gegen das CLI-Flag-Set, nach exakt der Fehlerklasse aus Runde 6
**Stand Commit:** nach `995bfd8`

---

## Ergebnis: `--sandbox` ohne Profil ließ 6 von 8 Datei-Schreib-Builtins durch — jetzt gegatet

Runde 7 hat die Runde-6-Methodik (jedes gefährliche Builtin: Profil-Pfad-Check
gegen CLI-Flag-Pfad-Check vergleichen) auf die Dimension **Datei-Schreibzugriff**
statt Netzwerk angewendet. Ergebnis: dieselbe Fehlerklasse trat erneut auf.
`pipe --sandbox <script>` (ohne `--sandbox-profile`) blockiert laut Hilfe
„dangerous builtins (exec, tcp, http, ai, fs-write)" — aber die meisten
Schreib-Builtins **ignorierten das Flag** und führten echte Schreibzugriffe
auf das Host-Dateisystem aus.

Wichtig zur Einordnung: Der **Profil-Pfad** (`strict`/`noexec`/`isolated`/custom
mit `fs: "none"` o.ä., abgedeckt durch
`pkg/object/sandbox_fs_inventory_test.go`) war nicht betroffen — er leitet
korrekt in den Temp-Jail des Profils um. Die **exec**-Domäne (`exec`,
`proc_start`) wurde ebenfalls geprüft und war bereits auf beiden Pfaden
korrekt gegatet. Die Lücke betraf ausschließlich den „`none`-Profil +
Sandbox-Flags"-Pfad für Datei-Schreibzugriffe — genau wie Runde 6 beim Netz.

---

## Die Lücke — der `if Name != "none"`-Pfad vergaß erneut die CLI-Flags

Das CLI-`--sandbox`-Flag setzt `Sandbox.Enabled = true` und lässt
`ActiveProfile` auf `"none"` (das „alles erlaubt"-Profil). Die korrekt
gegateten Builtins (`file_delete`, `remove_dir`) folgten diesem Muster:

```go
if ActiveProfile.Load().Name != "none" {
    // canonicalWrite via Profil
} else if Sandbox.Enabled && !Sandbox.AllowFS {
    return sandboxBlock("... (filesystem write)")
}
```

Der zweite Zweig (`else if Sandbox …`) **fehlte komplett** in sechs
Schreib-Builtins — sie prüften nur das Profil und ließen im `"none"`-Pfad
jeden Schreibzugriff durch:

- `write_file` (`pkg/object/builtins_io.go`) — die zentrale Schreib-Primitive
- `append_file` (`pkg/object/builtins_fs.go`)
- `file_move` (`pkg/object/builtins_fs.go`) — nur Zielpfad; der Quell-Read war bereits korrekt
- `file_copy` (`pkg/object/builtins_fs.go`) — nur Zielpfad; der Quell-Read war bereits korrekt
- `make_dir` (`pkg/object/builtins_fs.go`)
- `file_open` in den schreibfähigen Modi `w`/`a`/`rw`/`rw+` (`pkg/object/file_io.go`) — Lese-Modus (`r`) war by design nicht betroffen

Nur `file_delete` und `remove_dir` hatten den CLI-Flag-Zweig bereits. Ein
red-teamed Skript hätte unter `pipe --sandbox` also überall dort, wo die
OS-Rechte es zuließen, Dateien schreiben, anhängen, verschieben, kopieren
oder Verzeichnisse anlegen können — sowie ein Datei-Handle für beliebige
Schreibzugriffe öffnen können — exakt die Schreib-Oberfläche, die der
Sandbox-Modus verspricht zu unterbinden.

---

## Fix (Commit `c928178`)

- **Zentraler Helper** `checkFSWriteAccess(feature, path)` in
  `pkg/object/builtins_sandbox.go`: kapselt beide Zweige (Profil-
  `canonicalWrite` **und** `Sandbox.Enabled && !Sandbox.AllowFS`) an einer
  Stelle, analog zu `checkNetworkAccess` aus Runde 6.
- Alle acht Schreib-Builtins nutzen ihn jetzt: `write_file`, `append_file`,
  `file_delete`, `file_move` (Ziel), `file_copy` (Ziel), `make_dir`,
  `remove_dir`, `file_open` (schreibfähige Modi). Die reinen Lese-Builtins
  (`read_file`, `read_lines`, `file_exists`, `file_size`, `file_type`,
  `list_dir`, `file_open` Modus `r`) bleiben unverändert — by design
  beschränkt das CLI-Flag nur Schreibzugriffe, keine Lesezugriffe.
- Die Fehlerklasse „vergessener `AllowFS`-Branch" ist für
  Datei-Schreibzugriffe jetzt strukturell nicht mehr reproduzierbar: Ein
  künftiges Builtin mit Schreibzugriff muss nur `checkFSWriteAccess`
  aufrufen.

### Regressionstests (`pkg/object/fs_write_gate_test.go`)

| Test | prüft |
|------|-------|
| `TestFSWriteBuiltinsBlockedBySandboxFlag` | alle 9 Schreib-Aufrufvarianten (8 Builtins, `file_open` doppelt für `w`/`a`) blockiert unter `Sandbox.Enabled=true, AllowFS=false`, Temp-Verzeichnis bleibt danach leer (kein Datei-I/O bei einem geblockten Call) |
| `TestFSWriteBuiltinsAllowedBySandboxFlag` | `write_file` funktioniert und schreibt den erwarteten Inhalt unter `AllowFS=true` |
| `TestFileOpenReadModeUnaffectedBySandboxFlag` | `file_open` im Modus `"r"` bleibt vom CLI-Flag ungegatet, analog zu `read_file`/`read_lines`/`list_dir` |

---

## Live-Verifikation (CLI, kein privilegiertes Dateisystem nötig)

```text
$ printf 'write_file "escape.txt" "exfil"' > wf.pipe

$ timeout 20 pipe --sandbox wf.pipe
Runtime error: wf.pipe: SANDBOX: write_file (filesystem write) is disabled in sandbox mode
  in fn(write_file)
exit=1
$ ls escape.txt
ls: cannot access 'escape.txt': No such file or directory      # Block vor jedem Datei-I/O ✓

$ timeout 20 pipe wf.pipe                # Kontrolle ohne --sandbox:
exit=0
$ ls -la escape.txt
-rw-r--r-- 1 root root 5 ... escape.txt                         # Schreibzugriff landet auf Platte → Block ist sandbox-spezifisch ✓
```

Der Block passiert **vor** jedem Datei-I/O (kein `os.WriteFile`-Aufruf), wie
die Kontrollgruppe belegt, die die Datei tatsächlich anlegt.

---

## Einordnung (ehrlich)

- Kein bekanntes Escape im **AI-Egress** (zentraler Gate, Runde 5), im
  **Netz-Gate** (zentraler Helper, Runde 6) oder im **Profil-Pfad** (Ratschen,
  Runde 2; Schreib-Redirect-Inventur, `sandbox_fs_inventory_test.go`).
- **exec-Domäne geprüft und sauber**: `exec` und `proc_start` hatten den
  CLI-Flag-Zweig bereits auf beiden Pfaden — kein Fix nötig.
- Gefunden und gefixt: Der `--sandbox`-Flag-Pfad ließ 6 von 8
  Schreib-Builtins durch, allen voran `write_file`. Beide Richtungen
  (blockiert und erlaubt) sowie die Lese/Schreib-Grenze sind durch Tests
  abgesichert.
- Verbleibende bekannte Limitationen: unverändert aus den Runden 4/5/6
  (globales `ActiveProfile`; Layer-2 ohne Redirect-Re-Check — hier n/a, da
  diese Runde reines Dateisystem betrifft).
- Methodik-Hinweis: Dieser Fund entstand durch systematisches erneutes
  Anwenden der Runde-6-Audit-Methode auf eine andere Fähigkeits-Dimension,
  nicht durch Red-Team-LLM-Interaktion. Das spricht dafür, künftige Runden
  jede verbleibende gefährliche Dimension (z. B. `ai`-Builtins jenseits des
  zentralen Egress-Gates) auf dieselbe Weise durchzukämmen, statt
  anzunehmen, ein Fix in einer Dimension hätte das Muster überall behoben.
