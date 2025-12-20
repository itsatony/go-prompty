package prompty

import (
	"context"
)

// SecureStorageEngine extends StorageEngine with access control and hooks.
// All operations are checked against the configured AccessChecker and
// hooks are invoked at appropriate points.
type SecureStorageEngine struct {
	*StorageEngine
	checker  AccessChecker
	hooks    *HookRegistry
	auditor  AccessAuditor
}

// SecureStorageEngineConfig configures the SecureStorageEngine.
type SecureStorageEngineConfig struct {
	// StorageEngineConfig is the underlying storage engine configuration.
	StorageEngineConfig

	// AccessChecker performs access control checks.
	// If nil, AllowAllChecker is used (no access control).
	AccessChecker AccessChecker

	// Auditor logs access decisions.
	// If nil, no audit logging is performed.
	Auditor AccessAuditor
}

// NewSecureStorageEngine creates a new SecureStorageEngine.
func NewSecureStorageEngine(config SecureStorageEngineConfig) (*SecureStorageEngine, error) {
	se, err := NewStorageEngine(config.StorageEngineConfig)
	if err != nil {
		return nil, err
	}

	checker := config.AccessChecker
	if checker == nil {
		checker = &AllowAllChecker{}
	}

	// Wrap checker with auditing if auditor is provided
	if config.Auditor != nil {
		checker = NewAuditingChecker(checker, config.Auditor)
	}

	return &SecureStorageEngine{
		StorageEngine: se,
		checker:       checker,
		hooks:         NewHookRegistry(),
		auditor:       config.Auditor,
	}, nil
}

// MustNewSecureStorageEngine creates a SecureStorageEngine, panicking on error.
func MustNewSecureStorageEngine(config SecureStorageEngineConfig) *SecureStorageEngine {
	se, err := NewSecureStorageEngine(config)
	if err != nil {
		panic(err)
	}
	return se
}

// RegisterHook registers a hook for the specified point.
func (se *SecureStorageEngine) RegisterHook(point HookPoint, hook Hook) {
	se.hooks.Register(point, hook)
}

// RegisterHooks registers a hook for multiple points.
func (se *SecureStorageEngine) RegisterHooks(hook Hook, points ...HookPoint) {
	se.hooks.RegisterMultiple(hook, points...)
}

// ClearHooks removes all hooks for a specific point.
func (se *SecureStorageEngine) ClearHooks(point HookPoint) {
	se.hooks.Clear(point)
}

// ClearAllHooks removes all hooks.
func (se *SecureStorageEngine) ClearAllHooks() {
	se.hooks.ClearAll()
}

// Hooks returns the hook registry for direct access.
func (se *SecureStorageEngine) Hooks() *HookRegistry {
	return se.hooks
}

// AccessChecker returns the configured access checker.
func (se *SecureStorageEngine) AccessChecker() AccessChecker {
	return se.checker
}

// ExecuteSecure executes a stored template with access control.
func (se *SecureStorageEngine) ExecuteSecure(ctx context.Context, templateName string, data map[string]any, subject *AccessSubject) (string, error) {
	// Load template first to enable resource-level access control
	tmpl, err := se.StorageEngine.Get(ctx, templateName)
	if err != nil {
		return "", err
	}

	// Check access with loaded resource
	req := NewAccessRequest(OpExecute, templateName, subject).
		WithExecutionData(data).
		WithResource(tmpl)

	if err := se.checkAccess(ctx, req); err != nil {
		return "", err
	}

	// Run before hooks with template
	hookData := NewHookData(OpExecute, templateName, subject).
		WithExecutionData(data).
		WithTemplate(tmpl)

	if err := se.hooks.Run(ctx, HookBeforeExecute, hookData); err != nil {
		return "", NewHookError(HookBeforeExecute, err)
	}

	// Execute
	result, execErr := se.StorageEngine.Execute(ctx, templateName, data)

	// Run after hooks
	hookData.WithResult(result).WithError(execErr)
	_ = se.hooks.Run(ctx, HookAfterExecute, hookData)

	return result, execErr
}

