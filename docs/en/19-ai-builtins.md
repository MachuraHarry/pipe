# 19. AI Builtins

Pipe provides **27 AI builtins** for working with Large Language Models.
Communication happens via REST APIs to OpenAI, Anthropic, or DeepSeek.

---

## 19.1 Configuration

### Choosing a Provider

```pipe
-- OpenAI (default)
ai_provider "openai"
-- Anthropic Claude
ai_provider "anthropic"
-- DeepSeek
ai_provider "deepseek"
```

### Model and Timeout

Each provider uses a cheap & fast default model. Override it with a config
block passed to `ai_provider`, or at any time with `ai_model`:

```pipe
-- Defaults (cheapest & fastest per provider):
--   openai    → gpt-4o-mini
--   anthropic → claude-3-5-haiku-20241022
--   deepseek  → deepseek-v4-flash
--   ollama    → llama3.1:8b

-- Set provider + override model & timeout in one go
ai_provider "deepseek" {model: "deepseek-v4-pro", timeout: 120}

-- Later overrides (apply immediately)
ai_model "deepseek-v4-flash"
ai_host "https://api.deepseek.com"
ai_timeout 60
```

### Thinking & Reasoning Effort (DeepSeek V4)

DeepSeek V4 exposes thinking mode and graded reasoning effort as request
parameters. Pipe maps them onto `ai_provider` config keys `thinking` and
`effort`:

```pipe
-- Thinking mode ON + high effort (V4 default is enabled/high)
ai_provider "deepseek" {model: "deepseek-v4-pro", thinking: true, effort: "high"}

-- Thinking OFF (fast & cheap, no reasoning trace)
ai_provider "deepseek" {model: "deepseek-v4-flash", thinking: false}

-- Max effort for hard agentic tasks
ai_provider "deepseek" {model: "deepseek-v4-pro", effort: "max"}

-- "none" disables thinking entirely (same as thinking: false)
ai_provider "deepseek" {model: "deepseek-v4-pro", effort: "none"}
```

Accepted `effort` values are forwarded verbatim; DeepSeek performs the final
mapping server-side (`low`/`medium` → `high`, `xhigh` → `max` on pro):
`low`, `medium`, `high`, `xhigh`, `max`, `none`.

Notes:

- Both keys are **DeepSeek-only**; using them with another provider returns an
  error. They are safe to combine with `model`/`host`/`timeout` in one block.
- In thinking mode `temperature`, `top_p`, `presence_penalty` and
  `frequency_penalty` have no effect.
- Thinking tokens count towards the completion budget — keep `max_tokens`
  generous (`ai_chat`/`ai_with_tools`) or a request may end with reasoning but
  no final answer.
- Tool calls in thinking mode (`ai_with_tools`) round-trip `reasoning_content`
  automatically; the API requires it and returns a 400 error if omitted.

### API Keys

API keys are set via environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export DEEPSEEK_API_KEY="sk-..."
```

The provider is automatically detected based on which key is set. If the key
is missing, the request fails with an error.

---

## 19.2 Overview Table

| Builtin | Description | Signature |
|---------|-------------|-----------|
| `ai_provider` | Set provider (+ optional model/host/timeout/thinking/effort block) | `ai_provider name {model, host, timeout, thinking, effort}?` |
| `ai_model` | Set model | `ai_model name` |
| `ai_timeout` | Set timeout | `ai_timeout seconds` |
| `ai_host` | Set host URL | `ai_host url` |
| `ai_chat` | Low-level chat | `ai_chat system prompt` |
| `ai_chat_json` | Chat → JSON | `ai_chat_json system prompt` |
| `ai_stream` | Live streaming | `ai_stream system prompt` |
| `ai_parallel` | Parallel calls | `ai_parallel n system items` |
| `ai_batch` | Auto-parallel | `ai_batch system items` |
| `ai_rate_limit` | Rate limiting | `ai_rate_limit calls_per_sec` |
| `embed` | Text → Vector | `embed text` |
| `embed_batch` | Batch embedding | `embed_batch texts` |
| `cosine_sim` | Cosine similarity | `cosine_sim a b` |
| `dot_product` | Dot product | `dot_product a b` |
| `nearest` | Top-K nearest | `nearest query docs k` |
| `ai_tool` | Register tool | `ai_tool name desc schema fn` |
| `ai_with_tools` | Chat with tools | `ai_with_tools system user` |
| `summarize` | Summarize text | `summarize text` |
| `translate` | Translate text | `translate text target_language` |
| `classify` | Classify text | `classify text categories` |
| `extract` | Extract data | `extract text schema` |
| `generate` | Generate free text | `generate prompt` |
| `ask` | Answer question | `ask question` |
| `ai_cost` | Cost metrics | `ai_cost` |
| `ai_tokens` | Total tokens used | `ai_tokens` |
| `ai_cache_hits` | Cache hits | `ai_cache_hits` |
| `ai_cache_misses` | Cache misses | `ai_cache_misses` |

---

## 19.3 High-Level Functions

### summarize

```pipe
-- Signature
summarize text

