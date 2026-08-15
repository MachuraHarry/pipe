# CPU microbenchmark: list push 20_000 + sum
lst = []
for i in range(20_000):
    lst.append(i)
total = sum(lst)
total
