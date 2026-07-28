# 10. Builtin-Referenz

Pipe hat **80+ eingebaute Funktionen** — keine externen Abhängigkeiten
(alle in Go implementiert, nutzen nur die Standardbibliothek).

Die Builtins sind in **allen Ausführungsmodi** verfügbar (Tree-Walker und VM).

---

## 10.1 I/O und System

### print
```
print werte...
```
Gibt einen oder mehrere Werte aus, getrennt durch Leerzeichen, mit abschließender Newline.
```pipe
print "Hallo"            -- "Hallo"
print "Wert:" 42         -- "Wert: 42"
print (1 + 2)            -- "3"
```

### input
```
input prompt?
```
Liest eine Zeile von stdin. Optionaler Prompt wird zuerst ausgegeben.
```pipe
name: input "Name: "
print ("Hallo " ++ name)
```

### exec
```
exec kommando
```
Führt einen Shell-Befehl aus und gibt eine Map `{output, error, status}` zurück.
```pipe
ergebnis: exec "ls -la"
print (get ergebnis "output")
print (get ergebnis "status")     -- Exit-Code
```

### env
```
env name
```
Liest eine Umgebungsvariable. Gibt nil zurück, wenn sie nicht existiert.
```pipe
print (env "HOME")         -- "/home/user"
print (env "PATH")         -- "/usr/bin:..."
```

### sleep
```
sleep ms
```
Wartet für die angegebene Anzahl Millisekunden.
```pipe
sleep 1000                -- 1 Sekunde warten
print "Fertig"
```

---

## 10.2 Dateisystem

### read_file
```
read_file pfad
```
Liest den gesamten Inhalt einer Datei als String.
```pipe
inhalt: read_file "config.txt"
print inhalt
```

### write_file
```
write_file pfad inhalt
```
Schreibt einen String in eine Datei (überschreibt existierende Datei).
```pipe
write_file "ausgabe.txt" "Hallo Welt"
```

### append_file
```
append_file pfad inhalt
```
Hängt einen String an eine Datei an (erstellt sie, falls nicht vorhanden).
```pipe
append_file "log.txt" "Neuer Eintrag\n"
```

### read_lines
```
read_lines pfad
```
Liest eine Datei und gibt eine **Liste von Zeilen** zurück (ohne Newline-Zeichen).
```pipe
zeilen: read_lines "daten.csv"
print (len zeilen)            -- Anzahl Zeilen
print (at zeilen 0)           -- Erste Zeile
```

### file_exists
```
file_exists pfad
```
Prüft, ob eine Datei oder ein Verzeichnis existiert. Gibt `true`/`false` zurück.
```pipe
print (file_exists "/etc/hosts")     -- true
```

### file_delete
```
file_delete pfad
```
Löscht eine Datei.
```pipe
file_delete "/tmp/temp.txt"
```

### file_move
```
file_move quelle ziel
```
Benennt eine Datei um oder verschiebt sie.
```pipe
file_move "alt.txt" "neu.txt"
```

### file_copy
```
file_copy quelle ziel
```
Kopiert eine Datei.
```pipe
file_copy "original.txt" "kopie.txt"
```

### file_size
```
file_size pfad
```
Gibt die Dateigröße in **Bytes** zurück.
```pipe
print (file_size "daten.txt")     -- 1234
```

### file_type
```
file_type pfad
```
Gibt `"file"` für Dateien oder `"dir"` für Verzeichnisse zurück.
```pipe
print (file_type "/tmp")              -- "dir"
print (file_type "/tmp/test.txt")     -- "file"
```

### list_dir
```
list_dir pfad?
```
Listet den Inhalt eines Verzeichnisses auf. Verzeichnisse werden mit `/` am Ende markiert.
Ohne Argument wird das aktuelle Verzeichnis gelistet.
```pipe
dateien: list_dir "."
print dateien    -- ["main.pipe", "lib.pipe", "daten/"]
```

### make_dir
```
make_dir pfad
```
Erstellt ein Verzeichnis (rekursiv, inklusive Elternverzeichnisse).
```pipe
make_dir "/tmp/mein_projekt/daten"
```

### remove_dir
```
remove_dir pfad
```
Löscht ein Verzeichnis **rekursiv** (inklusive aller Inhalte). Vorsichtig verwenden!
```pipe
remove_dir "/tmp/mein_projekt"
```

