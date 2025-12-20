package prompty

import (
	"context"
	"sync"
	"time"
)

// AccessAuditor is the interface for audit logging implementations.
// Implement this to integrate with your logging/compliance systems.
type AccessAuditor interface {
	// Log records an access audit event.
	// Implementations should be non-blocking or use buffering.
	Log(ctx context.Context, event *AccessAuditEvent) error
}

// AccessAuditEvent represents an audit log entry for access decisions.
type AccessAuditEvent struct {
	// Timestamp is when the access check occurred.
	Timestamp time.Time

	// RequestID is a unique identifier for this request (for correlation).
	RequestID string

	// Subject identifies who requested access.
	Subject *AccessSubject

	// Operation is the type of access requested.
	Operation Operation

	// TemplateName is the name of the template.
	TemplateName string

	// TemplateID is the unique identifier of the template.
	TemplateID TemplateID

	// TemplateVersion is the version of the template (if known).
	TemplateVersion int

	// Decision is the access control decision.
	Decision *AccessDecision

	// Duration is how long the access check took.
	Duration time.Duration

	// Error is any error that occurred during the check.
	Error error

	// Metadata contains additional context.
	Metadata map[string]any
}

// NewAccessAuditEvent creates a new audit event.
func NewAccessAuditEvent(op Operation, templateName string, subject *AccessSubject, decision *AccessDecision) *AccessAuditEvent {
	return &AccessAuditEvent{
		Timestamp:    timeNow(),
		Operation:    op,
		TemplateName: templateName,
		Subject:      subject,
		Decision:     decision,
		Metadata:     make(map[string]any),
	}
}

// WithRequestID sets the request ID.
func (e *AccessAuditEvent) WithRequestID(id string) *AccessAuditEvent {
	e.RequestID = id
	return e
}

// WithTemplate sets template details from a StoredTemplate.
func (e *AccessAuditEvent) WithTemplate(tmpl *StoredTemplate) *AccessAuditEvent {
	if tmpl != nil {
		e.TemplateID = tmpl.ID
		e.TemplateName = tmpl.Name
		e.TemplateVersion = tmpl.Version
	}
	return e
}

// WithDuration sets the duration.
func (e *AccessAuditEvent) WithDuration(d time.Duration) *AccessAuditEvent {
	e.Duration = d
	return e
}

// WithError sets the error.
func (e *AccessAuditEvent) WithError(err error) *AccessAuditEvent {
	e.Error = err
	return e
}

// WithMetadata adds metadata to the event.
func (e *AccessAuditEvent) WithMetadata(key string, value any) *AccessAuditEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// AuditingChecker wraps an AccessChecker with audit logging.
type AuditingChecker struct {
	checker AccessChecker
	auditor AccessAuditor
}

// NewAuditingChecker creates a checker that logs all access decisions.
func NewAuditingChecker(checker AccessChecker, auditor AccessAuditor) *AuditingChecker {
	return &AuditingChecker{
		checker: checker,
		auditor: auditor,
	}
}

// Check evaluates access and logs the decision.
func (c *AuditingChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	start := timeNow()

	decision, err := c.checker.Check(ctx, req)

	// Log the audit event
	event := NewAccessAuditEvent(req.Operation, req.TemplateName, req.Subject, decision).
		WithDuration(time.Since(start)).
		WithError(err)

	if req.Resource != nil {
		event.WithTemplate(req.Resource)
	} else if req.TemplateID != "" {
		event.TemplateID = req.TemplateID
	}

	// Copy relevant metadata
	for k, v := range req.Metadata {
		event.WithMetadata(k, v)
	}

	// Log asynchronously to not block the request
	go func() {
		_ = c.auditor.Log(context.Background(), event)
	}()

	return decision, err
}

// BatchCheck evaluates multiple requests and logs each decision.
func (c *AuditingChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			return nil, err
		}
		decisions[i] = decision
	}
	return decisions, nil
}

// NoOpAuditor is an auditor that does nothing.
// Useful for testing or when audit logging is disabled.
type NoOpAuditor struct{}

// Log does nothing.
func (a *NoOpAuditor) Log(ctx context.Context, event *AccessAuditEvent) error {
	return nil
}

// MemoryAuditor stores audit events in memory.
// Useful for testing and debugging.
type MemoryAuditor struct {
	mu     sync.RWMutex
	events []*AccessAuditEvent
	limit  int
}

