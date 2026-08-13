# 10. Builtin-Referenz

Pipe hat **206 eingebaute Funktionen** — keine externen Abhängigkeiten
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
-- "Hallo"
print "Hallo"
-- "Wert: 42"
print "Wert:" 42
-- "3"
print (1 + 2)
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
-- Exit-Code
print (get ergebnis "status")
```

### env
```
env name
```
Liest eine Umgebungsvariable. Gibt nil zurück, wenn sie nicht existiert.
```pipe
-- "/home/user"
print (env "HOME")
-- "/usr/bin:..."
print (env "PATH")
```

### sleep
```
sleep ms
```
Wartet für die angegebene Anzahl Millisekunden.
```pipe
-- 1 Sekunde warten
sleep 1000
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
-- Anzahl Zeilen
print (len zeilen)
-- Erste Zeile
print (at zeilen 0)
```

### file_exists
```
file_exists pfad
```
Prüft, ob eine Datei oder ein Verzeichnis existiert. Gibt `true`/`false` zurück.
```pipe
-- true
print (file_exists "/etc/hosts")
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
-- 1234
print (file_size "daten.txt")
```

### file_type
```
file_type pfad
```
Gibt `"file"` für Dateien oder `"dir"` für Verzeichnisse zurück.
```pipe
-- "dir"
print (file_type "/tmp")
-- "file"
print (file_type "/tmp/test.txt")
```

### list_dir
```
list_dir pfad?
```
Listet den Inhalt eines Verzeichnisses auf. Verzeichnisse werden mit `/` am Ende markiert.
Ohne Argument wird das aktuelle Verzeichnis gelistet.
```pipe
dateien: list_dir "."
-- ["main.pipe", "lib.pipe", "daten/"]
print dateien
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
-- "/home/user/docs"
print (path_join "/home" "user" "docs")
```

### path_base
```
path_base pfad
```
Extrahiert den Dateinamen aus einem Pfad.
```pipe
-- "c.txt"
print (path_base "/a/b/c.txt")
```

### path_dir
```
path_dir pfad
```
Extrahiert das Verzeichnis aus einem Pfad.
```pipe
-- "/a/b"
print (path_dir "/a/b/c.txt")
```

### path_ext
```
path_ext pfad
```
Extrahiert die Dateiendung aus einem Pfad.
```pipe
-- ".txt"
print (path_ext "/a/b/c.txt")
-- ""
print (path_ext "/a/b/file")
```

### file_open
```
file_open pfad modus
```
Öffnet eine Datei im Direktzugriffsmodus und liefert einen numerischen Handle. Modi: `"r"` (lesen), `"w"` (schreiben, leeren), `"a"` (anhängen), `"rw"` (lesen/schreiben, erhalten), `"rw+"` (lesen/schreiben, leeren). Beachtet das aktive Sandbox-Profil.
```pipe
h: file_open "/tmp/data.bin" "rw"
```

### file_close
```
file_close handle
```
Schließt eine mit `file_open` geöffnete Datei und gibt den Handle frei.

### file_read
```
file_read handle offset n
```
Liest `n` Bytes ab `offset` und liefert sie als `bytes` (bei Dateiende weniger).
```pipe
-- erste 8 Bytes
file_read h 0 8
```

### file_write
```
file_write handle offset daten
```
Schreibt `daten` (bytes oder String) ab `offset` und überschreibt vorhandene Bytes. Liefert die Anzahl geschriebener Bytes.
```pipe
file_write h 0 (to_bytes "0123456789")
```

### file_truncate
```
file_truncate handle größe
```
Kürzt die Datei auf exakt `größe` Bytes.

### file_sync
```
file_sync handle
```
Schreibt Dateidaten und Metadaten auf den Datenträger (fsync).

---

## 10.3 String-Operationen

### upper
```
upper s
```
Konvertiert einen String in Großbuchstaben.
```pipe
-- "HALLO"
print (upper "hallo")
```

### lower
```
lower s
```
Konvertiert einen String in Kleinbuchstaben.
```pipe
-- "hallo"
print (lower "HALLO")
```

### trim
```
trim s
```
Entfernt Leerzeichen (Whitespace) am Anfang und Ende.
```pipe
-- "hallo"
print (trim "  hallo  ")
```

### split
```
split s trennzeichen
```
Teilt einen String an einem Trennzeichen und gibt eine Liste zurück.
```pipe
-- ["a", "b", "c"]
print (split "a,b,c" ",")
-- ["Hallo", "Welt"]
print (split "Hallo Welt" " ")
```

### join
```
join liste trennzeichen
```
Verbindet eine Liste von Strings mit einem Trennzeichen.
```pipe
-- "a-b-c"
print (join (["a", "b", "c"]) "-")
```

### contains
```
contains s substr
```
Prüft, ob ein String einen Teilstring enthält. Funktioniert auch für Listen.
```pipe
-- true
print (contains "Hallo Welt" "Welt")
-- false
print (contains "Hallo Welt" "Mars")
-- true
print (contains ([1, 2, 3]) 2)
```

### substring
```
substring s start ende
```
Liefert den Teilstring von `s` von `start` (inklusive) bis `ende` (exklusive). Indizes werden an die String-Grenzen geklemmt.
```pipe
-- "el"
print (substring "hello" 1 3)
```

### index_of
```
index_of s nadel
```
Liefert den 0-basierten Index des ersten Vorkommens von `nadel` in `s`, sonst `-1`.
```pipe
-- 6
print (index_of "hello world" "world")
```

---

## 10.4 Listen-Operationen

### len
```
len collection
```
Gibt die Anzahl Elemente in einer Liste, Map oder die Länge eines Strings zurück.
```pipe
-- 3
print (len ([1, 2, 3]))
-- 5
print (len "Hallo")
-- 2
print (len ({a: 1, b: 2}))
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
-- [1, 2, 3, 4, 5, 6]
print zahlen
```

### pop
```
pop liste
```
Entfernt das letzte Element einer Liste und gibt es zurück.
```pipe
zahlen: [1, 2, 3]
-- 3
print (pop zahlen)
-- [1, 2]
print zahlen
```

### at
```
at collection index
```
Gibt das Element an Position `index` zurück (0-basiert). Funktioniert für Listen und Strings.
```pipe
-- 20
print (at ([10, 20, 30]) 1)
-- "H"
print (at "Hallo" 0)
```

### sort
```
sort liste
sort liste vergleich
```
Liefert eine neue, sortierte Liste. Ohne Vergleichsfunktion: Zahlen numerisch, Strings alphabetisch. Mit `vergleich(a, b)`, das truthy liefert, wenn `a` vor `b` sortieren soll, wird diese für die Ordnung verwendet.
```pipe
-- [1, 2, 3]
print (sort ([3, 1, 2]))
-- ["a", "b", "c"]
print (sort (["c", "a", "b"]))
-- absteigend
print (sort [1, 2, 3] (fn a b: b < a))
```

### sorted_by
```
sorted_by liste schluesselfn
```
Liefert eine neue Liste, sortiert nach dem Schlüssel, den `schluesselfn(element)` für jedes Element liefert.
```pipe
-- ["a", "bb", "ccc"]
print (sorted_by ["ccc", "a", "bb"] (fn s: len s))
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
-- [0, 1, 2, 3, 4]
print (range 5)
-- [2, 3, 4, 5]
print (range 2 6)
-- [0, 2, 4, 6, 8]
print (range 0 10 2)
```

### slice_list
```
slice_list liste start ende
```
Erzeugt eine Teilliste von Index start (inklusive) bis ende (exklusive). Die `x[a..b]`-Syntax funktioniert für Listen, Strings und Bytes.
```pipe
-- [20, 30]
print (slice_list ([10, 20, 30, 40]) 1 3)
```

### slice
```
slice wert start ende
```
Liefert einen Ausschnitt von `wert` (Liste, String oder Bytes) von `start` (inklusive) bis `ende` (exklusive). Indizes werden geklemmt.
```pipe
-- [20, 30]
print (slice ([10, 20, 30, 40]) 1 3)
-- "el"
print (slice "hello" 1 3)
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
-- "Anna"
print (get person "name")
-- nil
print (get person "groesse")
```

### set
```
set map key wert
```
Setzt einen Wert für einen Schlüssel in einer Map (modifiziert die Map).
```pipe
person: {name: "Anna"}
set person "alter" 29
-- 29
print (get person "alter")
```

### keys
```
keys map
```
Gibt alle Schlüssel einer Map als Liste zurück.
```pipe
-- ["a", "b"]
print (keys ({a: 1, b: 2}))
```

### values
```
values map
```
Gibt alle Werte einer Map als Liste zurück.
```pipe
-- [1, 2]
print (values ({a: 1, b: 2}))
```

---

## 10.6 Mathematik

### abs
```
abs n
```
Absolutwert einer Zahl.
```pipe
-- 5
print (abs (-5))
-- 3[14]
print (abs 3[14])
```

### min
```
min a b...
```
Gibt das Minimum von zwei oder mehr Zahlen zurück.
```pipe
-- 1
print (min 3 1 5 2)
```

### max
```
max a b...
```
Gibt das Maximum von zwei oder mehr Zahlen zurück.
```pipe
-- 5
print (max 3 1 5 2)
```

### pow
```
pow basis exponent
```
Berechnet `basis` hoch `exponent`. Gibt einen Float zurück.
```pipe
-- 1024
print (pow 2 10)
-- 1.414... (Quadratwurzel)
print (pow 2 0[5])
```

### sqrt
```
sqrt n
```
Berechnet die Quadratwurzel.
```pipe
-- 4
print (sqrt 16)
-- 1.414...
print (sqrt 2)
```

### round
```
round n
```
Rundet eine Zahl zur nächsten Ganzzahl.
```pipe
-- 4
print (round 3[7])
-- 3
print (round 3[2])
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
-- 200
print (get antwort "status")
-- Antwort-Body
print (get antwort "body")
```

### http_post
```
http_post url body
```
Führt einen HTTP POST-Request mit einem Body aus. Timeout: 10 Sekunden.
```pipe
payload: to_json {name: "Pipe"}
antwort: http_post "https://httpbin.org/post" payload
-- 200
print (get antwort "status")
```

### http_get_json
```
http_get_json url
```
HTTP GET + automatisches JSON-Parsing. Gibt eine Map/Liste zurück.
```pipe
daten: http_get_json "https://api.github.com/users/torvalds"
-- "Linus Torvalds"
print (get daten "name")
-- 12
print (get daten "public_repos")
```

### http_request
```
http_request method url headers? body?
```
Führt einen HTTP-Request mit beliebiger Methode, URL, optionalem Headers-Map und Body aus. Gibt eine Map `{status, headers, body}` zurück.
```pipe
h: {}
set h "Content-Type" "application/json"
r: http_request "POST" "https://api.example.com/data" h "{\"key\":\"value\"}"
print (get r "status")
print (get r "body")
```

### http_server
```
http_server addr handler
```
Startet einen HTTP-Server auf `addr` (z.B. `"0.0.0.0:8080"`). `handler` ist eine Funktion `fn(req)`, die eine Request-Map `{method, path, query, headers, body}` erhält und eine Response-Map `{status, headers, body}` zurückgeben muss. Der Server läuft im Hintergrund. Gibt ein Server-Handle zurück.
```pipe
fn handle req
    name: get req "path"
    {status: 200, headers: {}, body: "Hallo " ++ name}