### path_join
```
path_join a b...
```
Fügt Pfad-Komponenten mit dem passenden Trennzeichen zusammen.
```pipe
print (path_join "/home" "user" "docs")     -- "/home/user/docs"
```

### path_base
```
path_base pfad
```
Extrahiert den Dateinamen aus einem Pfad.
```pipe
print (path_base "/a/b/c.txt")     -- "c.txt"
```

### path_dir
```
path_dir pfad
```
Extrahiert das Verzeichnis aus einem Pfad.
```pipe
print (path_dir "/a/b/c.txt")     -- "/a/b"
```

### path_ext
```
path_ext pfad
```
Extrahiert die Dateiendung aus einem Pfad.
```pipe
print (path_ext "/a/b/c.txt")     -- ".txt"
print (path_ext "/a/b/file")      -- ""
```

---

## 10.3 String-Operationen

### upper
```
upper s
```
Konvertiert einen String in Großbuchstaben.
```pipe
print (upper "hallo")     -- "HALLO"
```

### lower
```
lower s
```
Konvertiert einen String in Kleinbuchstaben.
```pipe
print (lower "HALLO")     -- "hallo"
```

### trim
```
trim s
```
Entfernt Leerzeichen (Whitespace) am Anfang und Ende.
```pipe
print (trim "  hallo  ")     -- "hallo"
```

### split
```
split s trennzeichen
```
Teilt einen String an einem Trennzeichen und gibt eine Liste zurück.
```pipe
print (split "a,b,c" ",")           -- ["a", "b", "c"]
print (split "Hallo Welt" " ")      -- ["Hallo", "Welt"]
```

### join
```
join liste trennzeichen
```
Verbindet eine Liste von Strings mit einem Trennzeichen.
```pipe
print (join ["a", "b", "c"] "-")     -- "a-b-c"
```

### contains
```
contains s substr
```
Prüft, ob ein String einen Teilstring enthält. Funktioniert auch für Listen.
```pipe
print (contains "Hallo Welt" "Welt")     -- true
print (contains "Hallo Welt" "Mars")     -- false
print (contains [1, 2, 3] 2)             -- true
```

---

## 10.4 Listen-Operationen

### len
```
len collection
```
Gibt die Anzahl Elemente in einer Liste, Map oder die Länge eines Strings zurück.
```pipe
print (len [1, 2, 3])       -- 3
print (len "Hallo")         -- 5
print (len {a: 1, b: 2})    -- 2
```

### push
```
push liste element...
```
Hängt ein oder mehrere Elemente an eine Liste an (modifiziert die Liste).
```pipe
zahlen: [1, 2]
push zahlen 3
push zahlen 4 5 6
print zahlen     -- [1, 2, 3, 4, 5, 6]
```

### pop
```
pop liste
```
Entfernt das letzte Element einer Liste und gibt es zurück.
```pipe
zahlen: [1, 2, 3]
print (pop zahlen)     -- 3
print zahlen           -- [1, 2]
```

### at
```
at collection index
```
Gibt das Element an Position `index` zurück (0-basiert). Funktioniert für Listen und Strings.
```pipe
print (at [10, 20, 30] 1)     -- 20
print (at "Hallo" 0)           -- "H"
```

### sort
```
sort liste
```
Sortiert eine Liste (Zahlen numerisch, Strings alphabetisch).
```pipe
print (sort [3, 1, 2])              -- [1, 2, 3]
print (sort ["c", "a", "b"])        -- ["a", "b", "c"]
```

### range
```
range n
range a b
range a b schritt
```
Erzeugt eine Liste von Zahlen.

| Variante | Bedeutung |
|----------|----------|
| `range(n)` | `[0, 1, 2, ..., n-1]` |
| `range(a, b)` | `[a, a+1, a+2, ..., b-1]` |
| `range(a, b, step)` | `[a, a+step, a+2*step, ..., <b]` |

```pipe
print (range 5)           -- [0, 1, 2, 3, 4]
print (range 2 6)         -- [2, 3, 4, 5]
print (range 0 10 2)      -- [0, 2, 4, 6, 8]
```

### slice_list
```
slice_list liste start ende
```
Erzeugt eine Teilliste von Index start (inklusive) bis ende (exklusive).
```pipe
print (slice_list [10, 20, 30, 40] 1 3)     -- [20, 30]
```

---

## 10.5 Map-Operationen

