package ai

import (
	"encoding/json"
	"fmt"
)

// SwarmAgentSpec describes one member of a swarm: its own system prompt, its own
// tool set, and the names of the other swarm members it may hand the conversation
// off to.
type SwarmAgentSpec struct {
	SystemPrompt string
	Tools        []ToolDef
	HandoffTo    []string
}

// SwarmResult is the outcome of a ChatSwarm run: the final agent's answer plus an
// observability trail of which agents actually handled the conversation.
type SwarmResult struct {
	Content string
	Path    []string
	Rounds  int
	// Aborted is true when abortCheck (see ChatSwarm) requested an early
	// stop. Content/Path/Rounds reflect whatever progress was made before
	// the abort — Content may be empty if it happened before any agent
	// produced text. An abort is a deliberate, caller-requested stop, not
	// a failure: it is reported via this field, never via the error return.
	Aborted bool
	// AbortReason is whatever string abortCheck returned when it requested
	// the stop (e.g. "cancelled by /stop"). Empty unless Aborted is true.
	AbortReason string
}

// swarmHandoffTool is a reserved tool name synthesized by ChatSwarm itself for any
// agent that declares HandoffTo targets. It is never present in the caller-supplied
// tool list, so Pipe-level code can never register (or collide with) it.
const swarmHandoffTool = "__handoff__"

// SwarmProgressFunc is an optional observer invoked as ChatSwarm makes progress,
// so a caller can surface a live status (e.g. editing a chat message) without
// waiting for the whole run to finish. event is one of "start" (a round begins
// for agent), "tool" (agent called a real tool named detail), "handoff" (agent
// handed off to the agent named detail), "final" (agent produced the answer),
// "reasoning" (the model's chain-of-thought for this round, in detail — only
// fired when the provider actually returns one), or "inject" (a SwarmRoundCheck
// steered the run with the interjection text in detail). argsJSON carries the
// tool call's raw JSON arguments for a "tool" event (so a caller can show e.g.
// the actual search query, not just the tool name); it is empty for every other
// event. Nil is safe to pass — ChatSwarm treats it as "no observer".
type SwarmProgressFunc func(agent, event, detail, argsJSON string)

// SwarmRoundAction is what a SwarmRoundCheck returns to control the next
// round. Abort/AbortReason stop ChatSwarm immediately, exactly like the
// former SwarmAbortCheck. Inject, when non-empty, is appended to the
// conversation as a new user-role message before the round proceeds — lets
// a caller steer an in-flight swarm with a fresh instruction without
// restarting it. A zero-value SwarmRoundAction (the common case) means
// "nothing to do this round".
type SwarmRoundAction struct {
	Abort       bool
	AbortReason string
	Inject      string
}

// SwarmRoundCheck is called at the START of every round, before the LLM is
// asked for anything. A nil roundCheck (the common case) disables this
// entirely; ChatSwarm then behaves exactly as it always did. This is
// deliberately the ONLY hook point: it runs synchronously on the same
// goroutine/VM as the rest of ChatSwarm, no concurrency of any kind is
// introduced by it (see the design note at its Pipe-level caller,
// muninn.pipe's make_round_check, for why that matters here).
type SwarmRoundCheck func() SwarmRoundAction

