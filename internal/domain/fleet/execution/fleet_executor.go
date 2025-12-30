package execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/felixgeelhaar/preflight/internal/domain/fleet/transport"
)

// Strategy defines how hosts are processed.
type Strategy string

const (
	// StrategyParallel processes all hosts in parallel.
	StrategyParallel Strategy = "parallel"
	// StrategyRolling processes hosts in batches.
	StrategyRolling Strategy = "rolling"
	// StrategyCanary processes a canary host first, then the rest.
	StrategyCanary Strategy = "canary"
)

// ExecutorConfig configures the fleet executor.
type ExecutorConfig struct {
	// Strategy is the execution strategy.
	Strategy Strategy
	// MaxParallel is the maximum concurrent host executions.
	MaxParallel int
	// BatchSize is the batch size for rolling strategy.
	BatchSize int
	// StopOnError stops execution on first error.
	StopOnError bool
	// Timeout is the per-host timeout.
	Timeout time.Duration
	// DryRun skips actual execution.
	DryRun bool
}

// DefaultExecutorConfig returns sensible defaults.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Strategy:    StrategyParallel,
		MaxParallel: 10,
		BatchSize:   5,
		StopOnError: false,
		Timeout:     5 * time.Minute,
		DryRun:      false,
	}
}

// FleetExecutor orchestrates step execution across multiple hosts.
type FleetExecutor struct {
	transport transport.Transport
	config    ExecutorConfig
}

