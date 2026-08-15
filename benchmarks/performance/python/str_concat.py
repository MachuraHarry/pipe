# CPU microbenchmark: string concatenation 20_000 × "ab"
s = ""
for _ in range(20_000):
    s += "ab"
len(s)
