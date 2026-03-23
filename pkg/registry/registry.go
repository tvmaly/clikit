package registry

import (
	"fmt"
	"sort"
	"sync"

	"github.com/tvmaly/clikit/pkg/cli"
)

// Registry holds the set of registered tools.
type Registry struct {
	mu   sync.RWMutex
	cmds map[string]*cli.Command
}

// New creates an empty Registry.
func New() *Registry {
	return &Registry{cmds: make(map[string]*cli.Command)}
}

// Register adds cmd to the registry. Panics if a command with the same name
// is already registered (use RegisterErr for safe registration).
func (r *Registry) Register(cmd *cli.Command) {
	if err := r.RegisterErr(cmd); err != nil {
		panic(err)
	}
}

// RegisterErr adds cmd to the registry, returning an error on duplicate.
func (r *Registry) RegisterErr(cmd *cli.Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.cmds[cmd.Name]; exists {
		return fmt.Errorf("tool %q already registered", cmd.Name)
	}
	r.cmds[cmd.Name] = cmd
	return nil
}

// Lookup returns the command registered under name, if any.
func (r *Registry) Lookup(name string) (*cli.Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.cmds[name]
	return cmd, ok
}

// Commands returns all registered commands, sorted by name.
func (r *Registry) Commands() []*cli.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*cli.Command, 0, len(r.cmds))
	for _, cmd := range r.cmds {
		result = append(result, cmd)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
