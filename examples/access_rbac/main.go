// Package main demonstrates implementing role-based access control (RBAC)
// with go-prompty's access control system.
//
// This example shows:
// - Defining roles with specific permissions
// - Implementing a custom AccessChecker for RBAC
// - Using SecureStorageEngine with RBAC
// - Audit logging for compliance
package main

import (
	"context"
	"fmt"
	"log"

	prompty "github.com/itsatony/go-prompty"
)

// Role represents a user role in the system.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleEditor   Role = "editor"
	RoleViewer   Role = "viewer"
	RoleReviewer Role = "reviewer"
)

// Permission represents an action that can be taken on a resource.
type Permission struct {
	Operations []prompty.Operation
	Tags       []string // Empty means all tags
}

// RBACConfig holds the role-to-permission mappings.
type RBACConfig struct {
	RolePermissions map[Role][]Permission
}

// DefaultRBACConfig returns a sensible default RBAC configuration.
func DefaultRBACConfig() *RBACConfig {
	return &RBACConfig{
		RolePermissions: map[Role][]Permission{
			// Admin can do everything
			RoleAdmin: {
				{Operations: []prompty.Operation{
					prompty.OpCreate,
					prompty.OpRead,
					prompty.OpUpdate,
					prompty.OpDelete,
					prompty.OpExecute,
					prompty.OpList,
				}},
			},
			// Editor can create, read, update, execute, list - but not delete
			RoleEditor: {
				{Operations: []prompty.Operation{
					prompty.OpCreate,
					prompty.OpRead,
					prompty.OpUpdate,
					prompty.OpExecute,
					prompty.OpList,
				}},
			},
			// Viewer can only read, execute, and list
			RoleViewer: {
				{Operations: []prompty.Operation{
					prompty.OpRead,
					prompty.OpExecute,
					prompty.OpList,
				}},
			},
			// Reviewer can read and list but not execute
			RoleReviewer: {
				{Operations: []prompty.Operation{
					prompty.OpRead,
					prompty.OpList,
				}},
			},
		},
	}
}

// RBACChecker implements AccessChecker using role-based access control.
type RBACChecker struct {
	config *RBACConfig
}

// NewRBACChecker creates a new RBAC checker with the given configuration.
func NewRBACChecker(config *RBACConfig) *RBACChecker {
	if config == nil {
		config = DefaultRBACConfig()
	}
	return &RBACChecker{config: config}
}

// Check evaluates whether the subject has permission for the requested operation.
func (c *RBACChecker) Check(ctx context.Context, req *prompty.AccessRequest) (*prompty.AccessDecision, error) {
	if req.Subject == nil {
		return prompty.Deny("no subject provided"), nil
	}

	// Check each role the subject has
	for _, roleStr := range req.Subject.Roles {
		role := Role(roleStr)
		permissions, exists := c.config.RolePermissions[role]
		if !exists {
			continue
		}

		// Check if any permission grants the requested operation
		for _, perm := range permissions {
			if c.operationAllowed(req.Operation, perm.Operations) {
				if c.tagsAllowed(req.Resource, perm.Tags) {
					return prompty.Allow(fmt.Sprintf("granted by role %s", role)), nil
				}
			}
		}
	}

	return prompty.Deny(fmt.Sprintf("no role grants %s permission", req.Operation)), nil
}

// BatchCheck evaluates multiple requests.
func (c *RBACChecker) BatchCheck(ctx context.Context, reqs []*prompty.AccessRequest) ([]*prompty.AccessDecision, error) {
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

func (c *RBACChecker) operationAllowed(op prompty.Operation, allowed []prompty.Operation) bool {
	for _, a := range allowed {
		if a == op {
			return true
		}
	}
	return false
}

func (c *RBACChecker) tagsAllowed(resource *prompty.StoredTemplate, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true // Empty means all tags allowed
	}
	if resource == nil {
		return true // No resource to check tags against
	}

	// Check if resource has at least one required tag
	for _, reqTag := range requiredTags {
		for _, resTag := range resource.Tags {
			if reqTag == resTag {
				return true
			}
		}
	}
	return false
}