// ChatSwarm runs a multi-agent conversation starting at entryAgent. Each round, the
// active agent's own tools (plus a synthetic handoff tool if it declares any
// HandoffTo targets) are offered to the model. A normal tool call is dispatched to
// executor exactly like ChatWithTools. A call to the handoff tool instead swaps the
// active agent for the next round while the full message history — including every
// prior agent's turns — is kept, so the new agent picks up with full context.
func ChatSwarm(entryAgent string, agents map[string]SwarmAgentSpec, userPrompt string,
	executor ToolExecutor, maxRounds int, onProgress SwarmProgressFunc, batchExecutor ToolBatchExecutor,
	roundCheck SwarmRoundCheck) (SwarmResult, error) {
	if err := gateEgress(EgressChat, ActiveConfig.APIHost); err != nil {
		return SwarmResult{}, err
	}
	if maxRounds <= 0 {
		maxRounds = 5
	}

	entry, ok := agents[entryAgent]
	if !ok {
		return SwarmResult{}, fmt.Errorf("swarm: unknown entry agent %q", entryAgent)
	}

	current := entryAgent
	path := []string{entryAgent}
	messages := []map[string]interface{}{
		{"role": "system", "content": entry.SystemPrompt},
		{"role": "user", "content": userPrompt},
	}

	for round := 0; round < maxRounds; round++ {
		if roundCheck != nil {
			action := roundCheck()
			if action.Abort {
				return SwarmResult{Content: "", Path: path, Rounds: round, Aborted: true, AbortReason: action.AbortReason}, nil
			}
			if action.Inject != "" {
				messages = append(messages, map[string]interface{}{
					"role":    "user",
					"content": "[User interjection] " + action.Inject,
				})
				if onProgress != nil {
					onProgress(current, "inject", action.Inject, "")
				}
			}
		}
		spec := agents[current]
		tools := swarmToolsFor(spec)

		// Two agents can get stuck bouncing a conversation back and forth
		// indefinitely — e.g. a "critic" repeatedly sending work back to a
		// "verifier" that has nothing new to add, and the critic judging it
		// incomplete again each time — burning every round without ever
		// reaching a final answer. Detected as an exact A,B,A,B tail in path:
		// withdraw the handoff back to that SPECIFIC partner for this round
		// (see oscillationTools) so the loop with THAT agent breaks, while
		// every other handoff target — crucially any finalizing agent, or an
		// unrelated specialist — stays available. Earlier this stripped ALL
		// tools down to spec.Tools, which for an agent with no tools of its
		// own (e.g. a pure-handoff "critic") left it with literally nothing
		// to call: live-reproduced on a genuine multi-round research task
		// (critic <-> fact-checker legitimately iterating three times to go
		// deep on a topic, not a stuck loop) — the critic couldn't even reach
		// its own finalizing agent and answered with a hollow "will pass this
		// on for review" non-answer instead of the real report.
		if isOscillating(path) {
			tools = oscillationTools(spec, path)
		}

		if onProgress != nil {
			onProgress(current, "start", "", "")
		}

		// Without a warning, a swarm that quietly runs out of rounds mid-task
		// (e.g. a long multi-tool-call research-and-write job) returns NO
		// answer at all — the hard error below discards everything done so
		// far. Once fewer than ~10% of rounds remain, append a one-off
		// reminder to THIS round's request only (never stored in messages,
		// so it does not compound every remaining round) telling the model
		// to wrap up now with whatever it already has instead of continuing
		// to explore. A visibly incomplete-but-real answer beats none.
		//
		// Floor of 3, not 1: a handoff itself consumes a round, so wrapping
		// up can still take several more hops (e.g. critic -> finalizer,
		// then the finalizer's own round to actually answer). Live-
		// reproduced with a deliberately tiny max_rounds=8: threshold=1 let
		// the warning arrive on literally the last round, a "critic" agent
		// dutifully handed off immediately as asked — and then the loop
		// ended before the target agent it handed off to ever got a turn,
		// so max-rounds was exceeded anyway despite the model doing exactly
		// what the warning asked. 3 gives at least one real hand-off-and-
		// respond chain room to finish inside the warning window.
		reqMessages := messages
		roundsLeft := maxRounds - round
		warnThreshold := maxRounds / 10
		if warnThreshold < 3 {
			warnThreshold = 3
		}
		if roundsLeft <= warnThreshold {
			reqMessages = append(append([]map[string]interface{}{}, messages...), map[string]interface{}{
				"role":    "user",
				"content": fmt.Sprintf("[SYSTEM] Only %d of %d rounds remain before this conversation is cut off with NO answer delivered at all. Wrap up NOW: stop exploring or gathering more information and use whatever you already have to produce a usable result THIS round — answer directly, or use your handoff tool to hand off immediately if you are not the one who finalizes. A complete-enough answer now is far better than no answer.", roundsLeft, maxRounds),
			})
		}

		resp, err := chatWithToolsRaw(reqMessages, tools)
		if err != nil {
			return SwarmResult{}, fmt.Errorf("swarm round %d (%s): %w", round, current, err)
		}

		if onProgress != nil && resp.ReasoningContent != "" {
			onProgress(current, "reasoning", resp.ReasoningContent, "")
		}

		if !resp.IsToolCall || len(resp.ToolCalls) == 0 {
			// A plain-text reply from an agent that still had a handoff or a
			// tool of its own on offer THIS round is not a real answer — it
			// is a non-terminal agent skipping its job (never calling its
			// actual tool, or never passing the conversation on) and just
			// narrating instead. Silently accepting that as final ends the
			// whole swarm right there, discarding whatever chain (e.g. a
			// finalizing agent's own verification/formatting step) was
			// supposed to happen next. Live-reproduced: a document
			// specialist, at the end of a long research handoff, replied
			// with a nicely-worded prose summary instead of ever calling
			// its create/write-document tool — no file was ever produced,
			// and the swarm ended immediately since a text reply always
			// used to terminate the round loop regardless of which agent
			// gave it. A terminal agent (no HandoffTo declared, e.g. a
			// "registrator") is exempt — that is its designed way to end.
			// An agent that genuinely has nothing left to call this round
			// (e.g. oscillation-breaking stripped its only handoff and it
			// has no tools of its own) is also exempt — forcing a retry
			// there would just be demanding the impossible.
			if len(spec.HandoffTo) > 0 && len(tools) > 0 {
				messages = append(messages, map[string]interface{}{"role": "assistant", "content": resp.Content})
				messages = append(messages, map[string]interface{}{
					"role":    "user",
					"content": "[SYSTEM] You did not call a tool. You are not the final agent in this chain — you must either use one of your own tools to make real progress, or use your handoff tool to pass the conversation on. A plain text reply from you is not accepted; try again now.",
				})
				continue
			}
			if onProgress != nil {
				onProgress(current, "final", "", "")
			}
			return SwarmResult{Content: resp.Content, Path: path, Rounds: round + 1}, nil
		}

		assistantMsg := map[string]interface{}{"role": "assistant"}
		if resp.ReasoningContent != "" {
			assistantMsg["reasoning_content"] = resp.ReasoningContent
		}
		var openaiToolCalls []map[string]interface{}
		for _, tc := range resp.ToolCalls {
			openaiToolCalls = append(openaiToolCalls, map[string]interface{}{
				"id":   tc.ID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      tc.Name,
					"arguments": tc.Arguments,
				},
			})
		}
		assistantMsg["tool_calls"] = openaiToolCalls
		messages = append(messages, assistantMsg)

		nextAgent := ""
		handoffRequested := false

		realCount := 0
		for _, tc := range resp.ToolCalls {
			if tc.Name != swarmHandoffTool {
				realCount++
			}
		}

		if batchExecutor != nil && realCount >= 2 {
			// Concurrent batch path: run every non-handoff call from this
			// round together instead of one at a time (only taken when
			// there are at least 2 — the dominant single-real-call round
			// always takes the untouched sequential path in the else
			// branch below, byte-for-byte as before). A handoff call
			// sharing the round is still processed synchronously right
			// here — handleSwarmHandoff mutates nextAgent/handoffRequested
			// via pointers, it's cheap and local, no reason to parallelize
			// it, and doing so would race those two variables.
			results := make([]string, len(resp.ToolCalls))
			callArgs := make([]map[string]interface{}, len(resp.ToolCalls))
			for i, tc := range resp.ToolCalls {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
					args = map[string]interface{}{"raw": tc.Arguments}
				}
				callArgs[i] = args
			}

			var realIdx []int
			for i, tc := range resp.ToolCalls {
				if tc.Name != swarmHandoffTool {
					realIdx = append(realIdx, i)
				}
			}
			batchReqs := make([]ToolCallRequest, len(realIdx))
			for j, i := range realIdx {
				tc := resp.ToolCalls[i]
				if onProgress != nil {
					onProgress(current, "tool", tc.Name, tc.Arguments)
				}
				batchReqs[j] = ToolCallRequest{Name: tc.Name, Args: callArgs[i]}
			}
			batchResults := batchExecutor(batchReqs)
			for j, i := range realIdx {
				if j < len(batchResults) {
					r := batchResults[j]
					if r.Err != nil {
						results[i] = "Error: " + r.Err.Error()
					} else {
						results[i] = r.Content
					}
				} else {
					results[i] = "Error: batch executor returned fewer results than requested calls"
				}
			}
			for i, tc := range resp.ToolCalls {
				if tc.Name == swarmHandoffTool {
					before := nextAgent
					results[i] = handleSwarmHandoff(spec, agents, callArgs[i], &nextAgent, &handoffRequested)
					if onProgress != nil && nextAgent != before && handoffRequested {
						onProgress(current, "handoff", nextAgent, "")
					}
				}
			}
			// Appended in the model's original tool_calls order — every
			// tool-role message must line up with its tool_call_id
			// regardless of which path (batched or synchronous handoff)
			// produced its content.
			for i, tc := range resp.ToolCalls {
				messages = append(messages, map[string]interface{}{
					"role":         "tool",
					"tool_call_id": tc.ID,
					"content":      results[i],
				})
			}
		} else {
			for _, tc := range resp.ToolCalls {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
					args = map[string]interface{}{"raw": tc.Arguments}
				}

				var content string
				if tc.Name == swarmHandoffTool {
					before := nextAgent
					content = handleSwarmHandoff(spec, agents, args, &nextAgent, &handoffRequested)
					if onProgress != nil && nextAgent != before && handoffRequested {
						onProgress(current, "handoff", nextAgent, "")
					}
				} else {
					if onProgress != nil {
						onProgress(current, "tool", tc.Name, tc.Arguments)
					}
					result, execErr := executor(tc.Name, args)
					if execErr != nil {
						content = "Error: " + execErr.Error()
					} else {
						content = result
					}
				}

				messages = append(messages, map[string]interface{}{
					"role":         "tool",
					"tool_call_id": tc.ID,
					"content":      content,
				})
			}
		}

		if handoffRequested {
			current = nextAgent
			path = append(path, nextAgent)
			messages[0] = map[string]interface{}{"role": "system", "content": agents[current].SystemPrompt}
		}
	}

	return SwarmResult{}, fmt.Errorf("swarm: max rounds (%d) exceeded without final response", maxRounds)
}