server: http_server "0.0.0.0:8080" handle
print "Server läuft auf :8080"
sleep 60000
http_close server
```

### http_close
```
http_close server
```
Fährt einen HTTP-Server herunter und gibt den Port frei.
```pipe
http_close server
```

### parse_json
```
parse_json json_string
```
Parst einen JSON-String und gibt eine verschachtelte Map/Liste zurück.
```pipe
daten: parse_json "{\"name\": \"Pipe\", \"version\": 1}"
-- "Pipe"
print (get daten "name")
```

### to_json
```
to_json wert
```
Konvertiert einen Wert (Map, Liste, etc.) in einen JSON-String.
```pipe
-- {name: "Pipe",version: 1}
print (to_json ({name: "Pipe", version: 1}))
-- [1,2,3]
print (to_json ([1, 2, 3]))
```

---

## 10.8 TCP

### tcp_listen
```
tcp_listen host port
```
Erstellt einen TCP-Listener auf dem angegebenen Host und Port.
```pipe
ln: tcp_listen "0.0.0[0]" 9999
```

### tcp_connect
```
tcp_connect host port
```
Verbindet zu einem TCP-Server.
```pipe
conn: tcp_connect "127.0.0[1]" 9999
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
-- true
print (regex_match "[0-9]+" "abc123")
-- true
print (regex_match "^\\d{3}$" "123")
-- true
print (regex_match "^\\S+@\\S+$" "a@b.com")
```

### regex_replace
```
regex_replace muster ersatz text
```
Ersetzt alle Vorkommen des Musters im Text.
```pipe
-- "Tel: ####"
print (regex_replace "[0-9]" "#" "Tel: 0123")
-- " Hi "
print (regex_replace "<[^>]+>" " " "<p>Hi</p>")
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
-- 1785100000
print ts
```

### format_time
```
format_time timestamp layout
```
Formatiert einen Unix-Timestamp nach Go-Zeitformat.
Das Go-Zeitformat (Referenzzeit) ist: `2006-01-02 15:04:05`
```pipe
ts: now 0
-- "2026-07-28"
print (format_time ts "2006-01-02")
-- "14:30:00"
print (format_time ts "15:04:05")
-- "Tuesday, 28 Jul 2026"
print (format_time ts "Monday, 02 Jan 2006")
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
-- 0.370874...
print (random 0)
```

### random_range
```
random_range min max
```
Gibt eine zufällige Ganzzahl im Bereich [min, max) zurück (max exklusive).
```pipe
-- 4 (Würfel, 1-6)
print (random_range 1 7)
-- 42 (1-100)
print (random_range 1 101)
```

---

## 10.12 Kryptographie (8 Funktionen)

> **Hinweis:** Im Gegensatz zu `random` und `random_range` (die `math/rand` verwenden) nutzen diese Funktionen `crypto/rand` und sind für kryptografische Zwecke wie Schlüsselgenerierung, Token und Nonces geeignet.

### secure_random
```
secure_random byte_anzahl
```
Erzeugt `byte_anzahl` kryptografisch sichere Zufallsbytes, als Hex-String. Maximal 1024 Bytes pro Aufruf.
```pipe
-- 32-stelliger Hex-String (16 Bytes)
key: secure_random 16
```

### secure_random_int
```
secure_random_int
```
Gibt eine kryptografisch sichere 64-Bit-Ganzzahl zurück.
```pipe
token: secure_random_int
```

### secure_random_range
```
secure_random_range min max
```
Gibt eine kryptografisch sichere Ganzzahl im Bereich `[min, max)` zurück.
```pipe
-- Zufallszahl zwischen 100000 und 999999
code: secure_random_range 100000 1000000
```

### secure_random_bytes
```
secure_random_bytes byte_anzahl
```
Gibt kryptografisch sichere Zufallsbytes zurück (raw bytes, kein Hex). Für Schlüsselmaterial, IVs, Nonces. Max 1024 Bytes.
```pipe
key: secure_random_bytes 32
```

### encrypt
```
encrypt key plaintext [associated_data]
```
Verschlüsselt `plaintext` mit AES-GCM. Der Key kann ein String (16/24/32 Bytes), ein Hex-String von `secure_random` oder raw Bytes sein. Eine zufällige Nonce wird vorangestellt. Optionales `associated_data` wird authentifiziert aber nicht verschlüsselt (AEAD).
```pipe
key: secure_random 32
enc: encrypt key "Hello World"
-- "g+F+k0q...base64..."

