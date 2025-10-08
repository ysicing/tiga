package alert

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// AlertEngine processes alert rules and triggers events
type AlertEngine struct {
	alertRepo   repository.MonitorAlertRepository
	hostRepo    repository.HostRepository
	serviceRepo repository.ServiceRepository
}

// NewAlertEngine creates a new alert engine
func NewAlertEngine(alertRepo repository.MonitorAlertRepository, hostRepo repository.HostRepository, serviceRepo repository.ServiceRepository) *AlertEngine {
	return &AlertEngine{
		alertRepo:   alertRepo,
		hostRepo:    hostRepo,
		serviceRepo: serviceRepo,
	}
}

// EvaluateHostRules evaluates all host-related alert rules
func (e *AlertEngine) EvaluateHostRules(ctx context.Context, hostID uuid.UUID, state *models.HostState) error {
	// Get all active host rules
	rules, err := e.alertRepo.GetActiveRules(ctx, string(models.AlertTypeHost))
	if err != nil {
		return err
	}

	// Evaluate each rule
	for _, rule := range rules {
		if rule.TargetID == hostID {
			e.evaluateRule(ctx, rule, state)
		}
	}

	return nil
}

// EvaluateServiceRules evaluates all service-related alert rules
func (e *AlertEngine) EvaluateServiceRules(ctx context.Context, serviceMonitorID uuid.UUID, availability *models.ServiceAvailability) error {
	// Get all active service rules
	rules, err := e.alertRepo.GetActiveRules(ctx, string(models.AlertTypeService))
	if err != nil {
		return err
	}

	// Evaluate each rule
	for _, rule := range rules {
		if rule.TargetID == serviceMonitorID {
			e.evaluateRule(ctx, rule, availability)
		}
	}

	return nil
}

// evaluateRule evaluates a single alert rule
func (e *AlertEngine) evaluateRule(ctx context.Context, rule *models.MonitorAlertRule, data interface{}) {
	// Prepare evaluation environment
	env := e.prepareEnv(data)

	// Compile and run the expression
	program, err := expr.Compile(rule.Condition, expr.Env(env))
	if err != nil {
		fmt.Printf("Error compiling expression: %v\n", err)
		return
	}

	output, err := expr.Run(program, env)
	if err != nil {
		fmt.Printf("Error running expression: %v\n", err)
		return
	}

	// Check if condition is met
	triggered, ok := output.(bool)
	if !ok || !triggered {
		// Condition not met, check if we need to resolve existing events
		e.resolveEvents(ctx, rule.ID)
		return
	}

	// Check if already firing
	firingEvents, _ := e.alertRepo.GetFiringEvents(ctx, rule.ID)
	if len(firingEvents) > 0 {
		// Already firing, update existing event
		return
	}

	// Create new alert event
	contextData, _ := json.Marshal(env)
	event := &models.MonitorAlertEvent{
		RuleID:   rule.ID,
		Status:   models.AlertStatusFiring,
		Severity: rule.Severity,
		Message:  fmt.Sprintf("Alert rule '%s' triggered", rule.Name),
		Context:  string(contextData),
	}

	if err := e.alertRepo.CreateEvent(ctx, event); err != nil {
		fmt.Printf("Error creating alert event: %v\n", err)
	}
}

// prepareEnv prepares the evaluation environment from data
func (e *AlertEngine) prepareEnv(data interface{}) map[string]interface{} {
	env := make(map[string]interface{})

	switch v := data.(type) {
	case *models.HostState:
		env["cpu_usage"] = v.CPUUsage
		env["mem_usage"] = v.MemUsage
		env["disk_usage"] = v.DiskUsage
		env["load_1"] = v.Load1
		env["load_5"] = v.Load5
		env["load_15"] = v.Load15

	case *models.ServiceAvailability:
		// Service monitoring metrics
		env["uptime_percentage"] = v.UptimePercentage
		env["avg_latency"] = v.AvgLatency
		env["total_checks"] = v.TotalChecks
		env["successful_checks"] = v.SuccessfulChecks
		env["failed_checks"] = v.FailedChecks
		env["status_code"] = string(models.ServiceStatusUnknown) // Will be calculated based on uptime

		// Calculate status code based on uptime percentage
		if v.UptimePercentage >= 95.0 {
			env["status_code"] = string(models.ServiceStatusGood)
		} else if v.UptimePercentage >= 80.0 {
			env["status_code"] = string(models.ServiceStatusLowAvailability)
		} else {
			env["status_code"] = string(models.ServiceStatusDown)
		}
	}

	return env
}

// resolveEvents resolves any firing events for a rule
func (e *AlertEngine) resolveEvents(ctx context.Context, ruleID uuid.UUID) {
	events, _ := e.alertRepo.GetFiringEvents(ctx, ruleID)
	for _, event := range events {
		event.Resolve(uuid.Nil, "Condition no longer met") // uuid.Nil indicates system auto-resolve
		e.alertRepo.UpdateEvent(ctx, event)
	}
}

// ProcessPeriodicCheck runs periodic evaluation of all active rules
func (e *AlertEngine) ProcessPeriodicCheck(ctx context.Context) {
	// This should be called periodically (e.g., every 30 seconds)
	// Get all active rules and evaluate them against latest states
}
