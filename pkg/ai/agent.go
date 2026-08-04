package ai

import (
	"sync"
)

type Agent struct {
	Name         string
	SystemPrompt string
	History      []Message
	mu           sync.Mutex
}

var (
	agentRegistry   = map[string]*Agent{}
	agentRegistryMu sync.RWMutex
)

func CreateAgent(name, systemPrompt string) *Agent {
	agentRegistryMu.Lock()
	defer agentRegistryMu.Unlock()

	a := &Agent{
		Name:         name,
		SystemPrompt: systemPrompt,
		History:      []Message{},
	}
	agentRegistry[name] = a
	return a
}

func GetAgent(name string) (*Agent, bool) {
	agentRegistryMu.RLock()
	defer agentRegistryMu.RUnlock()
	a, ok := agentRegistry[name]
	return a, ok
}

func DeleteAgent(name string) bool {
	agentRegistryMu.Lock()
	defer agentRegistryMu.Unlock()
	_, ok := agentRegistry[name]
	if ok {
		delete(agentRegistry, name)
	}
	return ok
}

func (a *Agent) Ask(userMessage string) (string, error) {
	a.mu.Lock()
	a.History = append(a.History, Message{Role: "user", Content: userMessage})
	historyLen := len(a.History)
	a.mu.Unlock()

	messages := []Message{
		{Role: "system", Content: a.SystemPrompt},
	}
	messages = append(messages, a.History...)

	req := ChatRequest{Messages: messages}
	resp, err := Chat(req)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	// Only append if history hasn't been cleared concurrently
	if len(a.History) == historyLen {
		a.History = append(a.History, Message{Role: "assistant", Content: resp.Content})
	}
	a.mu.Unlock()

	return resp.Content, nil
}

func (a *Agent) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.History = []Message{}
}
