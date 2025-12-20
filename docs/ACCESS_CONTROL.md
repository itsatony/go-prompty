# Access Control Guide

go-prompty provides a flexible, unopinionated access control system for template management. This guide covers all access control components and how to integrate them with your application's authentication and authorization systems.

## Design Philosophy

**Zero opinions on HOW access is checked.**

go-prompty provides:
- Abstract interfaces for access decisions
- Hooks at every operation point
- Composable checker patterns
- Audit logging interface

Users implement their own logic based on their:
- Authentication system
- Authorization model (RBAC, ABAC, custom)
- Multi-tenancy requirements
- Compliance needs

## Core Interfaces

### AccessChecker

The central interface for access control decisions:

```go
type AccessChecker interface {
    Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error)
    BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error)
}
```

### AccessRequest

Carries all context for making access decisions:

```go
type AccessRequest struct {
    Operation     Operation           // read, execute, create, update, delete, list
    TemplateName  string              // Name of the template
    Subject       *AccessSubject      // Who is requesting access
    Resource      *StoredTemplate     // The template (if loaded)
    ExecutionData map[string]any      // For execute operations
    Metadata      map[string]any      // Extensible context
}
```

**Operations:**
- `OpRead` - Reading template content
- `OpExecute` - Executing a template
- `OpCreate` - Creating a new template
- `OpUpdate` - Updating an existing template
- `OpDelete` - Deleting a template
- `OpList` - Listing templates

### AccessSubject

Represents the entity requesting access. Intentionally broad to support any auth model:

```go
type AccessSubject struct {
    ID        string              // User/service ID
    Type      string              // "user", "service", "system"
    TenantID  string              // Multi-tenant organization ID
    Roles     []string            // RBAC roles
    Groups    []string            // Group memberships
    Scopes    []string            // OAuth2-style scopes
    Attrs     map[string]string   // ABAC attributes
    Extra     map[string]any      // Anything else needed
}
```

### AccessDecision

The result of an access check:

```go
type AccessDecision struct {
    Allowed    bool                // Whether access is granted
    Reason     string              // Human-readable explanation
    Conditions []string            // Conditional grants
    ExpiresAt  *time.Time          // For caching decisions
}
```

## Built-in Helper Checkers

go-prompty provides several helper checkers for common patterns:

### AllowAllChecker

Allows all access (useful for development/testing):

```go
checker := &prompty.AllowAllChecker{}
```

### DenyAllChecker

Denies all access with a custom reason:

```go
checker := prompty.NewDenyAllChecker("system in maintenance mode")
```

### ChainedChecker

Chains multiple checkers with AND logic (all must allow):

```go
checker := prompty.NewChainedChecker(
    &prompty.TenantChecker{},
    &prompty.RoleChecker{RequiredRoles: []string{"editor"}},
)
```

### AnyOfChecker

Chains multiple checkers with OR logic (any can allow):

```go
checker := prompty.NewAnyOfChecker(
    adminChecker,    // Admins can do anything
    ownerChecker,    // Owners can do anything
    roleChecker,     // Or specific roles
)
```

### CachedChecker

Wraps any checker with caching for performance:

```go
checker := prompty.NewCachedChecker(innerChecker, prompty.CachedCheckerConfig{
    TTL:        5 * time.Minute,
    MaxEntries: 1000,
})
```

### TenantChecker

Enforces tenant isolation:

```go
checker := &prompty.TenantChecker{}
// Ensures req.Subject.TenantID matches req.Resource.TenantID
```

### RoleChecker

Requires specific roles:

```go
checker := &prompty.RoleChecker{
    RequiredRoles: []string{"admin", "editor"},
    RequireAll:    false,  // Any of the roles (OR)
}
```

### OperationChecker

Allows only specific operations:

```go
checker := prompty.NewOperationChecker(
    prompty.OpRead,
    prompty.OpExecute,
    prompty.OpList,
)
```

## Hook System

Hooks provide extension points at every operation stage:

### Hook Points

| Hook Point | When Called |
|------------|-------------|
| `HookBeforeLoad` | Before loading a template |
| `HookAfterLoad` | After loading a template |
| `HookBeforeExecute` | Before executing a template |
| `HookAfterExecute` | After executing a template |
| `HookBeforeSave` | Before saving a template |
| `HookAfterSave` | After saving a template |
| `HookBeforeDelete` | Before deleting a template |
| `HookAfterDelete` | After deleting a template |
| `HookBeforeValidate` | Before validating a template |
| `HookAfterValidate` | After validating a template |

### Hook Function Signature

```go
type Hook func(ctx context.Context, point HookPoint, data *HookData) error
```

### HookData

