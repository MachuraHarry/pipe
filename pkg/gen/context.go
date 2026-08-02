package gen

type context struct {
	scopes    []map[string]ScopeEntry
	names     *nameGen
	loopDepth int
	fnDepth   int
}

func newContext(seed int64) *context {
	return &context{
		names: newNameGen(seed),
	}
}

func (c *context) pushScope() {
	c.scopes = append(c.scopes, make(map[string]ScopeEntry))
}

func (c *context) popScope() {
	if len(c.scopes) > 0 {
		c.scopes = c.scopes[:len(c.scopes)-1]
	}
}

func (c *context) define(name string, entry ScopeEntry) {
	n := len(c.scopes)
	if n == 0 {
		c.pushScope()
	}
	c.scopes[n-1][name] = entry
}

func (c *context) resolve(name string) (ScopeEntry, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if e, ok := c.scopes[i][name]; ok {
			return e, true
		}
	}
	return ScopeEntry{}, false
}

func (c *context) addBuiltins() {
	for name, arity := range builtinArities {
		c.define(name, ScopeEntry{Kind: KindBuiltin, Type: TypeAny, Arity: arity})
	}
}
