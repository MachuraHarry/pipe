# RAG pipeline: chunk docs, embed with Ollama, FAISS nearest, answer with DeepSeek.
# Run: python benchmarks/python-vs-pipe/python/rag.py
# Data: benchmarks/python-vs-pipe/data/docs/*.txt

import os
import glob
from pathlib import Path

from langchain_deepseek import ChatDeepSeek
from langchain_ollama import OllamaEmbeddings
from langchain_community.vectorstores import FAISS
from langchain_text_splitters import RecursiveCharacterTextSplitter
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.output_parsers import StrOutputParser

data_dir = Path(__file__).parent.parent / "data" / "docs"

# 1. Load and chunk the knowledge base
raw = []
for f in sorted(data_dir.glob("*.txt")):
    raw.append(f.read_text())

splitter = RecursiveCharacterTextSplitter(chunk_size=300, chunk_overlap=30)
chunks = splitter.split_text("\n\n".join(raw))

# 2. Embed the chunks (Ollama nomic-embed-text, local)
embeddings = OllamaEmbeddings(model="nomic-embed-text")
store = FAISS.from_texts(chunks, embeddings)

# 3. Retrieve the most relevant chunks
question = "How do we rate-limit API requests?"
top = store.similarity_search(question, k=3)
context = "\n---\n".join(d.page_content for d in top)

# 4. Build chain and ask DeepSeek
llm = ChatDeepSeek(model="deepseek-v4-flash", temperature=0.7)
prompt = ChatPromptTemplate.from_template(
    "Context:\n{context}\n\nQuestion: {question}"
)
chain = prompt | llm | StrOutputParser()

print(chain.invoke({"context": context, "question": question}))