Contains all context passed to hooks:

```go
type HookData struct {
    Subject          *AccessSubject
    Template         *StoredTemplate
    TemplateName     string
    Operation        Operation
    ExecutionData    map[string]any
    Result           string           // For after_execute
    ValidationResult *ValidationResult // For after_validate
    Error            error            // For after_* hooks
    Metadata         map[string]any
}
```

### Hook Behavior

- **Before hooks**: Return an error to abort the operation
- **After hooks**: Errors are logged but don't affect the result

### Built-in Hooks

```go
// Logging hook
hook := prompty.LoggingHook(func(point HookPoint, data *HookData) {
    log.Printf("[%s] %s on %s", point, data.Operation, data.TemplateName)
})

// Timing hook
hook, getElapsed := prompty.TimingHook()
// Use getElapsed(data) to get execution time after hooks run

// Access check hook
hook := prompty.AccessCheckHook(checker)
// Performs access check in before hooks

// Audit hook
hook := prompty.AuditHook(auditor)
// Logs operations to an auditor in after hooks
```

## Audit Logging

### AccessAuditor Interface

```go
type AccessAuditor interface {
    Log(ctx context.Context, event *AccessAuditEvent) error
}
```

### AccessAuditEvent

```go
type AccessAuditEvent struct {
    ID              string
    Timestamp       time.Time
    Subject         *AccessSubject
    Operation       Operation
    TemplateName    string
    TemplateID      string
    TemplateVersion int
    Decision        *AccessDecision
    Duration        time.Duration
    Error           error
    RequestID       string
    Metadata        map[string]any
}
```

### Built-in Auditors

```go
// No-op (for development)
auditor := &prompty.NoOpAuditor{}

// In-memory (for testing)
auditor := prompty.NewMemoryAuditor(maxEvents)
events := auditor.Events()  // Get all events
auditor.Clear()             // Clear events

// Channel-based (for async processing)
ch := make(chan *AccessAuditEvent, 100)
auditor := prompty.NewChannelAuditor(ch)

// Function wrapper
auditor := prompty.NewFuncAuditor(func(ctx context.Context, event *AccessAuditEvent) error {
    return myLogger.LogAccess(event)
})

// Multi-auditor (send to multiple destinations)
auditor := prompty.NewMultiAuditor(
    fileAuditor,
    databaseAuditor,
    metricsAuditor,
)

// Auditing checker wrapper (automatic logging)
checker := prompty.NewAuditingChecker(innerChecker, auditor)
```

## SecureStorageEngine

`SecureStorageEngine` wraps `StorageEngine` with access control and hooks:

```go
engine, err := prompty.NewSecureStorageEngine(prompty.SecureStorageEngineConfig{
    StorageEngineConfig: prompty.StorageEngineConfig{
        Storage: storage,
    },
    AccessChecker: checker,
    Auditor:       auditor,
})
```

### Secure Operations

All operations require an `AccessSubject`:

```go
subject := prompty.NewAccessSubject("usr_123").
    WithTenant("org_456").
    WithRoles("editor")

// Execute with access control
result, err := engine.ExecuteSecure(ctx, "greeting", data, subject)

// Get with access control
tmpl, err := engine.GetSecure(ctx, "greeting", subject)

// Save with access control
err := engine.SaveSecure(ctx, tmpl, subject)

// Delete with access control
err := engine.DeleteSecure(ctx, "greeting", subject)

// List (filtered by access)
templates, err := engine.ListSecure(ctx, query, subject)

// Validate with access control
result, err := engine.ValidateSecure(ctx, "greeting", subject)
```

### Hook Registration

```go
// Register a single hook
engine.RegisterHook(prompty.HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
    log.Printf("Executing template: %s", data.TemplateName)
    return nil
})

// Register for multiple points
engine.RegisterHooks(myHook, prompty.HookBeforeLoad, prompty.HookAfterLoad)

// Clear hooks
engine.ClearHooks(prompty.HookBeforeExecute)
engine.ClearAllHooks()

// Access hook registry directly
registry := engine.Hooks()
```

## Implementation Examples

### RBAC Implementation

See `examples/access_rbac/main.go` for a complete example:

```go
type RBACChecker struct {
    rolePermissions map[string][]Permission
}

func (c *RBACChecker) Check(ctx context.Context, req *prompty.AccessRequest) (*prompty.AccessDecision, error) {
    if req.Subject == nil {
        return prompty.Deny("no subject"), nil
    }

    for _, role := range req.Subject.Roles {
        if perms, ok := c.rolePermissions[role]; ok {
            for _, perm := range perms {
                if perm.Allows(req.Operation) {
                    return prompty.Allow(fmt.Sprintf("granted by role %s", role)), nil
                }
            }
        }
    }

    return prompty.Deny("no role grants permission"), nil
}
```