-- Description
-- Summarizes the given text in 2–3 sentences.

-- Example
text: "Pipe is a modern scripting language focused on..."
print (summarize text)
-- -> Pipe is a scripting language with a pipeline operator and bytecode VM.
--   It supports 168 builtins and is optimized for data processing.
```

### translate

```pipe
-- Signature
translate text target_language

-- Description
-- Translates the text into the specified target language.

-- Example
text: "The quick brown fox jumps over the lazy dog."
print (translate text "German")
-- -> Der schnelle braune Fuchs springt über den faulen Hund.
```

### classify

```pipe
-- Signature
classify text categories

-- Description
-- Classifies the text into one of the given categories.
-- categories can be a string (comma-separated) or a list.

-- Example 1: String
result: classify "Apple announces new iPhone" "Tech, Sports, Politics"
-- -> Tech
print result

-- Example 2: List
categories: ["urgent", "low", "medium"]
print (classify "Server down since 2 hours" categories)
-- -> urgent
```

### extract

```pipe
-- Signature
extract text schema

-- Description
-- Extracts structured data as a map from the text.
-- schema describes the desired fields.

-- Example
text: "John Doe, born on March 15, 1985 in London,
       works as a software engineer and earns $95,000."
schema: { name: "str", birth_year: "num", city: "str", job: "str", salary: "num" }
data: extract text schema
-- -> John Doe
print data.name
-- -> 1985
print data.birth_year
-- -> London
print data.city
-- -> Software Engineer
print data.job
-- -> 95000
print data.salary
```

### generate

```pipe
-- Signature
generate prompt

-- Description
-- Generates free text based on the prompt.

-- Example
prompt: "Write a short product description for an AI tool
         for automated code review."
print (generate prompt)
-- -> Automate your code review process with AI-powered analysis
--   that catches bugs, suggests improvements, and ensures code
--   quality — all before your PR reaches human reviewers.
```

### ask

```pipe
-- Signature
ask question

-- Description
-- Answers a question based on the model's knowledge.

-- Example
print (ask "What is the difference between stack and heap?")
-- -> The stack is a LIFO memory region for local variables
--   and function calls with fast allocation. The heap is
--   a dynamic memory region for long-lived objects with
--   manual or automatic garbage collection.
```

---

## 19.4 Low-Level Functions

### ai_chat

```pipe
-- Signature
ai_chat system_prompt user_prompt [max_tokens]

-- Description
-- Direct chat with a system prompt and a user prompt.
-- Returns the raw model response as a string.
-- The optional third argument limits the response to max_tokens tokens.

-- Example
response: ai_chat
    "You are a Bash expert. Reply only with the finished command."
    "How do I list all files larger than 100 MB recursively?"

print response
-- -> find . -type f -size +100M -exec ls -lh {} \;
```

### ai_chat_json

```pipe
-- Signature
ai_chat_json system_prompt user_prompt [max_tokens]

-- Description
-- Like ai_chat, but the response is parsed as JSON.
-- Returns a map or list depending on the JSON structure.
-- The optional third argument limits the response to max_tokens tokens.

-- Example 1: Map
response: ai_chat_json
    "You are a data analyst. Reply exclusively with JSON."
    "Give the top 3 programming languages for 2025 with percentage share."

-- -> Python
print response.first
-- -> 34
print response.share_py

-- Example 2: List
languages: ai_chat_json
    "Reply as a JSON list of language strings."
    "Name 5 functional programming languages."

for lang in languages
    print lang
```

---

## 19.5 Streaming

Streaming outputs AI responses **token by token in real-time** — the user sees
the response appear like ChatGPT, instead of waiting for the full text.

### ai_stream

```
ai_stream system_prompt user_prompt
```

Streams the response live to stdout and returns the full text as a string.

```pipe
-- Live poem — appears word by word
ai_stream "You are a poet." "Write a short poem about coding."

