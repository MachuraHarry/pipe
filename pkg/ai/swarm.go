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
}

// swarmHandoffTool is a reserved tool name synthesized by ChatSwarm itself for any
// agent that declares HandoffTo targets. It is never present in the caller-supplied
// tool list, so Pipe-level code can never register (or collide with) it.
const swarmHandoffTool = "__handoff__"

// SwarmProgressFunc is an optional observer invoked as ChatSwarm makes progress,
// so a caller can surface a live status (e.g. editing a chat message) without
// waiting for the whole run to finish. event is one of "start" (a round begins
// for agent), "tool" (agent called a real tool named detail), "handoff" (agent
// handed off to the agent named detail), or "final" (agent produced the answer).
// Nil is safe to pass — ChatSwarm treats it as "no observer".
type SwarmProgressFunc func(agent, event, detail string)

// ChatSwarm runs a multi-agent conversation starting at entryAgent. Each round, the
// active agent's own tools (plus a synthetic handoff tool if it declares any
// HandoffTo targets) are offered to the model. A normal tool call is dispatched to
// executor exactly like ChatWithTools. A call to the handoff tool instead swaps the
// active agent for the next round while the full message history — including every
// prior agent's turns — is kept, so the new agent picks up with full context.
func ChatSwarm(entryAgent string, agents map[string]SwarmAgentSpec, userPrompt string,
	executor ToolExecutor, maxRounds int, onProgress SwarmProgressFunc) (SwarmResult, error) {
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
		spec := agents[current]
		tools := swarmToolsFor(spec)

		if onProgress != nil {
			onProgress(current, "start", "")
		}

		resp, err := chatWithToolsRaw(messages, tools)
		if err != nil {
			return SwarmResult{}, fmt.Errorf("swarm round %d (%s): %w", round, current, err)
		}

		if !resp.IsToolCall || len(resp.ToolCalls) == 0 {
			if onProgress != nil {
				onProgress(current, "final", "")
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
					onProgress(current, "handoff", nextAgent)
				}
			} else {
				if onProgress != nil {
					onProgress(current, "tool", tc.Name)
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

		if handoffRequested {
			current = nextAgent
			path = append(path, nextAgent)
			messages[0] = map[string]interface{}{"role": "system", "content": agents[current].SystemPrompt}
		}
	}

	return SwarmResult{}, fmt.Errorf("swarm: max rounds (%d) exceeded without final response", maxRounds)
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