func main() {
	ctx := context.Background()

	// Create storage and RBAC checker
	storage := prompty.NewMemoryStorage()
	rbacChecker := NewRBACChecker(DefaultRBACConfig())

	// Create audit logger to track access decisions
	auditor := prompty.NewMemoryAuditor(100)

	// Create secure storage engine with RBAC
	engine, err := prompty.NewSecureStorageEngine(prompty.SecureStorageEngineConfig{
		StorageEngineConfig: prompty.StorageEngineConfig{
			Storage: storage,
		},
		AccessChecker: rbacChecker,
		Auditor:       auditor,
	})
	if err != nil {
		log.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	// Create different users with different roles
	adminUser := prompty.NewAccessSubject("usr_admin").
		WithRoles("admin")

	editorUser := prompty.NewAccessSubject("usr_editor").
		WithRoles("editor")

	viewerUser := prompty.NewAccessSubject("usr_viewer").
		WithRoles("viewer")

	reviewerUser := prompty.NewAccessSubject("usr_reviewer").
		WithRoles("reviewer")

	// === Demo: Admin can create templates ===
	fmt.Println("=== Admin creating template ===")
	err = engine.SaveSecure(ctx, &prompty.StoredTemplate{
		Name:   "greeting",
		Source: "Hello {~prompty.var name=\"user\" default=\"World\" /~}!",
		Tags:   []string{"production"},
	}, adminUser)
	if err != nil {
		fmt.Printf("Admin create: DENIED - %v\n", err)
	} else {
		fmt.Println("Admin create: ALLOWED")
	}

	// === Demo: Editor can update templates ===
	fmt.Println("\n=== Editor updating template ===")
	err = engine.SaveSecure(ctx, &prompty.StoredTemplate{
		Name:   "greeting",
		Source: "Hi {~prompty.var name=\"user\" default=\"Friend\" /~}!",
		Tags:   []string{"production"},
	}, editorUser)
	if err != nil {
		fmt.Printf("Editor update: DENIED - %v\n", err)
	} else {
		fmt.Println("Editor update: ALLOWED")
	}

	// === Demo: Viewer can execute templates ===
	fmt.Println("\n=== Viewer executing template ===")
	result, err := engine.ExecuteSecure(ctx, "greeting", map[string]any{
		"user": "Alice",
	}, viewerUser)
	if err != nil {
		fmt.Printf("Viewer execute: DENIED - %v\n", err)
	} else {
		fmt.Printf("Viewer execute: ALLOWED - Result: %s\n", result)
	}

	// === Demo: Reviewer cannot execute templates ===
	fmt.Println("\n=== Reviewer trying to execute template ===")
	result, err = engine.ExecuteSecure(ctx, "greeting", map[string]any{
		"user": "Bob",
	}, reviewerUser)
	if err != nil {
		fmt.Printf("Reviewer execute: DENIED - %v\n", err)
	} else {
		fmt.Printf("Reviewer execute: ALLOWED - Result: %s\n", result)
	}

	// === Demo: Viewer cannot delete templates ===
	fmt.Println("\n=== Viewer trying to delete template ===")
	err = engine.DeleteSecure(ctx, "greeting", viewerUser)
	if err != nil {
		fmt.Printf("Viewer delete: DENIED - %v\n", err)
	} else {
		fmt.Println("Viewer delete: ALLOWED")
	}

	// === Demo: Admin can delete templates ===
	fmt.Println("\n=== Admin deleting template ===")
	err = engine.DeleteSecure(ctx, "greeting", adminUser)
	if err != nil {
		fmt.Printf("Admin delete: DENIED - %v\n", err)
	} else {
		fmt.Println("Admin delete: ALLOWED")
	}

	// === Show audit log ===
	fmt.Println("\n=== Audit Log ===")
	for i, event := range auditor.Events() {
		status := "ALLOWED"
		if !event.Decision.Allowed {
			status = "DENIED"
		}
		fmt.Printf("%d. [%s] %s on '%s' by %s - %s\n",
			i+1,
			status,
			event.Operation,
			event.TemplateName,
			event.Subject.ID,
			event.Decision.Reason,
		)
	}
}
