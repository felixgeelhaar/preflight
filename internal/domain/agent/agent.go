// Package agent provides the background agent domain for scheduled reconciliation.
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felixgeelhaar/statekit"
)

// State represents the agent's current state.
type State string

const (
	// StateStopped indicates the agent is not running.
	StateStopped State = "stopped"
	// StateStarting indicates the agent is initializing.
	StateStarting State = "starting"
	// StateRunning indicates the agent is active and waiting for next reconciliation.
	StateRunning State = "running"
	// StateReconciling indicates the agent is performing a reconciliation cycle.
	StateReconciling State = "reconciling"
	// StateStopping indicates the agent is shutting down.
	StateStopping State = "stopping"
	// StateError indicates the agent encountered an error.
	StateError State = "error"
)

// Event types for the agent state machine.
const (
	EventStart             = "START"
	EventStop              = "STOP"
	EventTick              = "TICK"
	EventReconcileComplete = "RECONCILE_COMPLETE"
	EventError             = "ERROR"
	EventRecover           = "RECOVER"
	EventStarted           = "STARTED"
)

// Context holds the runtime context for the agent state machine.
// This is used by statekit as the context type.
type Context struct {
	// Configuration
	Config *Config

	// Runtime state
	StartedAt       time.Time
	LastReconcileAt time.Time
	ReconcileCount  int
	ErrorCount      int
	LastError       error

	// Current reconciliation result
	LastResult *ReconciliationResult

	// Health status
	Health HealthStatus
}

// RuntimeContext wraps Context with thread-safe access.
type RuntimeContext struct {
	mu  sync.RWMutex
	ctx Context
}

// NewRuntimeContext creates a new runtime context with the given configuration.
func NewRuntimeContext(cfg *Config) *RuntimeContext {
	return &RuntimeContext{
		ctx: Context{
			Config: cfg,
			Health: HealthStatus{
				Status: HealthUnknown,
			},
		},
	}
}

// RecordStart records the agent start time.
func (c *RuntimeContext) RecordStart() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx.StartedAt = time.Now()
	c.ctx.Health.Status = HealthHealthy
	c.ctx.Health.LastCheck = time.Now()
}

// RecordReconciliation records a reconciliation cycle.
func (c *RuntimeContext) RecordReconciliation(result *ReconciliationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx.LastReconcileAt = time.Now()
	c.ctx.ReconcileCount++
	c.ctx.LastResult = result
	c.ctx.Health.LastCheck = time.Now()
}

// RecordError records an error occurrence.
func (c *RuntimeContext) RecordError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx.ErrorCount++
	c.ctx.LastError = err
	c.ctx.Health.Status = HealthDegraded
	c.ctx.Health.LastCheck = time.Now()
	c.ctx.Health.Message = err.Error()
}

// GetStatus returns a snapshot of the current status.
func (c *RuntimeContext) GetStatus() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Status{
		StartedAt:       c.ctx.StartedAt,
		LastReconcileAt: c.ctx.LastReconcileAt,
		ReconcileCount:  c.ctx.ReconcileCount,
		ErrorCount:      c.ctx.ErrorCount,
		LastError:       c.ctx.LastError,
		Health:          c.ctx.Health,
	}
}

// GetContext returns a copy of the current context.
func (c *RuntimeContext) GetContext() Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

// Status represents a snapshot of the agent's status.
type Status struct {
	State           State          `json:"state"`
	StartedAt       time.Time      `json:"started_at,omitempty"`
	LastReconcileAt time.Time      `json:"last_reconcile_at,omitempty"`
	ReconcileCount  int            `json:"reconcile_count"`
	ErrorCount      int            `json:"error_count"`
	LastError       error          `json:"last_error,omitempty"`
	Health          HealthStatus   `json:"health"`
	NextReconcileAt time.Time      `json:"next_reconcile_at,omitempty"`
	Uptime          time.Duration  `json:"uptime,omitempty"`
	PendingApproval string         `json:"pending_approval,omitempty"`
	DriftCount      map[string]int `json:"drift_count,omitempty"`
}

// Agent represents the background agent with state machine.
type Agent struct {
	interp  *statekit.Interpreter[Context]
	runtime *RuntimeContext

	// Callbacks
	onReconcile   func(ctx context.Context) (*ReconciliationResult, error)
	onStateChange func(from, to State)

	// Control
	stopCh    chan struct{}
	stoppedCh chan struct{}
	mu        sync.RWMutex
}

// NewAgent creates a new background agent with the given configuration.
func NewAgent(cfg *Config) (*Agent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	runtime := NewRuntimeContext(cfg)

	return &Agent{
		runtime:   runtime,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}, nil
}

// buildAgentMachine constructs the agent state machine using statekit.
// The runtime pointer is captured by closures to ensure actions modify the original context.
func buildAgentMachine(runtime *RuntimeContext) (*statekit.Interpreter[Context], error) {
	machine, err := statekit.NewMachine[Context]("preflight-agent").
		WithInitial("stopped").
		WithContext(runtime.GetContext()).
		WithAction("recordStart", func(_ *Context, _ statekit.Event) {
			// Use captured runtime pointer to modify the original context
			runtime.RecordStart()
		}).
		WithAction("recordError", func(_ *Context, event statekit.Event) {
			// Use captured runtime pointer to modify the original context
			if payload, ok := event.Payload.(map[string]interface{}); ok {
				if err, ok := payload["error"].(error); ok {
					runtime.RecordError(err)
				}
			}
		}).
		// Stopped state
		State("stopped").
		On(EventStart).Target("starting").Done().
		// Starting state
		State("starting").
		OnEntry("recordStart").
		On(EventStarted).Target("running").
		On(EventError).Target("error").Done().
		// Running state (waiting for next reconciliation)
		State("running").
		On(EventTick).Target("reconciling").
		On(EventStop).Target("stopping").
		On(EventError).Target("error").Done().
		// Reconciling state
		State("reconciling").
		On(EventReconcileComplete).Target("running").
		On(EventStop).Target("stopping").
		On(EventError).Target("error").Done().
		// Stopping state
		State("stopping").
		After(100 * time.Millisecond).Target("stopped").Done().
		// Error state
		State("error").
		OnEntry("recordError").
		On(EventRecover).Target("running").
		On(EventStop).Target("stopped").Done().
		Build()

	if err != nil {
		return nil, err
	}

	return statekit.NewInterpreter(machine), nil
}

