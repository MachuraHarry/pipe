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
	for key, val := range config.Pairs {
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
// dispatch for both ai_swarm and ai_swarm_trace, which differ only in how they
// shape the successful result for Pipe code.
func runSwarm(builtinName string, args ...Object) (ai.SwarmResult, Object) {
	if len(args) < 2 {
		return ai.SwarmResult{}, err(builtinName + " expects at least 2 arguments (task, entry_agent, [max_rounds])")
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

	result, swarmErr := ai.ChatSwarm(entry.Value, agents, task.Value, executor, maxRounds)
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
	return &Map{Pairs: map[string]Object{
		"content": &String{Value: result.Content},
		"path":    &List{Elements: pathElems},
		"rounds":  &Integer{Value: int64(result.Rounds)},
	}}
}