// ExecuteVersionSecure executes a specific version with access control.
func (se *SecureStorageEngine) ExecuteVersionSecure(ctx context.Context, templateName string, version int, data map[string]any, subject *AccessSubject) (string, error) {
	// Load template first to enable resource-level access control
	tmpl, err := se.StorageEngine.GetVersion(ctx, templateName, version)
	if err != nil {
		return "", err
	}

	// Check access with loaded resource
	req := NewAccessRequest(OpExecute, templateName, subject).
		WithExecutionData(data).
		WithResource(tmpl)

	if err := se.checkAccess(ctx, req); err != nil {
		return "", err
	}

	// Run before hooks with template
	hookData := NewHookData(OpExecute, templateName, subject).
		WithExecutionData(data).
		WithTemplate(tmpl)

	if err := se.hooks.Run(ctx, HookBeforeExecute, hookData); err != nil {
		return "", NewHookError(HookBeforeExecute, err)
	}

	// Execute
	result, execErr := se.StorageEngine.ExecuteVersion(ctx, templateName, version, data)

	// Run after hooks
	hookData.WithResult(result).WithError(execErr)
	_ = se.hooks.Run(ctx, HookAfterExecute, hookData)

	return result, execErr
}

// GetSecure retrieves a template with access control.
func (se *SecureStorageEngine) GetSecure(ctx context.Context, templateName string, subject *AccessSubject) (*StoredTemplate, error) {
	// Check access
	req := NewAccessRequest(OpRead, templateName, subject)

	if err := se.checkAccess(ctx, req); err != nil {
		return nil, err
	}

	// Run before hooks
	hookData := NewHookData(OpRead, templateName, subject)

	if err := se.hooks.Run(ctx, HookBeforeLoad, hookData); err != nil {
		return nil, NewHookError(HookBeforeLoad, err)
	}

	// Load
	tmpl, loadErr := se.StorageEngine.Get(ctx, templateName)

	// Run after hooks
	hookData.WithTemplate(tmpl).WithError(loadErr)
	_ = se.hooks.Run(ctx, HookAfterLoad, hookData)

	return tmpl, loadErr
}

// SaveSecure stores a template with access control.
func (se *SecureStorageEngine) SaveSecure(ctx context.Context, tmpl *StoredTemplate, subject *AccessSubject) error {
	// Determine operation (create or update)
	op := OpCreate
	exists, _ := se.StorageEngine.Exists(ctx, tmpl.Name)
	if exists {
		op = OpUpdate
	}

	// Check access
	req := NewAccessRequest(op, tmpl.Name, subject).
		WithResource(tmpl)

	if err := se.checkAccess(ctx, req); err != nil {
		return err
	}

	// Run before hooks
	hookData := NewHookData(op, tmpl.Name, subject).
		WithTemplate(tmpl)

	if err := se.hooks.Run(ctx, HookBeforeSave, hookData); err != nil {
		return NewHookError(HookBeforeSave, err)
	}

	// Save
	saveErr := se.StorageEngine.Save(ctx, tmpl)

	// Run after hooks
	hookData.WithError(saveErr)
	_ = se.hooks.Run(ctx, HookAfterSave, hookData)

	return saveErr
}

// DeleteSecure removes a template with access control.
func (se *SecureStorageEngine) DeleteSecure(ctx context.Context, templateName string, subject *AccessSubject) error {
	// Load template first to enable resource-level access control
	tmpl, err := se.StorageEngine.Get(ctx, templateName)
	if err != nil {
		return err
	}

	// Check access with loaded resource
	req := NewAccessRequest(OpDelete, templateName, subject).
		WithResource(tmpl)

	if err := se.checkAccess(ctx, req); err != nil {
		return err
	}

	// Run before hooks with template
	hookData := NewHookData(OpDelete, templateName, subject).
		WithTemplate(tmpl)

	if err := se.hooks.Run(ctx, HookBeforeDelete, hookData); err != nil {
		return NewHookError(HookBeforeDelete, err)
	}

	// Delete
	deleteErr := se.StorageEngine.Delete(ctx, templateName)

	// Run after hooks
	hookData.WithError(deleteErr)
	_ = se.hooks.Run(ctx, HookAfterDelete, hookData)

	return deleteErr
}

