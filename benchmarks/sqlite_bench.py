# SQLite Benchmark: Python (sqlite3)
import sqlite3
import time

N = 5000
conn = sqlite3.connect(":memory:")

def now():
    return time.time() * 1000

def bench(label, fn):
    start = now()
    result = fn()
    elapsed = now() - start
    print(f"{label}: {elapsed:.2f} ms")
    return result

print(f"Python SQLite Benchmark ({N} rows)")
print("=================================")

# CREATE TABLE
bench("create", lambda: conn.execute(
    "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER, city TEXT)"
))

# INSERT N rows
def do_insert():
    for i in range(N):
        conn.execute(
            "INSERT INTO users (name, age, city) VALUES (?, ?, ?)",
            (f"user_{i}", 20 + (i % 50), f"city_{i % 10}")
        )
    return N

inserted = bench("insert", do_insert)
print(f"Inserted: {inserted}")

# SELECT *
def do_select_all():
    cur = conn.execute("SELECT * FROM users")
    rows = cur.fetchall()
    cur.close()
    return len(rows)

n_all = bench("select *", do_select_all)
print(f"Rows returned: {n_all}")

# SELECT with WHERE
def do_select_where():
    cur = conn.execute("SELECT * FROM users WHERE age > 40")
    rows = cur.fetchall()
    cur.close()
    return len(rows)

n_filtered = bench("select where", do_select_where)
print(f"Filtered rows: {n_filtered}")

# SELECT with GROUP BY
def do_select_group():
    cur = conn.execute("SELECT city, COUNT(*) as cnt FROM users GROUP BY city")
    rows = cur.fetchall()
    cur.close()
    return len(rows)

n_grouped = bench("select group", do_select_group)
print(f"Groups: {n_grouped}")

print("=================================")
conn.close()