-- Live translation with post-processing
text: ai_stream "Translate to German." "Hello world, this is streaming."
print ("Translated: " ++ text)
```

**Behavior:**
- Tokens are output to stdout **immediately** (live experience)
- The full response text is returned as the **return value**
- Usable in pipelines: `> upper > print` processes the fully collected text

```pipe
-- Streaming + Pipeline (text appears live, then post-processed)
ai_stream "You are a translator." "Translate: Hello world"
    > upper
    > print
```

---

## 19.6 Parallel Calls

Parallel AI calls process multiple texts simultaneously, dramatically reducing
total execution time.

### ai_batch

```
ai_batch system_prompt items
```

Processes a list of texts in parallel with automatic concurrency (CPU cores × 2).
Returns a list of response strings in the original order.

```pipe
texts: ["Text 1", "Text 2", "Text 3", "Text 4", "Text 5"]
results: ai_batch "Summarize each text in one sentence." texts

for r in results
    print r
```

### ai_parallel

```
ai_parallel concurrency system_prompt items
```

Like `ai_batch` but with explicit concurrency control.

```text
-- Max 3 parallel calls
answers: ai_parallel 3 "Translate to German." (["Hello world",
    "Good morning",
    "How are you?"])
```

### ai_rate_limit

```
ai_rate_limit calls_per_second
```

Limits API calls per second using a global token bucket.

```pipe
-- Max 5 calls per second
ai_rate_limit 5

-- Process 100 texts with rate limiting
texts: read_lines "data.txt"
results: ai_batch "Analyze this text." texts
```

### Performance Comparison

```
3 texts sequential:   14s
3 texts ai_batch:      6s  (2.3× faster)
5 questions, 5 concurrent: 1s  (massively parallel)
```

---

## 19.7 Embeddings & Vector Search

Embeddings convert text into vectors (lists of numbers) that capture the *meaning*
of text in mathematical space. Similar texts are close together. This enables
**semantic search** — understanding meaning, not just keyword matching.

### embed

```
embed text
```

Converts text into an embedding vector (~1536 floats). Uses the `/v1/embeddings`
API (OpenAI-compatible, model: `text-embedding-3-small`).

```pipe
vec: embed "Pipe is a scripting language."
-- 1536
print (len vec)
```

**Note:** Requires `OPENAI_API_KEY`. DeepSeek does not support embeddings.

### embed_batch

```
embed_batch texts
```

Computes embeddings for a list of texts in parallel (4 concurrent).

### cosine_sim

```
cosine_sim vector_a vector_b
```

Cosine similarity between two vectors. Range: -1 (opposite) to 1 (identical).
**No API call** — pure math.

### dot_product

```
dot_product vector_a vector_b
```

Dot product of two vectors. **No API call.**

### nearest

```
nearest query_vector document_vectors k
```

Finds the top-K most similar documents to a query. Returns **indices**.

```pipe
question: embed "How does the VM work?"
top3: nearest question vectors 3

for idx in top3
    print (at documents idx)
```

### RAG with Pipe (complete example)

```pipe
-- 1. Embed knowledge base
documents: read_lines "knowledge.txt"
vectors: embed_batch documents

-- 2. Embed the question
question: "How does the compiler work?"
q_vec: embed question

-- 3. Find relevant documents
top: nearest q_vec vectors 5
context: ""
for idx in top
    context: context ++ (at documents idx) ++ "\n---\n"

-- 4. Answer with context
prompt: "Context:\n" ++ context ++ "\nQuestion: " ++ question
ask prompt > print
```

---

## 19.8 Tool Calling (Function Calling)

Tool Calling lets the LLM invoke Pipe functions as **tools**. Instead of just
responding, the AI can take action — call APIs, run calculations, fetch data.

### ai_tool

```
ai_tool name description parameters_schema function
```

Registers a Pipe function as a tool for the LLM. `parameters_schema` is a map
describing the expected parameters.

```pipe
-- Define a weather tool
fn get_weather city
    match city
        | "Berlin" -> "22°C, sunny"
        | "London" -> "15°C, rainy"
        | _ -> "No data"

-- Register the tool
schema: {city: "Name of the city"}
ai_tool "get_weather" "Get current weather for a city" schema get_weather
```

### ai_with_tools

```
ai_with_tools system_prompt user_prompt
```

Runs a chat with access to all registered tools. The LLM autonomously decides
if and when to call tools.

```pipe
result: ai_with_tools
    "You are a weather assistant. Use get_weather."
    "What's the weather in Berlin and London?"

