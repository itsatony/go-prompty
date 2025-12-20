// Package main demonstrates implementing multi-tenant isolation
// with go-prompty's access control system.
//
// This example shows:
// - Tenant isolation ensuring users can only access their tenant's templates
// - Combining tenant isolation with RBAC
// - Using hooks to automatically tag templates with tenant IDs
// - Cross-tenant access prevention
package main

import (
	"context"
	"fmt"
	"log"

	prompty "github.com/itsatony/go-prompty"
)

// TenantIsolationChecker ensures users can only access templates belonging
// to their tenant. This is the foundation of multi-tenant security.
type TenantIsolationChecker struct {
	// AllowSystemAccess permits subjects with Type="system" to access any tenant
	AllowSystemAccess bool
}

// NewTenantIsolationChecker creates a new tenant isolation checker.
func NewTenantIsolationChecker() *TenantIsolationChecker {
	return &TenantIsolationChecker{
		AllowSystemAccess: true,
	}
}

// Check verifies the subject's tenant matches the template's tenant.
func (c *TenantIsolationChecker) Check(ctx context.Context, req *prompty.AccessRequest) (*prompty.AccessDecision, error) {
	// No subject = deny
	if req.Subject == nil {
		return prompty.Deny("no subject provided"), nil
	}

	// System users can access any tenant (for admin operations)
	if c.AllowSystemAccess && req.Subject.Type == "system" {
		return prompty.Allow("system access granted"), nil
	}

	// List operations don't have a specific resource - allow the operation
	// but the list will be filtered by tenant in ListSecure
	if req.Operation == prompty.OpList {
		return prompty.Allow("list allowed, will be filtered by tenant"), nil
	}

	// For create operations, we allow - the hook will set the tenant ID
	if req.Operation == prompty.OpCreate && req.Resource != nil {
		// If template has no tenant ID, it will be set by hook
		if req.Resource.TenantID == "" {
			return prompty.Allow("create allowed, tenant will be set"), nil
		}
		// If template has tenant ID, it must match subject's tenant
		if req.Resource.TenantID != req.Subject.TenantID {
			return prompty.Deny("cannot create template for different tenant"), nil
		}
		return prompty.Allow("tenant matches"), nil
	}

	// For other operations, check the resource's tenant
	if req.Resource == nil {
		// No resource loaded yet - allow, the actual check will happen after load
		return prompty.Allow("no resource to check"), nil
	}

	// Check tenant match
	if req.Resource.TenantID == "" {
		// Templates without tenant ID are shared (legacy/global templates)
		return prompty.Allow("shared template"), nil
	}

	if req.Resource.TenantID != req.Subject.TenantID {
		return prompty.Deny(fmt.Sprintf(
			"tenant mismatch: template belongs to %s, user is from %s",
			req.Resource.TenantID,
			req.Subject.TenantID,
		)), nil
	}

	return prompty.Allow("tenant verified"), nil
}

// BatchCheck evaluates multiple requests.
func (c *TenantIsolationChecker) BatchCheck(ctx context.Context, reqs []*prompty.AccessRequest) ([]*prompty.AccessDecision, error) {
	decisions := make([]*prompty.AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			return nil, err
		}
		decisions[i] = decision
	}
	return decisions, nil
}

// TenantFilterChecker wraps another checker and filters List results by tenant.
// It's used in combination with TenantIsolationChecker.
type TenantFilterChecker struct {
	inner prompty.AccessChecker
}

// NewTenantFilterChecker creates a checker that filters by tenant.
func NewTenantFilterChecker(inner prompty.AccessChecker) *TenantFilterChecker {
	return &TenantFilterChecker{inner: inner}
}

// Check delegates to inner checker but adds tenant filtering for reads.
func (c *TenantFilterChecker) Check(ctx context.Context, req *prompty.AccessRequest) (*prompty.AccessDecision, error) {
	// First check with inner checker
	decision, err := c.inner.Check(ctx, req)
	if err != nil || !decision.Allowed {
		return decision, err
	}

	// For read operations on loaded resources, verify tenant
	if req.Resource != nil && req.Subject != nil {
		if req.Resource.TenantID != "" && req.Resource.TenantID != req.Subject.TenantID {
			// System users can read any tenant
			if req.Subject.Type == "system" {
				return decision, nil
			}
			return prompty.Deny("cross-tenant access denied"), nil
		}
	}

	return decision, nil
}

// BatchCheck evaluates multiple requests.
func (c *TenantFilterChecker) BatchCheck(ctx context.Context, reqs []*prompty.AccessRequest) ([]*prompty.AccessDecision, error) {
	decisions := make([]*prompty.AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			return nil, err
		}
		decisions[i] = decision
	}
	return decisions, nil
}

// TenantTaggingHook automatically sets the tenant ID on templates during save.
func TenantTaggingHook() prompty.Hook {
	return func(ctx context.Context, point prompty.HookPoint, data *prompty.HookData) error {
		// Only run on before_save
		if point != prompty.HookBeforeSave {
			return nil
		}

		// Set tenant ID from subject if template doesn't have one
		if data.Template != nil && data.Subject != nil {
			if data.Template.TenantID == "" && data.Subject.TenantID != "" {
				data.Template.TenantID = data.Subject.TenantID
				fmt.Printf("  [Hook] Auto-tagged template with tenant: %s\n", data.Subject.TenantID)
			}
		}

		return nil
	}
}