-- Mit Associated Data
enc: encrypt key "Hello" "meta-info"
```

### decrypt
```
decrypt key ciphertext [associated_data]
```
Entschlüsselt AES-GCM-Chiffretext. Der Key muss mit dem Verschlüsselungs-Key übereinstimmen. Bei falschem Key oder manipulierten Daten → Fehler.
```pipe
decrypt key enc           -- "Hello World"
decrypt wrong_key enc     -- ERROR: authentication failed
```

### hmac_sha256
```
hmac_sha256 key message
```
Berechnet HMAC-SHA256(key, message). Hex-kodierter 32-Byte MAC. Für Nachrichten-Authentifizierung, API-Signing, JWT.
```pipe
sig: hmac_sha256 "secret-key" "Transfer 100 EUR"
```

### hmac_sha512
```
hmac_sha512 key message
```
Berechnet HMAC-SHA512(key, message). Hex-kodierter 64-Byte MAC. Höhere Sicherheit als SHA256.
```pipe
sig: hmac_sha512 "key" "data"
```

---

## 10.13 Encoding (2 Funktionen)

### base64_encode
```
base64_encode s
```
Kodiert einen String in Base64.
```pipe
-- "SGFsbG8="
print (base64_encode "Hallo")
```

### base64_decode
```
base64_decode s
```
Dekodiert einen Base64-String.
```pipe
-- "Hallo"
print (base64_decode "SGFsbG8=")
```

---

## 10.14 Typ-Prüfung (6 Funktionen)

### type_of
```
type_of wert
```
Gibt den Typ eines Werts als String zurück. Mögliche Werte: `"INTEGER"`, `"FLOAT"`,
`"STRING"`, `"BOOLEAN"`, `"NIL"`, `"LIST"`, `"MAP"`, `"FUNCTION"`, `"CLOSURE"`,
`"COMPILED_FUNCTION"`, `"RETURN"`, `"BREAK"`, `"CONTINUE"`, `"DEFER"`, `"RESULT"`, `"STRUCT"`.
```pipe
-- "INTEGER"
print (type_of 42)
-- "FLOAT"
print (type_of 3[14])
-- "STRING"
print (type_of "Hallo")
-- "BOOLEAN"
print (type_of true)
-- "NIL"
print (type_of nil)
-- "LIST"
print (type_of ([1, 2]))
-- "MAP"
print (type_of {a: 1})
-- "STRUCT"
struct P: x, y
print (type_of P)
```

### is_num
```
is_num wert
```
Prüft, ob ein Wert eine Zahl ist (Integer oder Float).
```pipe
-- true
print (is_num 42)
-- true
print (is_num 3[14])
-- false
print (is_num "42")
```

### is_str
```
is_str wert
```
Prüft, ob ein Wert ein String ist.
```pipe
-- true
print (is_str "Hallo")
-- false
print (is_str 42)
```

### is_list
```
is_list wert
```
Prüft, ob ein Wert eine Liste ist.
```pipe
-- true
print (is_list ([1, 2]))
-- false
print (is_list "hi")
```

### is_map
```
is_map wert
```
Prüft, ob ein Wert eine Map ist.
```pipe
-- true
print (is_map {a: 1})
-- false
print (is_map ([1, 2]))
```

### is_nil
```
is_nil wert
```
Prüft, ob ein Wert nil ist.
```pipe
-- true
print (is_nil nil)
-- false
print (is_nil 0)
```

---

## 10.15 Typ-Konvertierung (2 Funktionen)

### to_str
```
to_str wert
```
Konvertiert einen beliebigen Wert in einen String.
```pipe
-- "42"
print (to_str 42)
-- "3[14]"
print (to_str 3[14])
-- "true"
print (to_str true)
-- "[1, 2, 3]"
print (to_str ([1, 2, 3]))
```

### to_num
```
to_num wert
```
Konvertiert einen Wert in eine Zahl. Strings werden geparst, bool wird zu 0/1,
alles andere zu 0.
```pipe
-- 42
print (to_num "42")
-- 3[14]
print (to_num "3[14]")
-- 1
print (to_num true)
-- 0
print (to_num false)
```

---

## 10.16 Result-Typ (6 Funktionen)

### Ok
```
Ok wert
```
Erzeugt ein erfolgreiches Result.
```pipe
r: Ok 42
-- true
print (is_ok r)
```

### Err
```
Err nachricht
```
Erzeugt ein Fehler-Result.
```pipe
r: Err "Fehler"
-- true
print (is_err r)
```

### is_ok
```
is_ok result
```
Prüft, ob ein Result erfolgreich ist.
```pipe
-- true
print (is_ok (Ok 42))
-- false
print (is_ok (Err "x"))
```

### is_err
```
is_err result
```
Prüft, ob ein Result ein Fehler ist.
```pipe
-- false
print (is_err (Ok 42))
-- true
print (is_err (Err "x"))
```

### unwrap
```
unwrap result
```
Extrahiert den Wert eines erfolgreichen Results. **Bricht bei Fehler ab.**
```pipe
-- 42
print (unwrap (Ok 42))
-- unwrap (Err "x")        -- ERROR
```

### unwrap_or
```
unwrap_or result default
```
Extrahiert den Wert oder gibt einen Default-Wert zurück, falls das Result ein Fehler ist.
```pipe
-- 42
print (unwrap_or (Ok 42) 0)
-- 0
print (unwrap_or (Err "x") 0)
```

---

## 10.17 Konkurrenz (Tree-Walker only)

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

## 10.18 Sandbox (5 Funktionen)

### sandbox_profile
```
sandbox_profile name
```
Wählt ein Sandbox-Profil aus (`none`, `strict`, `noexec`, `isolated`, `networked`). Definiert ein benanntes Profil mit `sandbox_profile "name" {fs: ..., network: ..., exec: ..., ai: ...}`. Während ein restriktives Profil aktiv ist, können nur Profile registriert werden, die nicht mehr Rechte (eine Teilmenge) einräumen.
```pipe
sandbox_profile "strict"
```

### set_sandbox
```
set_sandbox(profil)
```
Setzt die aktive Sandbox aus einem Profil-Map oder einem Namen. Von einem Nicht-`none`-Profil aus sind nur gleich- oder restriktivere Profile erreichbar (der Sandbox kann nur einschränken; `none` ist unerreichbar, sobald ein anderes Profil aktiv ist).
```pipe
set_sandbox ({type: "strict", write: false})
```

### with_sandbox
```
with_sandbox(profil, fn)
```
Führt `fn` unter dem angegebenen Sandbox-Profil aus und stellt danach das vorherige Profil wieder her.
```pipe
with_sandbox "noexec" (fn
    print "isoliert")
