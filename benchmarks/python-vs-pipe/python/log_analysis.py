# Log analysis → incident report: filter errors, summarize, translate, save.
# Run: python benchmarks/python-vs-pipe/python/log_analysis.py
# Set DEEPSEEK_API_KEY first.

from pathlib import Path

from langchain_deepseek import ChatDeepSeek

data_dir = Path(__file__).parent.parent / "data"

# 1. Read the log and keep only ERROR lines
logs = (data_dir / "incident.log").read_text().splitlines()
errors = [line for line in logs if "ERROR" in line]

# 2. Build a compact input
joined = "\n".join(errors)

# 3. Summarize the incident in English
llm = ChatDeepSeek(model="deepseek-v4-flash", temperature=0.7)
summary = llm.invoke(f"Summarize this incident in 2-3 sentences:\n{joined}").content

# 4. Translate to German
german = llm.invoke(
    f"Translate the following text to German:\n{summary}"
).content

# 5. Save the report
report = f"# Incident Report\n\n## English\n{summary}\n\n## Deutsch\n{german}\n"
(data_dir / "incident_report.md").write_text(report)
print(german)