func main() {
	ctx := context.Background()

	// Create storage with tenant isolation
	storage := prompty.NewMemoryStorage()

	// Create tenant isolation checker
	tenantChecker := NewTenantIsolationChecker()

	// Create secure storage engine with tenant isolation
	engine, err := prompty.NewSecureStorageEngine(prompty.SecureStorageEngineConfig{
		StorageEngineConfig: prompty.StorageEngineConfig{
			Storage: storage,
		},
		AccessChecker: tenantChecker,
	})
	if err != nil {
		log.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Register hook to automatically tag templates with tenant ID
	engine.RegisterHook(prompty.HookBeforeSave, TenantTaggingHook())

	// Create users from different tenants
	acmeUser := prompty.NewAccessSubject("usr_alice").
		WithTenant("tenant_acme").
		WithRoles("editor")

	globexUser := prompty.NewAccessSubject("usr_bob").
		WithTenant("tenant_globex").
		WithRoles("editor")

	systemUser := prompty.NewAccessSubject("sys_admin").
		WithType("system").
		WithRoles("admin")

	// === Demo: Acme user creates a template ===
	fmt.Println("=== Acme user creating template ===")
	err = engine.SaveSecure(ctx, &prompty.StoredTemplate{
		Name:   "acme-greeting",
		Source: "Welcome to ACME Corp, {~prompty.var name=\"user\" default=\"Customer\" /~}!",
	}, acmeUser)
	if err != nil {
		fmt.Printf("Acme create: DENIED - %v\n", err)
	} else {
		fmt.Println("Acme create: ALLOWED")
	}

	// === Demo: Globex user creates a template ===
	fmt.Println("\n=== Globex user creating template ===")
	err = engine.SaveSecure(ctx, &prompty.StoredTemplate{
		Name:   "globex-greeting",
		Source: "Greetings from Globex, {~prompty.var name=\"user\" default=\"Partner\" /~}!",
	}, globexUser)
	if err != nil {
		fmt.Printf("Globex create: DENIED - %v\n", err)
	} else {
		fmt.Println("Globex create: ALLOWED")
	}

	// === Demo: Acme user can access their own template ===
	fmt.Println("\n=== Acme user executing their own template ===")
	result, err := engine.ExecuteSecure(ctx, "acme-greeting", map[string]any{
		"user": "Alice",
	}, acmeUser)
	if err != nil {
		fmt.Printf("Acme execute own: DENIED - %v\n", err)
	} else {
		fmt.Printf("Acme execute own: ALLOWED - Result: %s\n", result)
	}

	// === Demo: Globex user CANNOT access Acme's template ===
	fmt.Println("\n=== Globex user trying to access Acme's template ===")
	result, err = engine.ExecuteSecure(ctx, "acme-greeting", map[string]any{
		"user": "Bob",
	}, globexUser)
	if err != nil {
		fmt.Printf("Globex execute Acme's: DENIED - %v\n", err)
	} else {
		fmt.Printf("Globex execute Acme's: ALLOWED - Result: %s\n", result)
	}

	// === Demo: System user CAN access any tenant's template ===
	fmt.Println("\n=== System user accessing Acme's template ===")
	result, err = engine.ExecuteSecure(ctx, "acme-greeting", map[string]any{
		"user": "System",
	}, systemUser)
	if err != nil {
		fmt.Printf("System execute Acme's: DENIED - %v\n", err)
	} else {
		fmt.Printf("System execute Acme's: ALLOWED - Result: %s\n", result)
	}

	// === Demo: List only shows tenant's templates ===
	fmt.Println("\n=== Acme user listing templates ===")
	templates, err := engine.ListSecure(ctx, nil, acmeUser)
	if err != nil {
		fmt.Printf("Acme list: DENIED - %v\n", err)
	} else {
		fmt.Printf("Acme list: ALLOWED - Found %d templates\n", len(templates))
		for _, t := range templates {
			fmt.Printf("  - %s (tenant: %s)\n", t.Name, t.TenantID)
		}
	}

	fmt.Println("\n=== Globex user listing templates ===")
	templates, err = engine.ListSecure(ctx, nil, globexUser)
	if err != nil {
		fmt.Printf("Globex list: DENIED - %v\n", err)
	} else {
		fmt.Printf("Globex list: ALLOWED - Found %d templates\n", len(templates))
		for _, t := range templates {
			fmt.Printf("  - %s (tenant: %s)\n", t.Name, t.TenantID)
		}
	}

	fmt.Println("\n=== System user listing all templates ===")
	templates, err = engine.ListSecure(ctx, nil, systemUser)
	if err != nil {
		fmt.Printf("System list: DENIED - %v\n", err)
	} else {
		fmt.Printf("System list: ALLOWED - Found %d templates\n", len(templates))
		for _, t := range templates {
			fmt.Printf("  - %s (tenant: %s)\n", t.Name, t.TenantID)
		}
	}

	// === Demo: Cross-tenant deletion is blocked ===
	fmt.Println("\n=== Globex user trying to delete Acme's template ===")
	err = engine.DeleteSecure(ctx, "acme-greeting", globexUser)
	if err != nil {
		fmt.Printf("Globex delete Acme's: DENIED - %v\n", err)
	} else {
		fmt.Println("Globex delete Acme's: ALLOWED")
	}

	// === Demo: Owner can delete their own template ===
	fmt.Println("\n=== Acme user deleting their own template ===")
	err = engine.DeleteSecure(ctx, "acme-greeting", acmeUser)
	if err != nil {
		fmt.Printf("Acme delete own: DENIED - %v\n", err)
	} else {
		fmt.Println("Acme delete own: ALLOWED")
	}
}
