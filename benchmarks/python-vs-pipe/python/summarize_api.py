# AI Summarize API: FastAPI + uvicorn, POST /summarize returns summary.
# Run: export DEEPSEEK_API_KEY=... && python python/summarize_api.py
# Test: curl -X POST http://localhost:8787/summarize -d '{"text":"..."}'

from langchain_deepseek import ChatDeepSeek
from fastapi import FastAPI, Request
import uvicorn

llm = ChatDeepSeek(model="deepseek-v4-flash", temperature=0.7)
app = FastAPI()


@app.post("/summarize")
async def summarize(req: Request):
    body = await req.json()
    result = (await llm.ainvoke(f"Summarize: {body['text']}")).content
    return {"result": result}


if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=8788)