```

### audit_log
```
audit_log()
```
Gibt den Audit-Trail des aktiven Profils als Liste von Maps zurück. Wird nur
gefüllt, wenn das Profil mit `audit_log: true` definiert wurde.
```pipe
sandbox_profile "audited" {fs: "full", network: true, exec: false, ai: true, audit_log: true}
set_sandbox "audited"

http_get "https://example.com"
for eintrag in audit_log
    print eintrag.event ++ " -> " ++ eintrag.detail
-- -> http_get -> https://example.com
```

### budget_spent
```
budget_spent()
```
Gibt die insgesamt für das aktive Profil verbuchten KI-Kosten in USD zurück.
Wird zusammen mit dem `budget`-Schlüssel zur Überwachung oder Durchsetzung von
Ausgabelimits verwendet.
```pipe
sandbox_profile "budgeted" {fs: "full", network: false, exec: false, ai: true, budget: 0.01}
set_sandbox "budgeted"

ask "Hallo"
print (budget_spent)
-- -> 0.000079 (ca.)
```

---

## 10.19 KI — Kosten-Tracking (4 Funktionen)

### ai_cost
```
ai_cost()
```
Gibt die kumulierten Kosten-Metriken des aktuellen Laufs als Map zurück. Mit
dem String `"reset"` werden alle Metriken zurückgesetzt.
```pipe
print (ai_cost)
-- -> {cache_hits: 0, calls: 2, cost_usd: 0.00012, cache_misses: 1}

