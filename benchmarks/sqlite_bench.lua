-- SQLite Benchmark: Lua (luasql)
local luasql = require("luasql.sqlite3")
local env = luasql.sqlite3()
local conn = env:connect(":memory:")

local N = 5000
local function now()
  return os.clock() * 1000
end

local function bench(label, fn)
  local start = now()
  local result = fn()
  local elapsed = now() - start
  print(string.format("%s: %.2f ms", label, elapsed))
  return result
end

print(string.format("Lua SQLite Benchmark (%d rows)", N))
print("=================================")

-- CREATE TABLE
bench("create", function()
  conn:execute("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER, city TEXT)")
end)

-- INSERT N rows
local inserted = bench("insert", function()
  for i = 0, N - 1 do
    local name = "user_" .. i
    conn:execute(string.format(
      "INSERT INTO users (name, age, city) VALUES ('%s', %d, '%s')",
      name, 20 + (i % 50), "city_" .. (i % 10)
    ))
  end
  return N
end)
print("Inserted: " .. inserted)

-- SELECT *
local n_all = bench("select *", function()
  local cur = conn:execute("SELECT * FROM users")
  local n = 0
  while cur:fetch() do n = n + 1 end
  cur:close()
  return n
end)
print("Rows returned: " .. n_all)

-- SELECT with WHERE
local n_filtered = bench("select where", function()
  local cur = conn:execute("SELECT * FROM users WHERE age > 40")
  local n = 0
  while cur:fetch() do n = n + 1 end
  cur:close()
  return n
end)
print("Filtered rows: " .. n_filtered)

-- SELECT with GROUP BY
local n_grouped = bench("select group", function()
  local cur = conn:execute("SELECT city, COUNT(*) as cnt FROM users GROUP BY city")
  local n = 0
  while cur:fetch() do n = n + 1 end
  cur:close()
  return n
end)
print("Groups: " .. n_grouped)

print("=================================")
conn:close()
env:close()