// isOscillating reports whether the last four entries of path are an exact
// A,B,A,B alternation between two distinct agents — i.e. two full round
// trips between the same pair with no other agent entering in between. One
// back-and-forth (A,B,A) is normal (e.g. a critic sending work back for one
// more pass); a second full repeat is a strong, cheap signal of a stuck
// loop rather than genuine progress.
func isOscillating(path []string) bool {
	n := len(path)
	if n < 4 {
		return false
	}
	a, b := path[n-1], path[n-2]
	return a != b && path[n-3] == a && path[n-4] == b
}

// oscillationTools builds the tool list for a round where isOscillating has
// fired. Only the handoff to the SPECIFIC partner the agent has been
// bouncing with is withdrawn — every other handoff target stays available,
// so a legitimate multi-round back-and-forth can still resolve via a
// different path (e.g. a critic that can no longer send work back to the
// fact-checker it was oscillating with can still hand off to a finalizing
// agent, instead of being left with literally no tool to call).
func oscillationTools(spec SwarmAgentSpec, path []string) []ToolDef {
	partner := path[len(path)-2]
	restricted := spec
	restricted.HandoffTo = nil
	for _, h := range spec.HandoffTo {
		if h != partner {
			restricted.HandoffTo = append(restricted.HandoffTo, h)
		}
	}
	return swarmToolsFor(restricted)
}

