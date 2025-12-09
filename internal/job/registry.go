package job

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

// Registry errors
var (
	ErrHandlerNotFound    = errors.New("handler not found")
	ErrHandlerExists      = errors.New("handler already registered")
)

// Registry manages job handlers
type Registry struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// NewRegistry creates a new job handler registry
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

// Register adds a handler for a job type
func (r *Registry) Register(jobType string, handler Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[jobType]; exists {
		return ErrHandlerExists
	}

	r.handlers[jobType] = handler
	return nil
}

// RegisterFunc adds a handler function for a job type
func (r *Registry) RegisterFunc(jobType string, fn func(context.Context, *Job) error) error {
	return r.Register(jobType, simpleHandler{fn: fn})
}

// simpleHandler wraps a simple function as a Handler
type simpleHandler struct {
	fn func(context.Context, *Job) error
}

func (h simpleHandler) Handle(ctx context.Context, job *Job) (json.RawMessage, error) {
	err := h.fn(ctx, job)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(`{"success":true}`), nil
}

// MustRegister adds a handler and panics on error
func (r *Registry) MustRegister(jobType string, handler Handler) {
	if err := r.Register(jobType, handler); err != nil {
		panic(err)
	}
}

// Get retrieves a handler for a job type
func (r *Registry) Get(jobType string) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[jobType]
	if !exists {
		return nil, ErrHandlerNotFound
	}

	return handler, nil
}

// Has checks if a handler exists for a job type
func (r *Registry) Has(jobType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.handlers[jobType]
	return exists
}

// Types returns all registered job types
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// Unregister removes a handler for a job type
func (r *Registry) Unregister(jobType string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.handlers, jobType)
}