// NewMemoryAuditor creates an in-memory auditor.
// If limit > 0, only the most recent events are kept.
func NewMemoryAuditor(limit int) *MemoryAuditor {
	return &MemoryAuditor{
		events: make([]*AccessAuditEvent, 0),
		limit:  limit,
	}
}

// Log stores an event in memory.
func (a *MemoryAuditor) Log(ctx context.Context, event *AccessAuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.events = append(a.events, event)

	// Trim if over limit
	if a.limit > 0 && len(a.events) > a.limit {
		a.events = a.events[len(a.events)-a.limit:]
	}

	return nil
}

// Events returns all stored events.
func (a *MemoryAuditor) Events() []*AccessAuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*AccessAuditEvent, len(a.events))
	copy(result, a.events)
	return result
}

// Clear removes all stored events.
func (a *MemoryAuditor) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = make([]*AccessAuditEvent, 0)
}

// Count returns the number of stored events.
func (a *MemoryAuditor) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.events)
}

// LastEvent returns the most recent event, or nil if none.
func (a *MemoryAuditor) LastEvent() *AccessAuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.events) == 0 {
		return nil
	}
	return a.events[len(a.events)-1]
}

// FilteredEvents returns events matching the filter function.
func (a *MemoryAuditor) FilteredEvents(filter func(*AccessAuditEvent) bool) []*AccessAuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []*AccessAuditEvent
	for _, event := range a.events {
		if filter(event) {
			result = append(result, event)
		}
	}
	return result
}

// ChannelAuditor sends events to a channel.
// Useful for streaming audit events to external systems.
type ChannelAuditor struct {
	ch chan<- *AccessAuditEvent
}

// NewChannelAuditor creates an auditor that sends events to a channel.
// The channel should be buffered to prevent blocking.
func NewChannelAuditor(ch chan<- *AccessAuditEvent) *ChannelAuditor {
	return &ChannelAuditor{ch: ch}
}

// Log sends the event to the channel.
// Returns immediately if the channel is full (non-blocking).
func (a *ChannelAuditor) Log(ctx context.Context, event *AccessAuditEvent) error {
	select {
	case a.ch <- event:
		return nil
	default:
		// Channel full, drop event
		return nil
	}
}

// FuncAuditor wraps a function as an auditor.
// Useful for simple logging integrations.
type FuncAuditor struct {
	fn func(context.Context, *AccessAuditEvent) error
}

// NewFuncAuditor creates an auditor from a function.
func NewFuncAuditor(fn func(context.Context, *AccessAuditEvent) error) *FuncAuditor {
	return &FuncAuditor{fn: fn}
}

// Log calls the wrapped function.
func (a *FuncAuditor) Log(ctx context.Context, event *AccessAuditEvent) error {
	return a.fn(ctx, event)
}

// MultiAuditor sends events to multiple auditors.
type MultiAuditor struct {
	auditors []AccessAuditor
}

// NewMultiAuditor creates an auditor that logs to multiple destinations.
func NewMultiAuditor(auditors ...AccessAuditor) *MultiAuditor {
	return &MultiAuditor{auditors: auditors}
}

// Log sends the event to all auditors.
// Continues even if individual auditors fail.
func (a *MultiAuditor) Log(ctx context.Context, event *AccessAuditEvent) error {
	var lastErr error
	for _, auditor := range a.auditors {
		if err := auditor.Log(ctx, event); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// AddAuditor adds an auditor to the multi-auditor.
func (a *MultiAuditor) AddAuditor(auditor AccessAuditor) {
	a.auditors = append(a.auditors, auditor)
}

// AuditHook creates a hook that logs audit events at specific points.
func AuditHook(auditor AccessAuditor) Hook {
	return func(ctx context.Context, point HookPoint, data *HookData) error {
		// Only audit after operations
		if isBeforeHook(point) {
			return nil
		}

		var decision *AccessDecision
		if data.Error != nil {
			decision = Deny("operation failed: " + data.Error.Error())
		} else {
			decision = Allow("operation succeeded")
		}

		event := NewAccessAuditEvent(data.Operation, data.TemplateName, data.Subject, decision).
			WithError(data.Error)

		if data.Template != nil {
			event.WithTemplate(data.Template)
		}

		return auditor.Log(ctx, event)
	}
}

// timeNow is a variable for testing.
var timeNow = time.Now