print result
```

### Execution Flow (internal)

```
1. User: "Weather in Berlin?"
2. LLM → Tool Call: get_weather("Berlin")
3. Pipe executes get_weather → "22°C, sunny"
4. Result sent back to LLM
5. LLM → Response: "It's 22°C and sunny in Berlin."
```

This loop repeats up to 5 rounds until the LLM gives a final answer
without further tool calls.

### Weather Assistant (complete)

```text
fn get_weather city
    match city
        | "Berlin" -> "22°C, sunny"
        | "London" -> "15°C, rainy"
        | "Paris" -> "25°C, clear"
        | _ -> city ++ ": No data"

schema: {city: "Name of the city"}
ai_tool "get_weather" "Get weather for a city" schema get_weather

ai_with_tools "You are a weather expert."
    "What's the weather in Berlin, London, and Paris?"
    > print
```

---

## 19.9 Pipeline with AI

AI operations integrate seamlessly into Pipe pipelines:

```pipe
-- Read text and summarize
read_file "minutes.txt"
    > summarize
    > upper
    > print

-- Translation with post-processing
"Hello world, how are you?"
    > translate "German"
    > split " "
    > len
    > print
-- -> 5 (words in the translated text)

-- Classification in data flow
documents: read_lines "emails.txt"
results: map documents fn doc
    class: classify doc "internal, external, spam, support"
    { doc: doc, type: class }
```

---

## 19.10 Error Handling

AI operations can fail — for example due to missing API keys,
network issues, or timeouts. These errors can be caught with
`try`/`catch`:

```text
-- Missing API key
try
    print (summarize "A long text...")
catch e
    print "AI error: " ++ e.message
    print "Please set your API key!"

-- Catch timeout
-- short timeout
ai_timeout 5
try
    print (generate "Write a 10-page essay about philosophy"))
catch e
    if contains e.message "timeout"
        print "Request took too long"
    else
        throw e

-- Fallback strategy
try
    ai_provider "openai"
    print (ask "What is a monad?"))
catch e
    try
        ai_provider "anthropic"
        print (ask "What is a monad?"))
    catch e2
        print "All providers failed: " ++ e2.message
```

---

## 19.11 Provider Details

| Provider | API Endpoint | Default Model | Environment Variable |
|----------|-------------|---------------|---------------------|
| OpenAI | `https://api.openai.com/v1/chat/completions` | `gpt-4o-mini` | `OPENAI_API_KEY` |
| Anthropic | `https://api.anthropic.com/v1/messages` | `claude-3-5-sonnet-20241022` | `ANTHROPIC_API_KEY` |
| DeepSeek | `https://api.deepseek.com/v1/chat/completions` | `deepseek-v4-pro` | `DEEPSEEK_API_KEY` |
| **Ollama** | `http://localhost:11434/v1/chat/completions` | `llama3.1:8b` | **No key needed!** |

### Ollama — Local AI Without the Cloud

Ollama is a **local LLM server** that runs on your own machine.
Pipe communicates via the OpenAI-compatible API on `localhost:11434`.

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model
ollama pull llama3.2:3b
```

```pipe
ai_provider "ollama"
ai_model "llama3[2]:3b"

ask "What is a pipeline?" > print
```

**Benefits:**
- No API keys, no registration
- No data leaves your system (GDPR/compliance)
- Works completely offline
- Free, unlimited usage
- All 27 AI builtins work with Ollama

**Remote Ollama** (e.g., on a local network):
```pipe
ai_provider "ollama"
ai_host "http://192.168.1[50]:11434"
ai_model "qwen2[5]-coder:7b"
```

---

## 19.12 Cost Tracking & Token Usage

Pipe tracks **cost and token usage** for every AI call. This is essential when
running agents at scale, enforcing budgets, or comparing providers.

### ai_cost

```
ai_cost
```

Returns a map with the cumulative metrics of the current run:

| Key | Type | Description |
|-----|------|-------------|
| `cost_usd` | num | Total cost in USD |
| `calls` | int | Number of AI API calls made |
| `cache_hits` | int | Responses served from the cache |
| `cache_misses` | int | Responses fetched from the provider |

Calling `ai_cost "reset"` zeroes all metrics.

```pipe
print (ai_cost)
-- -> {calls: 1, cost_usd: 0.000045, cache_hits: 0, cache_misses: 1}
```

### ai_tokens

```
ai_tokens
```

Returns the total number of tokens consumed by AI calls in the current run.

### ai_cache_hits / ai_cache_misses

```
ai_cache_hits
ai_cache_misses
```

Return the number of cache hits and misses. Repeated identical prompts are
served from the response cache — saving both money and latency:

```pipe
ask "What is a monad?" > print
-- -> Cache miss
ask "What is a monad?" > print
-- -> Cache hit (same prompt, served from cache)