// DeleteVersionSecure removes a specific version with access control.
func (se *SecureStorageEngine) DeleteVersionSecure(ctx context.Context, templateName string, version int, subject *AccessSubject) error {
	// Load template first to enable resource-level access control
	tmpl, err := se.StorageEngine.GetVersion(ctx, templateName, version)
	if err != nil {
		return err
	}

	// Check access with loaded resource
	req := NewAccessRequest(OpDelete, templateName, subject).
		WithResource(tmpl)

	if err := se.checkAccess(ctx, req); err != nil {
		return err
	}

	// Run before hooks with template
	hookData := NewHookData(OpDelete, templateName, subject).
		WithTemplate(tmpl)

	if err := se.hooks.Run(ctx, HookBeforeDelete, hookData); err != nil {
		return NewHookError(HookBeforeDelete, err)
	}

	// Delete
	deleteErr := se.StorageEngine.DeleteVersion(ctx, templateName, version)

	// Run after hooks
	hookData.WithError(deleteErr)
	_ = se.hooks.Run(ctx, HookAfterDelete, hookData)

	return deleteErr
}

// ValidateSecure validates a template with access control.
func (se *SecureStorageEngine) ValidateSecure(ctx context.Context, templateName string, subject *AccessSubject) (*ValidationResult, error) {
	// Check access (read access required for validation)
	req := NewAccessRequest(OpRead, templateName, subject)

	if err := se.checkAccess(ctx, req); err != nil {
		return nil, err
	}

	// Run before hooks
	hookData := NewHookData(OpRead, templateName, subject)

	if err := se.hooks.Run(ctx, HookBeforeValidate, hookData); err != nil {
		return nil, NewHookError(HookBeforeValidate, err)
	}

	// Validate
	result, validateErr := se.StorageEngine.Validate(ctx, templateName)

	// Run after hooks
	hookData.WithValidationResult(result).WithError(validateErr)
	_ = se.hooks.Run(ctx, HookAfterValidate, hookData)

	return result, validateErr
}

// ListSecure returns templates the subject can access.
func (se *SecureStorageEngine) ListSecure(ctx context.Context, query *TemplateQuery, subject *AccessSubject) ([]*StoredTemplate, error) {
	// Check list access
	req := NewAccessRequest(OpList, "", subject)

	if err := se.checkAccess(ctx, req); err != nil {
		return nil, err
	}

	// Get all templates matching query
	templates, err := se.StorageEngine.List(ctx, query)
	if err != nil {
		return nil, err
	}

	// Filter by read access
	var accessible []*StoredTemplate
	for _, tmpl := range templates {
		readReq := NewAccessRequest(OpRead, tmpl.Name, subject).
			WithResource(tmpl)

		decision, err := se.checker.Check(ctx, readReq)
		if err != nil {
			continue // Skip on error
		}
		if decision.Allowed {
			accessible = append(accessible, tmpl)
		}
	}

	return accessible, nil
}

// checkAccess performs an access check and returns an error if denied.
func (se *SecureStorageEngine) checkAccess(ctx context.Context, req *AccessRequest) error {
	decision, err := se.checker.Check(ctx, req)
	if err != nil {
		return NewAccessCheckError(req.Operation, req.TemplateName, err)
	}

	if !decision.Allowed {
		return NewAccessDeniedError(req.Operation, req.TemplateName, req.Subject)
	}

	return nil
}

// CheckAccess explicitly checks access for a request.
// Useful for checking access before expensive operations.
func (se *SecureStorageEngine) CheckAccess(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	return se.checker.Check(ctx, req)
}

// BatchCheckAccess checks access for multiple requests.
func (se *SecureStorageEngine) BatchCheckAccess(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	return se.checker.BatchCheck(ctx, reqs)
}