ai_cost "reset"   -- alle Metriken zurücksetzen
```

### ai_tokens
```
ai_tokens()
```
Gibt die Gesamtzahl der Tokens zurück, die alle KI-Aufrufe im aktuellen Lauf
verbraucht haben.

### ai_cache_hits
```
ai_cache_hits()
```
Gibt zurück, wie viele KI-Antworten aus dem Antwort-Cache bedient wurden.

### ai_cache_misses
```
ai_cache_misses()
```
Gibt zurück, wie viele KI-Antworten vom Provider geladen wurden (Cache-Miss).
```pipe
ask "Was ist ein Monad?" > print
ask "Was ist ein Monad?" > print
print (ai_cache_hits)   -- 1
print (ai_cache_misses) -- 1
```

---

## 10.20 Bytes und Binär (15 Funktionen)

### to_bytes
```
to_bytes wert
```
Wandelt einen String in seine UTF-8-Bytes oder eine Liste von Zahlen (0-255) in Bytes um. Bytes werden unverändert zurückgegeben.
```pipe
-- 0x4869
print (to_bytes "Hi")
-- 0x01ff
print (to_bytes [1, 255])
```

### from_bytes
```
from_bytes wert
```
Wandelt Bytes in einen String um (UTF-8-Dekodierung). Strings werden unverändert zurückgegeben.
```pipe
-- "Hi"
print (from_bytes (to_bytes "Hi"))
```

### bytes_append
```
bytes_append bytes chunk ...
```
Hängt einen oder mehrere Blöcke (Bytes oder Strings) an `bytes` an.
```pipe
-- 0x0102
print (bytes_append (to_bytes [1]) (to_bytes [2]))
```

### bytes_to_int
```
bytes_to_int bytes offset? n?
```
Interpretiert `n` Big-Endian-Bytes (max. 8) ab `offset` als vorzeichenlose Ganzzahl. Standardmäßig die gesamten `bytes`.
```pipe
-- 258
print (bytes_to_int (to_bytes [1, 2]) 0 2)
```

### int_to_bytes
```
int_to_bytes wert n?
```
Kodiert eine nicht-negative Ganzzahl als Big-Endian-Bytes (minimale Länge, oder exakt `n` Bytes).
```pipe
-- 0x0102
print (int_to_bytes 258 2)
```

### bytes_compare
```
bytes_compare a b
```
Vergleicht zwei Byte-Sequenzen lexikografisch. Negativ wenn `a < b`, 0 bei Gleichheit, positiv wenn `a > b`.
```pipe
-- -1
print (bytes_compare (to_bytes [1]) (to_bytes [2]))
```

### hex_encode
```
hex_encode bytes
```
Kodiert Bytes als hexadezimale Zeichenkette (Kleinbuchstaben).
```pipe
-- "4869"
print (hex_encode (to_bytes "Hi"))
```

### hex_decode
```
hex_decode s
```
Dekodiert eine hexadezimale Zeichenkette in Bytes.
```pipe
-- 0x4869
print (hex_decode "4869")
```

### bit_and
```
bit_and a b
```
Bitweises UND zweier Ganzzahlen.
```pipe
-- 2
print (bit_and 6 3)
```

### bit_or
```
bit_or a b
```
Bitweises ODER zweier Ganzzahlen.
```pipe
-- 7
print (bit_or 6 3)
```

### bit_xor
```
bit_xor a b
```
Bitweises XOR zweier Ganzzahlen.
```pipe
-- 5
print (bit_xor 6 3)
```

### bit_not
```
bit_not a
```
Bitweise Negation einer Ganzzahl (Zweierkomplement).
```pipe
-- -6
print (bit_not 5)
```

### bit_lshift
```
bit_lshift a n
```
Schiebt `a` um `n` Positionen nach links. `n` muss 0-63 sein.
```pipe
-- 1024
print (bit_lshift 1 10)
```

### bit_rshift
```
bit_rshift a n
```
Schiebt `a` um `n` Positionen nach rechts. `n` muss 0-63 sein.
```pipe
-- 16
print (bit_rshift 256 4)
```

### crc32
```
crc32 wert
```
Berechnet die IEEE-CRC-32-Prüfsumme eines Strings oder von Bytes.
```pipe
-- 907060870
print (crc32 "hello")
```

---

## 10.21 MCP — Model Context Protocol (13 Funktionen)

Pipe implementiert das Model Context Protocol sowohl als **Server** (stellt Tools, Ressourcen und Prompts für externe Clients wie Claude Desktop bereit) als auch als **Client** (nutzt externe MCP-Server in `ai_with_tools`). Transporte: stdio und Streamable HTTP. Die vollständige Anleitung steht in [Kapitel 25](25-mcp.md).

### mcp_server
```
mcp_server(name, version)
```
Erstellt einen MCP-Server. Bridgt automatisch alle über `ai_tool` registrierten Funktionen als MCP-Tools sowie alle registrierten Ressourcen und Prompts.
```pipe
mcp_server "Pipe Agent" "1.0.0"
```

### mcp_serve_stdio
```
mcp_serve_stdio
```
Startet den Server auf stdin/stdout (blockierend). Für Claude Desktop, Cursor usw.
```pipe
mcp_server "Pipe Agent" "1.0.0"
mcp_serve_stdio
```

### mcp_serve_sse
```
mcp_serve_sse(addr)
```
Startet einen Streamable-HTTP-Server auf `addr` (z. B. `:9090`, blockierend). Clients verbinden sich per `POST` + SSE; Sessions werden über `Mcp-Session-Id` verwaltet.
```pipe
mcp_server "Pipe Agent" "1.0.0"
mcp_serve_sse ":9090"
```

### mcp_tools
```
mcp_tools
```
Listet alle registrierten Tools (lokal + remote) als Liste von Maps mit `name`, `description` und `source`.
```pipe
print "Alle Tools: " ++ (to_str (len (mcp_tools)))
```

### mcp_resource
```
mcp_resource(uri, name, mime, read_fn)
```
Registriert eine statische Resource. `read_fn(uri)` wird mit der angefragten URI aufgerufen und liefert den Resourcentext.
```pipe
fn read_docs uri
    "Dokumentation zu " ++ uri

