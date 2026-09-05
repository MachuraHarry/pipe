package object

import (
	"sync"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

// swarmAgentDef is the Pipe-level registration for one swarm member: its own
// system prompt, the names of already-registered ai_tool tools it may use, and
// the names of other swarm members it may hand the conversation off to. Tool and
// handoff names are resolved lazily (at ai_swarm call time, not at swarm_agent
// registration time) so registration order never matters.
type swarmAgentDef struct {
	SystemPrompt string
	ToolNames    []string
	HandoffTo    []string
}

var (
	swarmRegistry   = map[string]swarmAgentDef{}
	swarmRegistryMu sync.Mutex
)

func bSwarmAgent(args ...Object) Object {
	if len(args) != 2 {
		return err("swarm_agent expects 2 arguments (name, config)")
	}
	name, ok := args[0].(*String)
	if !ok {
		return err("swarm_agent: name must be a string")
	}
	config, ok := args[1].(*Map)
	if !ok {
		return err("swarm_agent: config must be a block/map")
	}

	def := swarmAgentDef{}
	for _, entry := range config.Pairs {
		key, val := entry.Key, entry.Value
		switch key {
		case "system":
			s, ok := val.(*String)
			if !ok {
				return err("swarm_agent: system must be a string")
			}
			def.SystemPrompt = s.Value

		case "tools":
			lst, ok := val.(*List)
			if !ok {
				return err("swarm_agent: tools must be a list of tool names (strings)")
			}
			def.ToolNames = make([]string, 0, len(lst.Elements))
			for _, e := range lst.Elements {
				s, ok := e.(*String)
				if !ok {
					return err("swarm_agent: tools must be a list of tool names (strings)")
				}
				def.ToolNames = append(def.ToolNames, s.Value)
			}

		case "handoff":
			lst, ok := val.(*List)
			if !ok {
				return err("swarm_agent: handoff must be a list of agent names (strings)")
			}
			def.HandoffTo = make([]string, 0, len(lst.Elements))
			for _, e := range lst.Elements {
				s, ok := e.(*String)
				if !ok {
					return err("swarm_agent: handoff must be a list of agent names (strings)")
				}
				def.HandoffTo = append(def.HandoffTo, s.Value)
			}

		default:
			return err("swarm_agent: unknown config key '" + key + "'. Use system, tools, or handoff")
		}
	}

	if def.SystemPrompt == "" {
		return err("swarm_agent: config must include 'system' (string)")
	}

	swarmRegistryMu.Lock()
	swarmRegistry[name.Value] = def
	swarmRegistryMu.Unlock()

	return TRUE
}

// buildSwarmAgents snapshots the swarm registry into the shape ai.ChatSwarm
// expects, resolving each agent's tool names against the ai_tool registry
// (toolRegistry, pkg/object/builtins_ai.go). A tool name that isn't registered
// yet is a hard error here rather than at swarm_agent time, since registration
// order between ai_tool and swarm_agent calls is not guaranteed.
func buildSwarmAgents() (map[string]ai.SwarmAgentSpec, *Error) {
	swarmRegistryMu.Lock()
	defer swarmRegistryMu.Unlock()

	agents := make(map[string]ai.SwarmAgentSpec, len(swarmRegistry))
	for name, def := range swarmRegistry {
		spec := ai.SwarmAgentSpec{SystemPrompt: def.SystemPrompt, HandoffTo: def.HandoffTo}
		for _, toolName := range def.ToolNames {
			entry, exists := toolRegistry[toolName]
			if !exists {
				return nil, err("swarm_agent: tool '" + toolName + "' is not registered. Call ai_tool before ai_swarm.")
			}
			spec.Tools = append(spec.Tools, entry.Def)
		}
		agents[name] = spec
	}
	return agents, nil
}

// runSwarm implements the shared argument parsing, sandbox gate, and ai.ChatSwarm
// dispatch for ai_swarm, ai_swarm_trace, and ai_swarm_stream, which differ only
// in how they shape the successful result for Pipe code (and whether a 4th
// progress-callback argument is present/required).
func runSwarm(builtinName string, args ...Object) (ai.SwarmResult, Object) {
	if len(args) < 2 {
		return ai.SwarmResult{}, err(builtinName + " expects at least 2 arguments (task, entry_agent, [max_rounds], [on_progress])")
	}
	task, ok := args[0].(*String)
	if !ok {
		return ai.SwarmResult{}, err(builtinName + ": task must be a string")
	}
	entry, ok := args[1].(*String)
	if !ok {
		return ai.SwarmResult{}, err(builtinName + ": entry_agent must be a string")
	}

	maxRounds := 5
	if len(args) >= 3 {
		n, ok := ToInt(args[2])
		if !ok {
			return ai.SwarmResult{}, err(builtinName + ": max_rounds must be a number")
		}
		maxRounds = int(n)
	}

	// A 4th argument, when present, is a Pipe closure invoked as
	// cb(agent, event, detail, args_json, round, max_rounds) after every
	// swarm step — see SwarmProgressFunc (round/max_rounds: the round
	// safety-budget counters, NOT a task-completion estimate). Bridging
	// back into Pipe from Go uses CallUserFunction exactly like
	// map/filter's callback dispatch (builtins_collections.go).
	var onProgress ai.SwarmProgressFunc
	if len(args) >= 4 {
		cb := args[3]
		onProgress = func(agent, event, detail, argsJSON string, round, maxRounds int) {
			CallUserFunction(cb, &String{Value: agent}, &String{Value: event}, &String{Value: detail}, &String{Value: argsJSON}, &Integer{Value: int64(round)}, &Integer{Value: int64(maxRounds)})
		}
	}

	// A 5th argument, when present, is a Pipe closure invoked with no
	// arguments at the start of every round — see ai.SwarmRoundCheck. It is
	// expected to return a map with any of "abort" (bool), "abort_reason"
	// (string), "inject" (string) — every field optional, missing/wrong-typed
	// fields default to their zero value. Returning nil, a non-map, or an
	// empty map is a fully inert round (identical to no closure at all).
	// Bridging back into Pipe uses CallUserFunction exactly like onProgress
	// above; the *Map-then-.Get-per-key pattern mirrors writeResponse in
	// builtins_http_server.go.
	var roundCheck ai.SwarmRoundCheck
	if len(args) >= 5 {
		cb := args[4]
		roundCheck = func() ai.SwarmRoundAction {
			result := CallUserFunction(cb)
			m, ok := result.(*Map)
			if !ok {
				return ai.SwarmRoundAction{}
			}
			action := ai.SwarmRoundAction{}
			if v, ok := m.Get("abort"); ok {
				if b, ok := v.(*Boolean); ok {
					action.Abort = b.Value
				}
			}
			if v, ok := m.Get("abort_reason"); ok {
				if s, ok := v.(*String); ok {
					action.AbortReason = s.Value
				}
			}
			if v, ok := m.Get("inject"); ok {
				if s, ok := v.(*String); ok {
					action.Inject = s.Value
				}
			}
			return action
		}
	}

	// Same two-branch gate as ai_chat: profile.CanAI() under a registered
	// profile, CLI --sandbox flag otherwise. The authoritative enforcement is
	// ai.ChatSwarm's own gateEgress call (mirroring ai.ChatWithTools), but this
	// early check avoids building the agent map and executor for a call that
	// is going to be rejected anyway.
	if ActiveProfile.Load().Name != "none" {
		if canErr := ActiveProfile.Load().CanAI(); canErr != nil {
			return ai.SwarmResult{}, err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowAI {
		return ai.SwarmResult{}, sandboxBlock(builtinName + " (AI calls)")
	}

	agents, buildErr := buildSwarmAgents()
	if buildErr != nil {
		return ai.SwarmResult{}, buildErr
	}
	if _, exists := agents[entry.Value]; !exists {
		return ai.SwarmResult{}, err(builtinName + ": unknown entry agent '" + entry.Value + "'. Register it with swarm_agent first.")
	}

	profile := ActiveProfile.Load()
	executor := func(toolName string, targs map[string]interface{}) (string, error) {
		return executeTool(profile, toolName, targs)
	}
	batchExecutor := func(calls []ai.ToolCallRequest) []ai.ToolCallResult {
		return runToolBatch(profile, calls)
	}

	result, swarmErr := ai.ChatSwarm(entry.Value, agents, task.Value, executor, maxRounds, onProgress, batchExecutor, roundCheck)
	if swarmErr != nil {
		return ai.SwarmResult{}, err(builtinName + ": " + swarmErr.Error())
	}
	return result, nil
}

func bAiSwarm(args ...Object) Object {
	result, errObj := runSwarm("ai_swarm", args...)
	if errObj != nil {
		return errObj
	}
	return &String{Value: result.Content}
}

func bAiSwarmTrace(args ...Object) Object {
	result, errObj := runSwarm("ai_swarm_trace", args...)
	if errObj != nil {
		return errObj
	}
	pathElems := make([]Object, len(result.Path))
	for i, p := range result.Path {
		pathElems[i] = &String{Value: p}
	}
	return MapFromGo(map[string]Object{
		"content": &String{Value: result.Content},
		"path":    &List{Elements: pathElems},
		"rounds":  &Integer{Value: int64(result.Rounds)},
	})
}

// bAiSwarmStream is ai_swarm_trace plus a mandatory 4th argument: a Pipe
// closure called as cb(agent, event, detail, args_json) after every swarm
// step (event is one of "start"/"tool"/"handoff"/"final"/"reasoning"/
// "inject" — see ai.SwarmProgressFunc), so callers can surface live progress
// (e.g. editing a chat message) instead of waiting for the whole run to
// finish. An optional 5th argument is a Pipe closure called with no
// arguments at the start of every round (see ai.SwarmRoundCheck) — it may
// return a map with any of "abort" (bool), "abort_reason" (string), "inject"
// (string), every field optional. A truthy "abort" stops the run early; the
// result then has "aborted"=true and "abort_reason" set, with "content"
// reflecting whatever partial progress was made (often empty). A non-empty
// "inject" is appended to the live conversation as a new instruction before
// the round proceeds, letting a caller steer an in-flight run. Otherwise
// identical success shape to ai_swarm_trace.
func bAiSwarmStream(args ...Object) Object {
	if len(args) < 4 {
		return err("ai_swarm_stream expects 4 arguments (task, entry_agent, max_rounds, on_progress)")
	}
	result, errObj := runSwarm("ai_swarm_stream", args...)
	if errObj != nil {
		return errObj
	}
	pathElems := make([]Object, len(result.Path))
	for i, p := range result.Path {
		pathElems[i] = &String{Value: p}
	}
	return MapFromGo(map[string]Object{
		"content":      &String{Value: result.Content},
		"path":         &List{Elements: pathElems},
		"rounds":       &Integer{Value: int64(result.Rounds)},
		"aborted":      &Boolean{Value: result.Aborted},
		"abort_reason": &String{Value: result.AbortReason},
	})
}
