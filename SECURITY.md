# Security Policy

## Attack Surface

Pipe operates as a **full runtime** — it can execute code, access files, make network calls, and invoke AI models.
This power comes with responsibility. Below are all attack vectors and their mitigations.

---

## Builtin Risk Classification

### 🔴 CRITICAL — Arbitrary Execution
| Builtin | Risk | Mitigation |
|---------|------|-----------|
| `exec` | Shell command injection, RCE | `--sandbox` disables it |
| `import` | Code injection from untrusted files | Path validation, `--sandbox` restricts to project dir |

### 🟠 HIGH — Data Exfiltration / Destruction
| Builtin | Risk | Mitigation |
|---------|------|-----------|
| `read_file`, `write_file` | Read/write arbitrary files | `--sandbox` restricts to project dir |
| `file_delete`, `remove_dir` | Destructive file operations | `--sandbox` disables |
| `http_get`, `http_post` | SSRF, data exfiltration | `--sandbox` disables |
| `tcp_connect`, `tcp_write` | Network exfiltration | `--sandbox` disables |

### 🟡 MEDIUM — Information Disclosure
| Builtin | Risk | Mitigation |
|---------|------|-----------|
| `env` | Environment variable leak (API keys, secrets) | `--sandbox` disables |
| `parse_json`, `to_json` | Deserialization of untrusted data | Input validation |
| `ai_*` (all AI builtins) | Data sent to external LLM APIs | `--sandbox` can disable; Ollama for local only |

### 🟢 LOW — Resource Exhaustion
| Pattern | Risk | Mitigation |
|---------|------|-----------|
| `while true` | Infinite loop, CPU exhaustion | `--timeout N` |
| Deep recursion | Stack overflow | `--timeout N` |
| `ai_batch` with huge lists | API cost explosion | Rate limiting |

---

## Defenses

### 1. Sandbox Mode (`--sandbox`)
Restricts Pipe to safe operations only. Disables:
- `exec`, `tcp_*`, `http_*`, `ai_*`
- `file_delete`, `file_move`, `remove_dir`, `make_dir`
- `env`, `input`
- `import` restricts paths to current directory

```bash
./bin/pipe --sandbox script.pipe          # Safe execution
./bin/pipe --sandbox --allow-ai script.pipe  # Allow AI, still block exec/network/fs
```

### 2. Execution Timeout (`--timeout N`)
Kills execution after N seconds. Prevents infinite loops and runaway API costs.

```bash
./bin/pipe --timeout 30 script.pipe       # Max 30 seconds
```

### 3. Safe by Default Principles

**Never run untrusted `.pipe` files.** Pipe code has full access to your system — treat it like a shell script, not like a web page.

**Minimal permissions.** Only enable what you need:
```pipe
# Instead of: exec "rm -rf /tmp/build"
# Use builtins:
remove_dir "/tmp/build"     # Auditable, sandboxable
```

**API key hygiene.** Never embed keys in `.pipe` files. Use environment variables:
```bash
export DEEPSEEK_API_KEY="sk-..."
./bin/pipe script.pipe
```

### 4. Prompt Injection Protection
When user input flows into AI prompts, sanitize or restrict:
```pipe
user_input: input "Enter query: "
# BAD: Direct injection of user text into prompt
# GOOD: Validate/sanitize before passing to AI
```

### 5. Import Validation
Only import from trusted locations. The `--sandbox` flag restricts imports to the project directory.

---

## Reporting a Vulnerability

Report security issues to the repository maintainer directly.
Do NOT open a public issue for security vulnerabilities.

---

## Security Checklist for Pipe Users

- [ ] Use `--sandbox` for untrusted code
- [ ] Set `--timeout N` for AI pipelines (API costs)
- [ ] Never embed API keys in `.pipe` files
- [ ] Validate user input before passing to AI prompts
- [ ] Run with minimal filesystem permissions
- [ ] Keep Pipe updated to latest version
