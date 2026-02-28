package engine

import "fmt"

type Registry struct {
	engines map[string]Engine
}

func NewRegistry() *Registry {
	return &Registry{engines: make(map[string]Engine)}
}

func (r *Registry) Register(e Engine) {
	r.engines[e.Name()] = e
}

func (r *Registry) Get(name string) (Engine, error) {
	e, ok := r.engines[name]
	if !ok {
		return nil, fmt.Errorf("unknown engine: %s", name)
	}
	return e, nil
}

func (r *Registry) Enabled(names []string) []Engine {
	var engines []Engine
	for _, name := range names {
		if e, ok := r.engines[name]; ok {
			engines = append(engines, e)
		}
	}
	return engines
}

func (r *Registry) All() []Engine {
	engines := make([]Engine, 0, len(r.engines))
	for _, e := range r.engines {
		engines = append(engines, e)
	}
	return engines
}
