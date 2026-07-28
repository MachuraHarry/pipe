# 16. Kochrezepte

Praktische Code-Beispiele für häufige Aufgaben.

## 16.1 Fibonacci-Zahlen

```pipe
fn fib n
    match n
        | 0  -> 0
        | 1  -> 1
        | _  -> fib(n - 1) + fib(n - 2)

print (fib 10)    -- 55
```

## 16.2 FizzBuzz

```pipe
fn fizzbuzz n
    if n % 15 == 0
        "FizzBuzz"
    else if n % 3 == 0
        "Fizz"
    else if n % 5 == 0
        "Buzz"
    else
        n

for n in (range 1 16)
    print (fizzbuzz n)
```

## 16.3 Primzahlen finden

```pipe
fn is_prime n
    is_prime_helper n 2

fn is_prime_helper n d
    if d * d > n
        true
    else if n % d == 0
        false
    else
        is_prime_helper n (d + 1)

fn primes up_to
    filter (range 2 (up_to + 1)) is_prime

print "Primzahlen bis 30:"
print (primes 30)
```

## 16.4 Fakultät

```pipe
fn fakultaet n
    if n <= 1
        1
    else
        n * (fakultaet (n - 1))

print (fakultaet 5)     -- 120
print (fakultaet 10)    -- 3628800
```

## 16.5 Palindrom-Prüfer

```pipe
fn reverse_str s
    reverse_helper s (len s - 1)

fn reverse_helper s i
    if i < 0
        ""
    else
        (at s i) ++ (reverse_helper s (i - 1))

fn is_palindrome s
    s == (reverse_str s)

print (is_palindrome "racecar")     -- true
print (is_palindrome "hello")       -- false
print (is_palindrome "anna")        -- true
```

## 16.6 Celsius ↔ Fahrenheit

```pipe
fn celsius_to_fahrenheit c
    c * 9 / 5 + 32

fn fahrenheit_to_celsius f
    (f - 32) * 5 / 9

print "0°C = " ++ (to_str (celsius_to_fahrenheit 0)) ++ "°F"
print "100°C = " ++ (to_str (celsius_to_fahrenheit 100)) ++ "°F"
print "32°F = " ++ (to_str (fahrenheit_to_celsius 32)) ++ "°C"
print "212°F = " ++ (to_str (fahrenheit_to_celsius 212)) ++ "°C"
```

## 16.7 Caesar-Verschlüsselung

```pipe
fn shift_char ch offset
    idx: find_char "abcdefghijklmnopqrstuvwxyz" ch 0
    new_idx: (idx + offset) % 26
    at "abcdefghijklmnopqrstuvwxyz" new_idx

fn find_char alphabet ch pos
    if pos >= (len alphabet)
        pos
    else if (at alphabet pos) == ch
        pos
    else
        find_char alphabet ch (pos + 1)

fn caesar text offset
    caesar_helper text offset 0 ""

fn caesar_helper text offset pos result
    if pos >= (len text)
        result
    else
        c: at text pos
        shifted: shift_char c offset
        caesar_helper text offset (pos + 1) (result ++ shifted)

print "Caesar('hallo', 3):"
print (caesar "hallo" 3)     -- "kdoor"
```

## 16.8 Listen-Summe

```pipe
fn summe liste
    total: 0
    for n in liste
        total: total + n
    total

print (summe [1, 2, 3, 4, 5])    -- 15
```

## 16.9 Text-Statistik

```pipe
fn analyze text
    words: split text " "
    print "Text: " ++ text
    print "Wortanzahl: " ++ (to_str (len words))
    print "In GROSS: " ++ (upper text)
    print "Enthält 'Welt'? " ++ (to_str (contains text "Welt"))

analyze "Hallo Welt von Pipe"
```

## 16.10 Rechner mit match

```pipe
fn calc a op b
    match op
        | "+" -> a + b
        | "-" -> a - b
        | "*" -> a * b
        | "/" -> a / b
        | "%" -> a % b
        | _   -> 0

print (calc 10 "+" 5)     -- 15
print (calc 10 "-" 5)     -- 5
print (calc 10 "*" 5)     -- 50
print (calc 10 "/" 5)     -- 2
print (calc 10 "%" 3)     -- 1
```

