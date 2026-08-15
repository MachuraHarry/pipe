# CPU microbenchmark: integer loop sum 0..1_000_000
s = 0
for i in range(1_000_000):
    s += i
s