print (ai_cache_hits)   -- 1
print (ai_cache_misses) -- 1
```

### Cost Trace (CLI)

After a script finishes, the CLI prints a **cost trace** to stderr whenever AI
calls were made:

```
═══ Cost Trace ═══
Total cost:    $0.000045
Total tokens:  50
API calls:     1
Cache hits:    0 | misses: 0
  #1 deepseek/deepseek-chat | 50 tokens | $0.000045
══════════════════════════
```

### Budget Enforcement

AI budgets can be enforced via sandbox profiles — see
[22. Sandbox Profiles](22-sandbox-profiles.md) (the `budget` key and the
`budget_spent` builtin).

### Pricing (estimation fallback)

When a provider does not return a cost, Pipe estimates it from the token counts
using the provider's per-1K-token pricing:

| Provider / Model | Prompt ($ / 1K tokens) | Completion ($ / 1K tokens) |
|------------------|------------------------|----------------------------|
| OpenAI `gpt-4o-mini` | 0.00015 | 0.0006 |
| OpenAI `gpt-4` | 0.03 | 0.06 |
| OpenAI (other) | 0.005 | 0.015 |
| DeepSeek | 0.0009 | 0.0009 |
| Anthropic Claude Haiku | 0.0008 | 0.004 |
| Anthropic Claude Sonnet | 0.003 | 0.015 |
| Anthropic Claude Opus | 0.015 | 0.075 |
| Ollama | 0 | 0 |

---

## 19.13 Example: Complete Workflow

This workflow demonstrates all AI builtins in a practical application:

```text
-- ai_demo.pipe — AI workflow with all builtins
-- Usage: OPENAI_API_KEY="sk-..." pipe ai_demo.pipe

ai_provider "openai"
ai_model "gpt-4o-mini"
ai_timeout 90

print "=== 1. Generate text ==="
description: generate "Describe the Rust programming language in 2 sentences"
print description

print "\n=== 2. Summarize ==="
summary: summarize description
print summary

print "\n=== 3. Translate ==="
translation: translate description "German"
print translation

print "\n=== 4. Classify ==="
category: classify description "Programming Languages, Hardware, Networking, Databases"
print "Category: " ++ category

print "\n=== 5. Extract data ==="
resume: "
  John Doe, Senior Developer at TechCorp in London.
  He has 12 years of experience with Python, Go, and Rust.
  His email is john@techcorp.com and he earns $95,000 per year."
schema: { name: "str", company: "str", years_experience: "num",
           languages: "list", email: "str", salary: "num" }
data: extract resume schema
print "Name:         " ++ data.name
print "Company:      " ++ data.company
print "Experience:   " ++ (to_str data.years_experience) ++ " years"
print "Languages:    " ++ (join data.languages ", ")
print "Email:        " ++ data.email
print "Salary:       $" ++ (to_str data.salary)

print "\n=== 6. Answer question ==="
question: "Which of these languages — " ++ (join data.languages ", ") ++
          " — is best suited for systems programming and why?"
answer: ask question
print answer

print "\n=== 7. Low-level chat ==="
analysis: ai_chat
    "You are a career coach. Analyze the profile and give 3 recommendations.
     Reply with numbered bullet points."
    "Profile: " ++ resume

print analysis

print "\n=== 8. Structured JSON response ==="
recommendations: ai_chat_json
    "Reply as a JSON object with fields: strengths (list),
     improvements (list), next_career_step (string),
     salary_forecast (number)."
    "Analyze: " ++ resume

print "Strengths:"
for s in recommendations.strengths
    print "  - " ++ s

print "Areas for improvement:"
for i in recommendations.improvements
    print "  - " ++ i

print "Next step: " ++ recommendations.next_career_step
print "Salary forecast: $" ++ (to_str recommendations.salary_forecast)

print "\n=== 9. Error handling in workflow ==="
try
    ai_provider "deepseek"
    print (ask "What is TDD?"))
catch e
    print "DeepSeek unavailable: " ++ e.message
    print "Falling back to OpenAI..."
    ai_provider "openai"
    print (ask "What is TDD?"))

print "\n=== Workflow complete ==="
```