## 16.11 HTTP-Client

```pipe
-- Einfacher GET-Request
antwort: http_get "https://httpbin.org/get"
print "Status: " ++ (to_str (get antwort "status"))

-- JSON-API aufrufen
daten: http_get_json "https://api.github.com/users/torvalds"
print "Name: " ++ (get daten "name")
print "Repos: " ++ (to_str (get daten "public_repos"))

-- HTTP POST
payload: to_json {name: "Pipe", typ: "sprache"}
post_resp: http_post "https://httpbin.org/post" payload
print "POST Status: " ++ (to_str (get post_resp "status"))
```

## 16.12 Datei-Operationen

```pipe
-- Schreiben
write_file "/tmp/demo.txt" "Hallo Welt\nZeile 2"

-- Lesen
inhalt: read_file "/tmp/demo.txt"
print inhalt

-- Zeilenweise
zeilen: read_lines "/tmp/demo.txt"
print "Anzahl Zeilen: " ++ (to_str (len zeilen))

-- Prüfungen
print "Existiert? " ++ (to_str (file_exists "/tmp/demo.txt")))
print "Typ: " ++ (file_type "/tmp/demo.txt")
print "Größe: " ++ (to_str (file_size "/tmp/demo.txt")) ++ " Bytes"

-- Anhängen
append_file "/tmp/demo.txt" "\nAngehängte Zeile"

-- Verzeichnisinhalt
dateien: list_dir "/tmp"
print "Dateien in /tmp: " ++ (to_str dateien)

-- Aufräumen
file_delete "/tmp/demo.txt"
```

## 16.13 Verzeichnis-Operationen

```pipe
-- Verzeichnis anlegen
make_dir "/tmp/mein_projekt/unterordner"
print "Existiert? " ++ (to_str (file_exists "/tmp/mein_projekt")))

-- Pfad-Operationen
p: "/home/user/docs/report.txt"
print "Dateiname:   " ++ (path_base p)
print "Verzeichnis: " ++ (path_dir p)
print "Endung:      " ++ (path_ext p)
print "Join:        " ++ (path_join "/tmp" "sub" "file.txt")

-- Kopieren und Umbenennen
file_copy "/tmp/demo.txt" "/tmp/kopie.txt"
file_move "/tmp/kopie.txt" "/tmp/umbenannt.txt"

-- Aufräumen
remove_dir "/tmp/mein_projekt"
```

## 16.14 TCP Echo-Server

```pipe
-- Server
fn run_server
    print "Echo-Server auf :9999..."
    ln: tcp_listen "0.0.0.0" 9999
    conn: tcp_accept ln
    msg: tcp_read conn
    print "Empfangen: " ++ msg
    tcp_write conn ("ECHO: " ++ msg)
    tcp_close conn
    tcp_close ln

-- Client
fn run_client message
    print "Verbinde zu :9999..."
    conn: tcp_connect "127.0.0.1" 9999
    tcp_write conn message
    reply: tcp_read conn
    print "Antwort: " ++ reply
    tcp_close conn
```

## 16.15 JSON Konfiguration

```pipe
-- JSON schreiben
config: {
    name: "MeineApp",
    version: "1.0.0",
    debug: false,
    ports: [8080, 8443]
}
write_file "config.json" (to_json config)

-- JSON lesen
raw: read_file "config.json"
config: parse_json raw
print "App: " ++ (get config "name")
print "Version: " ++ (get config "version")
```

## 16.16 Regex-Validierung

```pipe
fn is_valid_email email
    regex_match "^\\S+@\\S+\\.\\S+$" email

fn mask_phone phone
    regex_replace "[0-9]" "#" phone

print "Email gültig? " ++ (to_str (is_valid_email "test@example.com")))
print "Tel maskiert: " ++ (mask_phone "Tel: 0123-456789"))
```