### Multi-Tenant Isolation

See `examples/access_tenant/main.go` for a complete example:

```go
type TenantChecker struct{}

func (c *TenantChecker) Check(ctx context.Context, req *prompty.AccessRequest) (*prompty.AccessDecision, error) {
    if req.Subject == nil {
        return prompty.Deny("no subject"), nil
    }

    // System users can access any tenant
    if req.Subject.Type == "system" {
        return prompty.Allow("system access"), nil
    }

    // Check tenant match on resource
    if req.Resource != nil && req.Resource.TenantID != "" {
        if req.Resource.TenantID != req.Subject.TenantID {
            return prompty.Deny("cross-tenant access denied"), nil
        }
    }

    return prompty.Allow("tenant verified"), nil
}
```

### Auto-Tagging with Hooks

```go
// Automatically set tenant ID on new templates
engine.RegisterHook(prompty.HookBeforeSave, func(ctx context.Context, point HookPoint, data *HookData) error {
    if data.Template != nil && data.Subject != nil {
        if data.Template.TenantID == "" {
            data.Template.TenantID = data.Subject.TenantID
        }
    }
    return nil
})
```

## Best Practices

### 1. Fail Closed

Always deny access on errors:

```go
func (c *MyChecker) Check(ctx context.Context, req *prompty.AccessRequest) (*prompty.AccessDecision, error) {
    // If anything goes wrong, deny
    if req.Subject == nil {
        return prompty.Deny("no subject"), nil
    }

    result, err := c.authService.CheckPermission(ctx, req)
    if err != nil {
        // Log the error, but deny access
        log.Printf("auth check failed: %v", err)
        return prompty.Deny("authorization check failed"), nil
    }

    return result, nil
}
```

### 2. Cache Decisions

Use `CachedChecker` for frequently repeated checks:

```go
checker := prompty.NewCachedChecker(expensiveChecker, prompty.CachedCheckerConfig{
    TTL:        5 * time.Minute,
    MaxEntries: 10000,
})
```

### 3. Audit Everything

Use the auditing checker to log all access decisions:

```go
checker := prompty.NewAuditingChecker(
    innerChecker,
    prompty.NewMultiAuditor(memoryAuditor, externalAuditor),
)
```

### 4. Compose Checkers

Build complex policies by composing simple checkers:

```go
// Allow system users OR (tenant match AND role match)
checker := prompty.NewAnyOfChecker(
    &SystemUserChecker{},
    prompty.NewChainedChecker(
        &prompty.TenantChecker{},
        &prompty.RoleChecker{RequiredRoles: []string{"editor"}},
    ),
)
```

### 5. Use Hooks for Cross-Cutting Concerns

Hooks are perfect for:
- Logging and metrics
- Rate limiting
- Request tracing
- Automatic tagging
- Validation

```go
engine.RegisterHook(prompty.HookBeforeExecute, rateLimitHook)
engine.RegisterHook(prompty.HookAfterExecute, metricsHook)
engine.RegisterHook(prompty.HookBeforeSave, validationHook)
```

## Error Handling

### Access Errors

```go
// Create access denied error
err := prompty.NewAccessDeniedError(prompty.OpExecute, "greeting", subject)

// Check for access denied
if prompty.IsAccessDenied(err) {
    // Handle access denied
}

// Hook errors include the hook point
err := prompty.NewHookError(prompty.HookBeforeExecute, cause)
```

### Error Types

| Error | Description |
|-------|-------------|
| `AccessDeniedError` | Access was explicitly denied |
| `AccessCheckError` | Error during access check |
| `HookError` | Error from a hook function |

## Thread Safety

All access control components are thread-safe:
- `SecureStorageEngine` can be shared across goroutines
- `HookRegistry` handles concurrent registration/execution
- Built-in checkers are safe for concurrent use
- Memory auditor uses proper synchronization

## Migration Guide

### From Non-Secure to Secure Engine

```go
// Before
engine, _ := prompty.NewStorageEngine(config)
result, _ := engine.Execute(ctx, "greeting", data)

// After
secureEngine, _ := prompty.NewSecureStorageEngine(secureConfig)
result, _ := secureEngine.ExecuteSecure(ctx, "greeting", data, subject)
```

### Adding Access Control to Existing Code

1. Define your `AccessChecker` implementation
2. Create subjects for your users
3. Replace `StorageEngine` with `SecureStorageEngine`
4. Add `AccessSubject` to all operation calls
5. Register hooks for logging/auditing
