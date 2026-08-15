# AI agent with tool calling: the LLM decides when to call get_weather.
# Run: python benchmarks/python-vs-pipe/python/agent.py
# Set DEEPSEEK_API_KEY first.

from langchain_deepseek import ChatDeepSeek
from langchain.agents import create_agent
from langchain_core.tools import tool

# Define a weather tool
@tool
def get_weather(city: str) -> str:
    """Get current weather for a city."""
    return {
        "Berlin": "22°C, sunny",
        "London": "15°C, rainy",
        "Paris": "25°C, clear",
    }.get(city, f"{city}: No data")

# Wire the agent with the tool
llm = ChatDeepSeek(model="deepseek-v4-flash", temperature=0.7)
agent = create_agent(
    llm,
    [get_weather],
    system_prompt="You are a weather expert.",
)

result = agent.invoke({"messages": [("human", "What's the weather in Berlin, London, and Paris?")]})
print(result["messages"][-1].content)