// swarmToolsFor builds the tool list offered to the model for one round: the
// agent's own tools plus a synthetic handoff tool restricted to its declared
// targets, or just its own tools if it declares no handoff targets.
func swarmToolsFor(spec SwarmAgentSpec) []ToolDef {
	if len(spec.HandoffTo) == 0 {
		return spec.Tools
	}
	handoff := ToolDef{
		Name:        swarmHandoffTool,
		Description: "Transfer this conversation to another agent better suited to handle it.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"to": map[string]interface{}{
					"type":        "string",
					"description": "Name of the agent to transfer to.",
					"enum":        spec.HandoffTo,
				},
			},
			"required": []string{"to"},
		},
	}
	return append(append([]ToolDef{}, spec.Tools...), handoff)
}

// handleSwarmHandoff validates a requested handoff target against the current
// agent's declared allowlist and the registered agent set. On success it sets
// *nextAgent and *handoffRequested and returns an acknowledgement string for the
// tool-result message; on failure it leaves the active agent unchanged and returns
// an error string the model can see and react to.
func handleSwarmHandoff(spec SwarmAgentSpec, agents map[string]SwarmAgentSpec, args map[string]interface{}, nextAgent *string, handoffRequested *bool) string {
	to, ok := args["to"].(string)
	if !ok || to == "" {
		return "Error: handoff requires a 'to' agent name"
	}
	allowed := false
	for _, t := range spec.HandoffTo {
		if t == to {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Sprintf("Error: handoff to %q is not permitted from this agent", to)
	}
	if _, exists := agents[to]; !exists {
		return fmt.Sprintf("Error: agent %q is not registered", to)
	}
	if *handoffRequested {
		return fmt.Sprintf("Error: a handoff to %q was already requested this round, ignoring", to)
	}
	*nextAgent = to
	*handoffRequested = true
	return "Transferred to " + to
}