### get
```
get map key
```
Gibt den Wert für einen Schlüssel zurück. `nil`, wenn der Schlüssel nicht existiert.
Funktioniert auch für Map-Zugriffe und (mit Integer-Index) für Listen.
```pipe
person: {name: "Anna", alter: 28}
print (get person "name")     -- "Anna"
print (get person "groesse")  -- nil
```

### set
```
set map key wert
```
Setzt einen Wert für einen Schlüssel in einer Map (modifiziert die Map).
```pipe
person: {name: "Anna"}
set person "alter" 29
print (get person "alter")    -- 29
```

### keys
```
keys map
```
Gibt alle Schlüssel einer Map als Liste zurück.
```pipe
print (keys {a: 1, b: 2})     -- ["a", "b"]
```

### values
```
values map
```
Gibt alle Werte einer Map als Liste zurück.
```pipe
print (values {a: 1, b: 2})     -- [1, 2]
```

---

## 10.6 Mathematik

### abs
```
abs n
```
Absolutwert einer Zahl.
```pipe
print (abs (-5))      -- 5
print (abs 3.14)      -- 3.14
```

### min
```
min a b...
```
Gibt das Minimum von zwei oder mehr Zahlen zurück.
```pipe
print (min 3 1 5 2)     -- 1
```

### max
```
max a b...
```
Gibt das Maximum von zwei oder mehr Zahlen zurück.
```pipe
print (max 3 1 5 2)     -- 5
```

### pow
```
pow basis exponent
```
Berechnet `basis` hoch `exponent`. Gibt einen Float zurück.
```pipe
print (pow 2 10)      -- 1024
print (pow 2 0.5)     -- 1.414... (Quadratwurzel)
```

### sqrt
```
sqrt n
```
Berechnet die Quadratwurzel.
```pipe
print (sqrt 16)      -- 4
print (sqrt 2)       -- 1.414...
```

### round
```
round n
```
Rundet eine Zahl zur nächsten Ganzzahl.
```pipe
print (round 3.7)     -- 4
print (round 3.2)     -- 3
```

---

## 10.7 Netzwerk und HTTP

### http_get
```
http_get url
```
Führt einen HTTP GET-Request aus. Gibt eine Map `{status, body}` zurück.
```pipe
antwort: http_get "https://httpbin.org/get"
print (get antwort "status")     -- 200
print (get antwort "body")       -- Antwort-Body
```

### http_post
```
http_post url body
```
Führt einen HTTP POST-Request mit einem Body aus. Timeout: 10 Sekunden.
```pipe
payload: to_json {name: "Pipe"}
antwort: http_post "https://httpbin.org/post" payload
print (get antwort "status")     -- 200
```

### http_get_json
```
http_get_json url
```
HTTP GET + automatisches JSON-Parsing. Gibt eine Map/Liste zurück.
```pipe
daten: http_get_json "https://api.github.com/users/torvalds"
print (get daten "name")          -- "Linus Torvalds"
print (get daten "public_repos")  -- 12
```

### parse_json
```
parse_json json_string
```
Parst einen JSON-String und gibt eine verschachtelte Map/Liste zurück.
```pipe
daten: parse_json "{\"name\": \"Pipe\", \"version\": 1}"
print (get daten "name")     -- "Pipe"
```

### to_json
```
to_json wert
```
Konvertiert einen Wert (Map, Liste, etc.) in einen JSON-String.
```pipe
print (to_json {name: "Pipe", version: 1})    -- {"name":"Pipe","version":1}
print (to_json [1, 2, 3])                     -- [1,2,3]
```

---

## 10.8 TCP

### tcp_listen
```
tcp_listen host port
```
Erstellt einen TCP-Listener auf dem angegebenen Host und Port.
```pipe
ln: tcp_listen "0.0.0.0" 9999
```

### tcp_connect
```
tcp_connect host port
```
Verbindet zu einem TCP-Server.
```pipe
conn: tcp_connect "127.0.0.1" 9999
```

### tcp_accept
```
tcp_accept listener
```
Akzeptiert eine eingehende TCP-Verbindung. Blockiert bis eine Verbindung eingeht.
```pipe
conn: tcp_accept ln
```

### tcp_read
```
tcp_read connection
```
Liest Daten von einer TCP-Verbindung (4096 Byte Puffer).
```pipe
nachricht: tcp_read conn
print nachricht
```