mcp_resource "docs://pipe" "Pipe-Doku" "text/markdown" read_docs
```

### mcp_resource_template
```
mcp_resource_template(template, name, mime, read_fn)
```
Registriert eine URI-Template-Resource, z. B. `file:///{path}`. `read_fn(uri)` wird mit der konkreten URI jeder passenden Anfrage aufgerufen.
```pipe
fn read_file uri
    content: read_file (replace uri "file:///" "")
    content

mcp_resource_template "file:///{path}" "Datei" "text/plain" read_file
```

### mcp_prompt
```
mcp_prompt(name, description, args_map, build_fn)
```
Registriert eine Prompt-Vorlage. `args_map` bildet Argumentnamen auf eine Beschreibung (String) oder auf eine Map mit `description` und optionalem `required` (Standard `true`) ab. `build_fn(args)` liefert den gerenderten Prompttext.
```pipe
fn build_summary args
    "Bitte zusammenfassen: " ++ (get args "text")

mcp_prompt "summarize" "Fasse Text zusammen" {text: "Der Text"} build_summary
```

### mcp_resources
```
mcp_resources
```
Listet alle Ressourcen (lokale Registrierungen + Remote-Clients) als Liste von Maps mit `uri`, `name`, `mimeType`, `description` und `source`.
```pipe
print (mcp_resources)
```

