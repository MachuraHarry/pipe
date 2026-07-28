# 19. AI Builtins

Pipe provides **12 AI builtins** for working with Large Language Models.
Communication happens via REST APIs to OpenAI, Anthropic, or DeepSeek.

---

## 19.1 Configuration

### Choosing a Provider

```pipe
ai_provider "openai"       -- OpenAI (default)
ai_provider "anthropic"    -- Anthropic Claude
ai_provider "deepseek"     -- DeepSeek
```

### Model and Timeout

```pipe
ai_model "gpt-4o"           -- Set model (default: gpt-4o-mini)
ai_timeout 120               -- Timeout in seconds (default: 60)
```

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
| `ai_provider` | Set provider | `ai_provider name` |
| `ai_model` | Set model | `ai_model name` |
| `ai_timeout` | Set timeout | `ai_timeout seconds` |
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
-- → Pipe is a scripting language with a pipeline operator and bytecode VM.
--   It supports 80+ builtins and is optimized for data processing.
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
-- → Der schnelle braune Fuchs springt über den faulen Hund.
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
print result     -- → Tech

-- Example 2: List
categories: ["urgent", "low", "medium"]
print (classify "Server down since 2 hours" categories)
-- → urgent
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
print data.name        -- → John Doe
print data.birth_year  -- → 1985
print data.city        -- → London
print data.job         -- → Software Engineer
print data.salary      -- → 95000
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
-- → Automate your code review process with AI-powered analysis
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
-- → The stack is a LIFO memory region for local variables
--   and function calls with fast allocation. The heap is
--   a dynamic memory region for long-lived objects with
--   manual or automatic garbage collection.
```

---

## 19.4 Low-Level Functions

### ai_chat

```pipe
-- Signature
ai_chat system_prompt user_prompt

-- Description
-- Direct chat with a system prompt and a user prompt.
-- Returns the raw model response as a string.

-- Example
response: ai_chat
    "You are a Bash expert. Reply only with the finished command."
    "How do I list all files larger than 100 MB recursively?"

print response
-- → find . -type f -size +100M -exec ls -lh {} \;
```

### ai_chat_json

```pipe
-- Signature
ai_chat_json system_prompt user_prompt

-- Description
-- Like ai_chat, but the response is parsed as JSON.
-- Returns a map or list depending on the JSON structure.

-- Example 1: Map
response: ai_chat_json
    "You are a data analyst. Reply exclusively with JSON."
    "Give the top 3 programming languages for 2025 with percentage share."

print response.first       -- → Python
print response.share_py    -- → 34

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

```pipe
-- Max 3 parallel calls
answers: ai_parallel 3 "Translate to German." [
    "Hello world",
    "Good morning",
    "How are you?"
]
```

### ai_rate_limit

```
ai_rate_limit calls_per_second
```

Limits API calls per second using a global token bucket.

```pipe
ai_rate_limit 5       -- Max 5 calls per second

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
print (len vec)    -- 1536
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

```pipe
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
-- → 5 (words in the translated text)

-- Classification in data flow
documents: read_lines "emails.txt"
results: map documents fn(doc)
    class: classify doc "internal, external, spam, support"
    { doc: doc, type: class }
```

---

## 19.10 Error Handling

AI operations can fail — for example due to missing API keys,
network issues, or timeouts. These errors can be caught with
`try`/`catch`:

```pipe
-- Missing API key
try
    print (summarize "A long text...")
catch e
    print "AI error: " ++ e.message
    print "Please set your API key!"

-- Catch timeout
ai_timeout 5     -- short timeout
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
| Anthropic | `https://api.anthropic.com/v1/messages` | `claude-3-haiku-20240307` | `ANTHROPIC_API_KEY` |
| DeepSeek | `https://api.deepseek.com/v1/chat/completions` | `deepseek-chat` | `DEEPSEEK_API_KEY` |

The OpenAI-compatible API interface allows connecting additional compatible
services via the `OPENAI_API_KEY` environment variable and `OPENAI_API_BASE`
for custom endpoints.

---

## 19.12 Example: Complete Workflow

This workflow demonstrates all AI builtins in a practical application:

```pipe
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