### tcp_write
```
tcp_write connection daten
```
Schreibt einen String auf eine TCP-Verbindung.
```pipe
tcp_write conn "Hallo Server!\n"
```

### tcp_close
```
tcp_close connection|listener
```
Schließt eine TCP-Verbindung oder einen TCP-Listener.
```pipe
tcp_close conn
tcp_close ln
```

---

## 10.9 Regex

### regex_match
```
regex_match muster text
```
Prüft, ob das Regex-Muster im Text vorkommt.
```pipe
print (regex_match "[0-9]+" "abc123")           -- true
print (regex_match "^\\d{3}$" "123")            -- true
print (regex_match "^\\S+@\\S+$" "a@b.com")     -- true
```

### regex_replace
```
regex_replace muster ersatz text
```
Ersetzt alle Vorkommen des Musters im Text.
```pipe
print (regex_replace "[0-9]" "#" "Tel: 0123"))  -- "Tel: ####"
print (regex_replace "<[^>]+>" " " "<p>Hi</p>")  -- " Hi "
```

---

## 10.10 Datum und Zeit

### now
```
now dummy
```
Gibt den aktuellen Unix-Timestamp in Sekunden zurück. Das Argument wird ignoriert
(existiert aus historischen Gründen — `0` übergeben).
```pipe
ts: now 0
print ts     -- 1785100000
```

### format_time
```
format_time timestamp layout
```
Formatiert einen Unix-Timestamp nach Go-Zeitformat.
Das Go-Zeitformat (Referenzzeit) ist: `2006-01-02 15:04:05`
```pipe
ts: now 0
print (format_time ts "2006-01-02")              -- "2026-07-28"
print (format_time ts "15:04:05")                -- "14:30:00"
print (format_time ts "Monday, 02 Jan 2006")     -- "Tuesday, 28 Jul 2026"
```

---

## 10.11 Zufall

### random
```
random dummy
```
Gibt eine Zufallszahl zwischen 0.0 und 1.0 (Float) zurück. Das Argument wird ignoriert
(`0` übergeben).
```pipe
print (random 0)     -- 0.370874...
```

### random_range
```
random_range min max
```
Gibt eine zufällige Ganzzahl im Bereich [min, max) zurück (max exklusive).
```pipe
print (random_range 1 7)      -- 4 (Würfel, 1-6)
print (random_range 1 101)    -- 42 (1-100)
```

---

## 10.12 Encoding

### base64_encode
```
base64_encode s
```
Kodiert einen String in Base64.
```pipe
print (base64_encode "Hallo")     -- "SGFsbG8="
```

### base64_decode
```
base64_decode s
```
Dekodiert einen Base64-String.
```pipe
print (base64_decode "SGFsbG8=")     -- "Hallo"
```

---

## 10.13 Typ-Prüfung

### type_of
```
type_of wert
```
Gibt den Typ eines Werts als String zurück. Mögliche Werte: `"INTEGER"`, `"FLOAT"`,
`"STRING"`, `"BOOLEAN"`, `"NIL"`, `"LIST"`, `"MAP"`, `"FUNCTION"`, `"CLOSURE"`,
`"COMPILED_FUNCTION"`, `"RETURN"`, `"BREAK"`, `"CONTINUE"`, `"DEFER"`, `"RESULT"`.
```pipe
print (type_of 42)          -- "INTEGER"
print (type_of 3.14)        -- "FLOAT"
print (type_of "Hallo")     -- "STRING"
print (type_of true)        -- "BOOLEAN"
print (type_of nil)         -- "NIL"
print (type_of [1, 2])      -- "LIST"
print (type_of {a: 1})      -- "MAP"
```

### is_num
```
is_num wert
```
Prüft, ob ein Wert eine Zahl ist (Integer oder Float).
```pipe
print (is_num 42)       -- true
print (is_num 3.14)     -- true
print (is_num "42")     -- false
```

### is_str
```
is_str wert
```
Prüft, ob ein Wert ein String ist.
```pipe
print (is_str "Hallo")     -- true
print (is_str 42)          -- false
```

### is_list
```
is_list wert
```
Prüft, ob ein Wert eine Liste ist.
```pipe
print (is_list [1, 2])     -- true
print (is_list "hi")       -- false
```

### is_map
```
is_map wert
```
Prüft, ob ein Wert eine Map ist.
```pipe
print (is_map {a: 1})     -- true
print (is_map [1, 2])     -- false
```