### mcp_read_resource
```
mcp_read_resource(uri)
```
Liest eine Resource (statisch oder per Template-Match) aus den lokalen Registries, vom lokalen Server oder von einem verbundenen MCP-Client. Funktioniert auch ohne laufenden Server.
```pipe
print (mcp_read_resource "docs://pipe")
```

### mcp_prompts
```
mcp_prompts
```
Listet alle Prompts (lokale Registrierungen + Remote-Clients) als Liste von Maps mit `name`, `description` und `source`.
```pipe
print (mcp_prompts)
```

### mcp_prompt_get
```
mcp_prompt_get(name, args?)
```
Rendert einen Prompt aus den lokalen Registries, vom lokalen Server oder von einem verbundenen MCP-Client. Fehlende Pflicht-Argumente werden mit einem Fehler abgelehnt.
```pipe
print (mcp_prompt_get "summarize" {text: "Ein langer Artikel ..."})
```

### mcp_use_stdio
```
mcp_use_stdio(command, args...)
```
Startet einen Subprozess und verbindet sich per stdio als MCP-Client. Entdeckt dessen Tools und registriert sie mit Präfix `mcp0_`, `mcp1_`, ... im Tool-Registry. Ressourcen und Prompts werden ebenfalls entdeckt, falls beworben. Unterliegt der `exec`-Richtlinie des aktiven Sandbox-Profils (blockiert, außer `exec: true`).
```pipe
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-everything"
```

### mcp_use_sse
```
mcp_use_sse(url)
```
Verbindet sich per POST + SSE mit einem Streamable-HTTP-MCP-Server (Session-verwaltet). Registriert dessen Tools mit Präfix `mcp2_`, `mcp3_`, ... Unterliegt der `network`-Richtlinie des aktiven Sandbox-Profils: Die URL und jede weitere Anfrage (inkl. Redirects) werden gegen die `network_whitelist` des Profils geprüft.
```pipe
mcp_use_sse "http://localhost:9090/"
```

---

## 10.22 Übersicht aller Builtins

### IO & System (8)
`print`, `input`, `exec`, `env`, `sleep`, `args`, `read_stdin`, `go`

### Dateisystem (23)
`read_file`, `write_file`, `append_file`, `read_lines`, `file_exists`,
`file_delete`, `file_move`, `file_copy`, `file_size`, `file_type`,
`list_dir`, `make_dir`, `remove_dir`, `path_join`, `path_base`, `path_dir`, `path_ext`,
`file_open`, `file_close`, `file_read`, `file_write`, `file_truncate`, `file_sync`

### Strings (9)
`upper`, `lower`, `trim`, `split`, `join`, `contains`, `repeat`, `substring`, `index_of`

### Listen (13)
`len`, `push`, `pop`, `at`, `slice_list`, `slice`, `sort`, `sorted_by`, `range`,
`map`, `filter`, `reduce`, `each`

### Bytes und Binär (15)
`to_bytes`, `from_bytes`, `bytes_append`, `bytes_to_int`, `int_to_bytes`,
`bytes_compare`, `hex_encode`, `hex_decode`, `bit_and`, `bit_or`, `bit_xor`,
`bit_not`, `bit_lshift`, `bit_rshift`, `crc32`

### Maps (4)
`get`, `set`, `keys`, `values`

### Mathematik (6)
`abs`, `min`, `max`, `pow`, `sqrt`, `round`

### Netzwerk & HTTP (6)
`http_get`, `http_post`, `http_get_json`, `http_request`, `parse_json`, `to_json`

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

### CSV (2)
`csv_parse`, `csv_format`

### Hashing (4)
`sha256`, `md5`, `sha1`, `sha512`

### Datenbank — SQLite (Modul)
`db_open`, `db_close`, `db_exec`, `db_query`, `q`, `exec`, `row_get`, `row_eq`, `row_ne`