// SetReconcileHandler sets the function to call during reconciliation.
func (a *Agent) SetReconcileHandler(fn func(ctx context.Context) (*ReconciliationResult, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onReconcile = fn
}

// SetStateChangeHandler sets the callback for state changes.
func (a *Agent) SetStateChangeHandler(fn func(from, to State)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onStateChange = fn
}

// Start starts the agent.
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Build the state machine
	interp, err := buildAgentMachine(a.runtime)
	if err != nil {
		return fmt.Errorf("failed to build state machine: %w", err)
	}
	a.interp = interp

	// Reset control channels
	a.stopCh = make(chan struct{})
	a.stoppedCh = make(chan struct{})

	// Start the interpreter
	a.interp.Start()

	// Send START event to begin initialization
	a.interp.Send(statekit.Event{Type: EventStart})

	// Small delay for state transition, then send STARTED
	time.AfterFunc(50*time.Millisecond, func() {
		a.mu.RLock()
		interp := a.interp
		a.mu.RUnlock()
		if interp != nil {
			interp.Send(statekit.Event{Type: EventStarted})
		}
	})

	// Start the scheduler goroutine
	go a.runScheduler(ctx)

	return nil
}

// Stop stops the agent gracefully.
func (a *Agent) Stop(ctx context.Context) error {
	a.mu.Lock()
	interp := a.interp
	stopCh := a.stopCh
	stoppedCh := a.stoppedCh

	if interp == nil {
		a.mu.Unlock()
		return nil
	}

	// Signal stop
	select {
	case <-stopCh:
		// Already closed
	default:
		close(stopCh)
	}
	a.mu.Unlock()

	// Send STOP event
	interp.Send(statekit.Event{Type: EventStop})

	// Wait for scheduler to finish
	select {
	case <-stoppedCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Stop the interpreter
	a.mu.Lock()
	interp.Stop()
	a.interp = nil
	a.mu.Unlock()

	return nil
}

// State returns the current state.
func (a *Agent) State() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.interp == nil {
		return StateStopped
	}
	return State(a.interp.State().Value)
}

// Status returns the current agent status.
func (a *Agent) Status() Status {
	status := a.runtime.GetStatus()
	status.State = a.State()

	if !status.StartedAt.IsZero() {
		status.Uptime = time.Since(status.StartedAt)
	}

	// Calculate next reconcile time
	ctx := a.runtime.GetContext()
	if ctx.Config != nil && a.State() == StateRunning {
		if !status.LastReconcileAt.IsZero() {
			status.NextReconcileAt = status.LastReconcileAt.Add(ctx.Config.Schedule.Interval())
		} else {
			status.NextReconcileAt = status.StartedAt.Add(ctx.Config.Schedule.Interval())
		}
	}

	return status
}

// runScheduler runs the reconciliation scheduler loop.
func (a *Agent) runScheduler(ctx context.Context) {
	defer close(a.stoppedCh)

	// Wait a bit for the agent to fully start
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return
	case <-a.stopCh:
		return
	}

	runtimeCtx := a.runtime.GetContext()
	schedule := runtimeCtx.Config.Schedule
	ticker := time.NewTicker(schedule.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.triggerReconciliation(ctx)
		}
	}
}

// triggerReconciliation triggers a reconciliation cycle.
func (a *Agent) triggerReconciliation(ctx context.Context) {
	// Only reconcile if in running state
	if a.State() != StateRunning {
		return
	}

	// Send TICK event to transition to reconciling
	a.interp.Send(statekit.Event{Type: EventTick})

	// Get reconcile handler
	a.mu.RLock()
	handler := a.onReconcile
	a.mu.RUnlock()

	if handler == nil {
		a.interp.Send(statekit.Event{Type: EventReconcileComplete})
		return
	}

	// Run reconciliation
	result, err := handler(ctx)
	if err != nil {
		a.interp.Send(statekit.Event{
			Type:    EventError,
			Payload: map[string]interface{}{"error": err},
		})
		return
	}

	// Record result
	a.runtime.RecordReconciliation(result)

	// Transition back to running
	a.interp.Send(statekit.Event{Type: EventReconcileComplete})
}

// SendEvent sends an event to the agent state machine.
func (a *Agent) SendEvent(event string, data map[string]interface{}) {
	a.mu.RLock()
	interp := a.interp
	a.mu.RUnlock()

	if interp != nil {
		interp.Send(statekit.Event{Type: statekit.EventType(event), Payload: data})
	}
}

// Recover attempts to recover the agent from an error state.
func (a *Agent) Recover() {
	a.SendEvent(EventRecover, nil)
}

// Runtime returns the runtime context for testing purposes.
func (a *Agent) Runtime() *RuntimeContext {
	return a.runtime
}