### is_nil
```
is_nil wert
```
Prüft, ob ein Wert nil ist.
```pipe
print (is_nil nil)     -- true
print (is_nil 0)       -- false
```

---

## 10.14 Typ-Konvertierung

### to_str
```
to_str wert
```
Konvertiert einen beliebigen Wert in einen String.
```pipe
print (to_str 42)          -- "42"
print (to_str 3.14)        -- "3.14"
print (to_str true)        -- "true"
print (to_str [1, 2, 3])   -- "[1, 2, 3]"
```

### to_num
```
to_num wert
```
Konvertiert einen Wert in eine Zahl. Strings werden geparst, bool wird zu 0/1,
alles andere zu 0.
```pipe
print (to_num "42")        -- 42
print (to_num "3.14")      -- 3.14
print (to_num true)        -- 1
print (to_num false)       -- 0
```

---

## 10.15 Result-Typ

### Ok
```
Ok wert
```
Erzeugt ein erfolgreiches Result.
```pipe
r: Ok 42
print (is_ok r)     -- true
```

### Err
```
Err nachricht
```
Erzeugt ein Fehler-Result.
```pipe
r: Err "Fehler"
print (is_err r)    -- true
```

### is_ok
```
is_ok result
```
Prüft, ob ein Result erfolgreich ist.
```pipe
print (is_ok (Ok 42))       -- true
print (is_ok (Err "x"))     -- false
```

### is_err
```
is_err result
```
Prüft, ob ein Result ein Fehler ist.
```pipe
print (is_err (Ok 42))       -- false
print (is_err (Err "x"))     -- true
```

### unwrap
```
unwrap result
```
Extrahiert den Wert eines erfolgreichen Results. **Bricht bei Fehler ab.**
```pipe
print (unwrap (Ok 42))     -- 42
-- unwrap (Err "x")        -- ERROR
```

### unwrap_or
```
unwrap_or result default
```
Extrahiert den Wert oder gibt einen Default-Wert zurück, falls das Result ein Fehler ist.
```pipe
print (unwrap_or (Ok 42) 0)       -- 42
print (unwrap_or (Err "x") 0)     -- 0
```

---

## 10.16 Konkurrenz (Tree-Walker only)

### go
```
go funktionsaufruf
```
Startet eine Funktion in einer neuen **Goroutine** (nebenläufig).
**Nur im Tree-Walker verfügbar.**
```pipe
go print "Nebenläufig"
print "Hauptprogramm"
-- Beide Ausgaben erscheinen (Reihenfolge nicht determiniert)
```

---

## 10.17 Übersicht aller Builtins

### IO & System (5)
`print`, `input`, `exec`, `env`, `sleep`

### Dateisystem (16)
`read_file`, `write_file`, `append_file`, `read_lines`, `file_exists`,
`file_delete`, `file_move`, `file_copy`, `file_size`, `file_type`,
`list_dir`, `make_dir`, `remove_dir`, `path_join`, `path_base`, `path_dir`, `path_ext`

### Strings (6)
`upper`, `lower`, `trim`, `split`, `join`, `contains`

### Listen (12)
`len`, `push`, `pop`, `at`, `slice_list`, `sort`, `range`,
`map`, `filter`, `reduce`, `each`

### Maps (4)
`get`, `set`, `keys`, `values`

### Mathematik (6)
`abs`, `min`, `max`, `pow`, `sqrt`, `round`

### Netzwerk & HTTP (5)
`http_get`, `http_post`, `http_get_json`, `parse_json`, `to_json`

### TCP (6)
`tcp_listen`, `tcp_connect`, `tcp_accept`, `tcp_read`, `tcp_write`, `tcp_close`

### Regex (2)
`regex_match`, `regex_replace`

### Datum & Zeit (2)
`now`, `format_time`

### Zufall (2)
`random`, `random_range`

### Encoding (2)
`base64_encode`, `base64_decode`

### Typ-Prüfung (6)
`type_of`, `is_num`, `is_str`, `is_list`, `is_map`, `is_nil`

### Konvertierung (2)
`to_str`, `to_num`

### Result-Typ (6)
`Ok`, `Err`, `is_ok`, `is_err`, `unwrap`, `unwrap_or`

### Konkurrenz (1)
`go` (Tree-Walker only)

**Gesamt: 81 Builtins**