// NewFleetExecutor creates a new fleet executor.
func NewFleetExecutor(t transport.Transport, config ExecutorConfig) *FleetExecutor {
	if config.MaxParallel <= 0 {
		config.MaxParallel = 10
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	return &FleetExecutor{
		transport: t,
		config:    config,
	}
}

// Execute runs steps on all hosts.
func (e *FleetExecutor) Execute(ctx context.Context, hosts []*fleet.Host, steps []*RemoteStep) *FleetResult {
	result := NewFleetResult()

	if len(hosts) == 0 {
		result.Complete()
		return result
	}

	switch e.config.Strategy {
	case StrategyParallel:
		e.executeParallel(ctx, hosts, steps, result)
	case StrategyRolling:
		e.executeRolling(ctx, hosts, steps, result)
	case StrategyCanary:
		e.executeCanary(ctx, hosts, steps, result)
	default:
		e.executeParallel(ctx, hosts, steps, result)
	}

	result.Complete()
	return result
}

func (e *FleetExecutor) executeParallel(ctx context.Context, hosts []*fleet.Host, steps []*RemoteStep, result *FleetResult) {
	sem := make(chan struct{}, e.config.MaxParallel)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var stopFlag bool

	for _, host := range hosts {
		if e.config.StopOnError && stopFlag {
			hr := &HostResult{
				HostID:   host.ID(),
				Hostname: host.SSH().Hostname,
				Status:   HostStatusSkipped,
			}
			mu.Lock()
			result.AddHostResult(hr)
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(h *fleet.Host) {
			defer wg.Done()
			defer func() { <-sem }()

			hr := e.executeOnHost(ctx, h, steps)

			mu.Lock()
			result.AddHostResult(hr)
			if hr.Status == HostStatusFailed && e.config.StopOnError {
				stopFlag = true
			}
			mu.Unlock()
		}(host)
	}

	wg.Wait()
}

func (e *FleetExecutor) executeRolling(ctx context.Context, hosts []*fleet.Host, steps []*RemoteStep, result *FleetResult) {
	for i := 0; i < len(hosts); i += e.config.BatchSize {
		end := i + e.config.BatchSize
		if end > len(hosts) {
			end = len(hosts)
		}
		batch := hosts[i:end]

		batchResult := NewFleetResult()
		e.executeParallel(ctx, batch, steps, batchResult)

		for _, hr := range batchResult.HostResults {
			result.AddHostResult(hr)
		}

		// Check for stop condition after each batch
		if e.config.StopOnError && batchResult.FailedHosts() > 0 {
			// Skip remaining hosts
			for j := end; j < len(hosts); j++ {
				hr := &HostResult{
					HostID:   hosts[j].ID(),
					Hostname: hosts[j].SSH().Hostname,
					Status:   HostStatusSkipped,
				}
				result.AddHostResult(hr)
			}
			break
		}
	}
}

func (e *FleetExecutor) executeCanary(ctx context.Context, hosts []*fleet.Host, steps []*RemoteStep, result *FleetResult) {
	if len(hosts) == 0 {
		return
	}

	// Execute on canary host first
	canary := hosts[0]
	canaryResult := e.executeOnHost(ctx, canary, steps)
	result.AddHostResult(canaryResult)

	// If canary failed, skip remaining hosts
	if canaryResult.Status == HostStatusFailed {
		for i := 1; i < len(hosts); i++ {
			hr := &HostResult{
				HostID:   hosts[i].ID(),
				Hostname: hosts[i].SSH().Hostname,
				Status:   HostStatusSkipped,
				Error:    fmt.Errorf("skipped due to canary failure on %s", canary.ID()),
			}
			result.AddHostResult(hr)
		}
		return
	}

	// Execute on remaining hosts
	if len(hosts) > 1 {
		e.executeRolling(ctx, hosts[1:], steps, result)
	}
}

func (e *FleetExecutor) executeOnHost(ctx context.Context, host *fleet.Host, steps []*RemoteStep) *HostResult {
	hr := &HostResult{
		HostID:      host.ID(),
		Hostname:    host.SSH().Hostname,
		Status:      HostStatusRunning,
		StartTime:   time.Now(),
		StepResults: make([]StepResult, 0, len(steps)),
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// Connect to host
	conn, err := e.transport.Connect(ctx, host)
	if err != nil {
		hr.Status = HostStatusFailed
		hr.Error = fmt.Errorf("connection failed: %w", err)
		hr.EndTime = time.Now()
		return hr
	}
	defer func() { _ = conn.Close() }()

	// Execute each step
	for _, step := range steps {
		sr := e.executeStep(ctx, conn, step)
		hr.StepResults = append(hr.StepResults, sr)

		if sr.Error != nil && e.config.StopOnError {
			hr.Status = HostStatusFailed
			hr.Error = fmt.Errorf("step %s failed: %w", step.ID(), sr.Error)
			hr.EndTime = time.Now()
			return hr
		}
	}

	// Determine final status
	failed := false
	for _, sr := range hr.StepResults {
		if sr.Error != nil {
			failed = true
			break
		}
	}

	if failed {
		hr.Status = HostStatusFailed
	} else {
		hr.Status = HostStatusSuccess
	}

	hr.EndTime = time.Now()
	return hr
}

func (e *FleetExecutor) executeStep(ctx context.Context, conn transport.Connection, step *RemoteStep) StepResult {
	sr := StepResult{
		StepID: step.ID(),
	}

	start := time.Now()

	// Check if step is needed
	if !e.config.DryRun {
		status, err := step.Check(ctx, conn)
		if err != nil {
			sr.Status = StepStatusUnknown
			sr.Error = err
			sr.Duration = time.Since(start)
			return sr
		}

		sr.Status = status

		if status == StepStatusSatisfied {
			sr.Applied = false
			sr.Duration = time.Since(start)
			return sr
		}
	} else {
		sr.Status = StepStatusNeeds
	}

	// Apply step
	if !e.config.DryRun {
		err := step.Apply(ctx, conn)
		if err != nil {
			sr.Error = err
			sr.Applied = false
		} else {
			sr.Applied = true
		}
	} else {
		sr.Applied = false
		sr.Output = "[dry-run] would apply: " + step.Command()
	}

	sr.Duration = time.Since(start)
	return sr
}

// Plan generates a plan without executing.
func (e *FleetExecutor) Plan(ctx context.Context, hosts []*fleet.Host, steps []*RemoteStep) (*FleetPlan, error) {
	plan := &FleetPlan{
		Hosts: make([]*HostPlan, 0, len(hosts)),
	}

	for _, host := range hosts {
		hostPlan := &HostPlan{
			HostID:   host.ID(),
			Hostname: host.SSH().Hostname,
			Steps:    make([]StepPlan, 0, len(steps)),
		}

		// Connect to check current state
		conn, err := e.transport.Connect(ctx, host)
		if err != nil {
			hostPlan.Error = err
			plan.Hosts = append(plan.Hosts, hostPlan)
			continue
		}

		for _, step := range steps {
			stepPlan := StepPlan{
				StepID:      step.ID(),
				Description: step.Description(),
				Command:     step.Command(),
			}

			status, err := step.Check(ctx, conn)
			if err != nil {
				stepPlan.Status = StepStatusUnknown
				stepPlan.Error = err
			} else {
				stepPlan.Status = status
			}

			hostPlan.Steps = append(hostPlan.Steps, stepPlan)
		}

		_ = conn.Close()
		plan.Hosts = append(plan.Hosts, hostPlan)
	}

	return plan, nil
}

// FleetPlan represents the planned changes for all hosts.
type FleetPlan struct {
	Hosts []*HostPlan
}

// TotalChanges returns the total number of changes across all hosts.
func (p *FleetPlan) TotalChanges() int {
	count := 0
	for _, hp := range p.Hosts {
		for _, sp := range hp.Steps {
			if sp.Status == StepStatusNeeds {
				count++
			}
		}
	}
	return count
}

// HostsWithChanges returns the number of hosts that have changes.
func (p *FleetPlan) HostsWithChanges() int {
	count := 0
	for _, hp := range p.Hosts {
		for _, sp := range hp.Steps {
			if sp.Status == StepStatusNeeds {
				count++
				break
			}
		}
	}
	return count
}

// HostPlan represents the plan for a single host.
type HostPlan struct {
	HostID   fleet.HostID
	Hostname string
	Steps    []StepPlan
	Error    error
}

// HasChanges returns true if this host has changes to apply.
func (p *HostPlan) HasChanges() bool {
	for _, sp := range p.Steps {
		if sp.Status == StepStatusNeeds {
			return true
		}
	}
	return false
}

// StepPlan represents the plan for a single step.
type StepPlan struct {
	StepID      string
	Description string
	Command     string
	Status      StepStatus
	Error       error
}