## 16.17 Datum und Zeit

```pipe
ts: now 0
print "Unix:    " ++ (to_str ts)
print "Datum:   " ++ (format_time ts "2006-01-02")
print "Zeit:    " ++ (format_time ts "15:04:05")
print "Wochentag: " ++ (format_time ts "Monday")
```

## 16.18 Zufallszahlen

```pipe
print "Würfel (1-6):"
for i in (range 1 4)
    print (to_str (random_range 1 7))

print "Zufall 0-1: " ++ (to_str (random 0)))
```

## 16.19 Shell-Befehle ausführen

```pipe
-- Befehl ausführen
ergebnis: exec "ls -la"
print (get ergebnis "output")

-- Exit-Code prüfen
status: get ergebnis "status"
if status == 0
    print "Erfolgreich"
else
    print "Fehler: " ++ (get ergebnis "error")

-- Umgebungsvariable lesen
home: env "HOME"
print "HOME = " ++ home

-- Warten
print "Warte 1 Sekunde..."
sleep 1000
print "Fertig"
```

## 16.20 Fehlerbehandlung in der Praxis

```pipe
fn sichere_division a b
    if b == 0
        Err "Division durch Null"
    else
        Ok (a / b)

fn verarbeite_division a b
    ergebnis: sichere_division a b
    if is_ok ergebnis
        print "Ergebnis: " ++ (to_str (unwrap ergebnis))
    else
        print "Fehler!"

verarbeite_division 10 2     -- "Ergebnis: 5"
verarbeite_division 10 0     -- "Fehler!"
```

## 16.21 Closures (Funktionsfabrik)

```pipe
fn make_multiplier faktor
    fn multiplier x
        x * faktor

verdoppler: make_multiplier 2
verdreifacher: make_multiplier 3

print (verdoppler 10)       -- 20
print (verdreifacher 10)    -- 30
```

## 16.22 Binäre Suche

```pipe
fn binary_search target list low high
    if low > high
        -1
    else
        mid: (low + high) / 2
        wert: at list mid
        if wert == target
            mid
        else if target < wert
            binary_search target list low (mid - 1)
        else
            binary_search target list (mid + 1) high

zahlen: [1, 3, 5, 7, 9, 11, 13, 15]
print "Position von 7: " ++ (to_str (binary_search 7 zahlen 0 7)))     -- 3
print "Position von 10: " ++ (to_str (binary_search 10 zahlen 0 7)))   -- -1
```

## 16.23 Enum + Match

```pipe
enum Status: Aktiv, Inaktiv, Gelöscht

fn status_text s
    match s
        | Aktiv     -> "Aktiv"
        | Inaktiv   -> "Inaktiv"
        | Gelöscht  -> "Gelöscht"
        | _         -> "Unbekannt"

print "Status 0: " ++ (status_text Aktiv))     -- "Status 0: Aktiv"
print "Status 1: " ++ (status_text Inaktiv))   -- "Status 1: Inaktiv"
```

## 16.24 Import & Modul-System

**math.pipe:**
```pipe
export fn quadrat x
    x * x

export fn kubik x
    x * (quadrat x)

fn intern x          -- Nicht exportiert
    x + 1
```

**main.pipe:**
```pipe
import "math.pipe"

print "7² = " ++ (to_str (quadrat 7)))     -- 49
print "3³ = " ++ (to_str (kubik 3)))       -- 27
```

## 16.25 Defer für Cleanup

```pipe
fn verarbeite_datei pfad
    print "Öffne " ++ pfad
    defer print "Schließe " ++ pfad
    inhalt: read_file pfad
    print "Inhalt: " ++ inhalt
    print "Verarbeitung abgeschlossen"

write_file "/tmp/test.txt" "Test-Daten"
verarbeite_datei "/tmp/test.txt"
file_delete "/tmp/test.txt"
-- Ausgabe:
--   Öffne /tmp/test.txt
--   Inhalt: Test-Daten
--   Verarbeitung abgeschlossen
--   Schließe /tmp/test.txt     ← defer!
```
