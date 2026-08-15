# 3 LLM calls: sequential baseline vs asyncio.gather parallel.
# Run: python benchmarks/python-vs-pipe/python/parallel.py
# Set DEEPSEEK_API_KEY first.

import asyncio
import time

from langchain_deepseek import ChatDeepSeek

llm = ChatDeepSeek(model="deepseek-v4-flash", temperature=0.7)

questions = [
    "Löse 7*8+4 und antworte nur mit der Zahl.",
    "Löse 12*12 und antworte nur mit der Zahl.",
    "Löse 100/4 und antworte nur mit der Zahl.",
]

# Sequential baseline
t = time.monotonic()
for q in questions:
    print("Frage:", llm.invoke(q).content)
seq = time.monotonic() - t

# Parallel via asyncio.gather
t = time.monotonic()
async def run():
    return await asyncio.gather(*(llm.ainvoke(q) for q in questions))

answers = asyncio.run(run())
par = time.monotonic() - t

for a in answers:
    print("Frage:", a.content)
print(f"Sequenziell: {seq:.2f}s | Parallel: {par:.2f}s")