Die `db_*`-Builtins wurden durch ein natives Pipe-Modul ersetzt, das in der [`pipe-modules`](https://github.com/MachuraHarry/pipe-modules)-Registry verfügbar ist. Die externe `modernc.org/sqlite`-Abhängigkeit ist entfernt — Binary wieder dependency-free (~7 MB).

Installiere mit `pipe -get sqlite`, dann import mit `import "sqlite"`:

```pipe
-- Klassische API
import "sqlite"
h: db_open ":memory:"
db_exec h "CREATE TABLE tasks (id INTEGER PRIMARY KEY, title TEXT, priority TEXT)"
db_exec h "INSERT INTO tasks VALUES (1, 'Bug fixen', 'high'), (2, 'Doku updaten', 'medium')"
rows: db_query h "SELECT * FROM tasks WHERE priority = 'high'"
for row in rows
    print (row.title)
db_close h

-- Pipeline-API (komponierbar via > Operator)
fn is_high row
    (row.priority) == "high"
q h "SELECT * FROM tasks" > filter is_high > each print
```

**Unterstütztes SQL:** CREATE TABLE, INSERT (auch multi-value), UPDATE, DELETE, SELECT mit WHERE, GROUP BY + Aggregaten (COUNT, SUM, AVG, MIN, MAX), ORDER BY, LIMIT/OFFSET, JOIN (INNER, LEFT, RIGHT), DISTINCT, BEGIN/COMMIT/ROLLBACK. SQL ist case-insensitiv.

**Pipeline-Helper:** `q` / `exec` (Short-Aliase), `row_get` / `row_eq` / `row_ne` (Filter-Prädikate). Demo: `examples/sqlite_pipeline.pipe`.

Persistenz erfolgt über ein binäres Paging-Format mit CRC32-Prüfsummen; `":memory:"` für In-Memory-Datenbanken.

Siehe das Kapitel [SQLite-Modul](26-sqlite-modul.md) für Architektur-Details und Benchmarks (Pipe vs Python vs Lua).

> **Hinweis:** Der TV-Modus ist vollständig funktionsfähig — alle Operationen laufen (CREATE, INSERT, SELECT, WHERE, GROUP BY, UPDATE, DELETE). Der VM-Modus hat einen bekannten Compiler-Bug bei großen Modul-Importen — einzelne Operationen funktionieren, aber komplexe Dispatch-Aufrufe (db_exec → exec_insert) scheitern. Siehe das Kapitel [SQLite-Modul](26-sqlite-modul.md).

| Funktion | Signatur | Beschreibung |
|----------|----------|--------------|
| `db_open` | `db_open(path)` | Öffnet Datenbank-Datei oder `":memory:"`, gibt Handle zurück |
| `db_close` | `db_close(handle)` | Schließt Datenbank und persistiert Änderungen |
| `db_exec` | `db_exec(handle, sql)` | Führt DDL/DML aus, gibt Anzahl betroffener Zeilen zurück |
| `db_query` | `db_query(handle, sql)` | Führt SELECT aus, gibt Liste von Row-Maps zurück |
| `q` | `q(handle, sql)` | Abkürzung für `db_query` (pipeline-freundlich) |
| `exec` | `exec(handle, sql)` | Abkürzung für `db_exec` |
| `row_get` | `row_get(row, key)` | Nil-sicherer Feldzugriff aus einer Row-Map |
| `row_eq` | `row_eq(row, key, val)` | Prädikat: row[key] == val (für `filter`) |
| `row_ne` | `row_ne(row, key, val)` | Prädikat: row[key] != val (für `filter`) |

### Typ-Prüfung (6)
`type_of`, `is_num`, `is_str`, `is_list`, `is_map`, `is_nil`

### Konvertierung (2)
`to_str`, `to_num`

### Result-Typ (6)
`Ok`, `Err`, `is_ok`, `is_err`, `unwrap`, `unwrap_or`

### KI — Konfiguration (6)
`ai_provider`, `ai_model`, `ai_set_key`, `ai_host`, `ai_timeout`, `ai_cache`

### KI — Chat (2)
`ai_chat`, `ai_chat_json`

### KI — Streaming (1)
`ai_stream`

### KI — Convenience (7)
`summarize`, `translate`, `classify`, `extract`, `generate`, `generate_json`, `ask`

### KI — Parallel (3)
`ai_batch`, `ai_parallel`, `ai_rate_limit`

### KI — Tool Calling (2)
`ai_tool`, `ai_with_tools`

### KI — Agenten (3)
`agent`, `agent_ask`, `agent_clear`

### KI — Embeddings (5)
`embed`, `embed_batch`, `cosine_sim`, `dot_product`, `nearest`

### KI — Suche (2)
`web_search`, `wiki_search`

### KI — Kosten-Tracking (5)
`ai_cost`, `ai_tokens`, `ai_cache_hits`, `ai_cache_misses`, `try_ai_log`

### Test-Assertions (6)
`assert`, `assert_eq`, `assert_not_eq`, `assert_lt`, `assert_gt`, `assert_error`

### Sandbox (5)
`sandbox_profile`, `set_sandbox`, `with_sandbox`, `audit_log`, `budget_spent`

### MCP (13)
`mcp_server`, `mcp_serve_stdio`, `mcp_serve_sse`, `mcp_tools`, `mcp_resource`,
`mcp_resource_template`, `mcp_prompt`, `mcp_resources`, `mcp_read_resource`,
`mcp_prompts`, `mcp_prompt_get`, `mcp_use_stdio`, `mcp_use_sse`

**Gesamt: 206 Builtins**
