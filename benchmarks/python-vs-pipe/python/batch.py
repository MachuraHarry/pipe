# Batch processing: classify 10 article titles in parallel (asyncio + semaphore).
# Run: python benchmarks/python-vs-pipe/python/batch.py
# Set DEEPSEEK_API_KEY first.

import asyncio
import json
import time
from pathlib import Path

from langchain_deepseek import ChatDeepSeek

data_dir = Path(__file__).parent.parent / "data"

articles = json.loads((data_dir / "articles.json").read_text())
titles = [a["title"] for a in articles]

llm = ChatDeepSeek(model="deepseek-v4-flash", temperature=0.7)


async def worker(sem, title):
    async with sem:
        return (await llm.ainvoke(
            f"Classify the topic of this article title. Answer with one word.\n{title}"
        )).content


async def main():
    sem = asyncio.Semaphore(10)
    t = time.monotonic()
    results = await asyncio.gather(*(worker(sem, t) for t in titles))
    print(f"Fertig in {time.monotonic() - t:.0f}s für {len(results)} Artikel")
    for r in results:
        print(r)


asyncio.run(main())
