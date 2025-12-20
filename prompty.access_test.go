package prompty

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessSubject_NewAccessSubject(t *testing.T) {
	subject := NewAccessSubject("usr_123")

	assert.Equal(t, "usr_123", subject.ID)
	assert.Equal(t, SubjectTypeUser, subject.Type)
	assert.NotNil(t, subject.Attrs)
	assert.NotNil(t, subject.Extra)
}

func TestAccessSubject_Builder(t *testing.T) {
	subject := NewAccessSubject("usr_123").
		WithType(SubjectTypeService).
		WithTenant("tenant_abc").
		WithRoles("admin", "editor").
		WithGroups("team-a", "team-b").
		WithScopes("read", "write").
		WithAttr("department", "engineering").
		WithExtra("session_id", "sess_xyz")

	assert.Equal(t, SubjectTypeService, subject.Type)
	assert.Equal(t, "tenant_abc", subject.TenantID)
	assert.Equal(t, []string{"admin", "editor"}, subject.Roles)
	assert.Equal(t, []string{"team-a", "team-b"}, subject.Groups)
	assert.Equal(t, []string{"read", "write"}, subject.Scopes)
	assert.Equal(t, "engineering", subject.Attrs["department"])
	assert.Equal(t, "sess_xyz", subject.Extra["session_id"])
}

func TestAccessSubject_HasRole(t *testing.T) {
	subject := NewAccessSubject("usr_123").WithRoles("admin", "editor")

	assert.True(t, subject.HasRole("admin"))
	assert.True(t, subject.HasRole("editor"))
	assert.False(t, subject.HasRole("viewer"))
}

func TestAccessSubject_HasAnyRole(t *testing.T) {
	subject := NewAccessSubject("usr_123").WithRoles("admin", "editor")

	assert.True(t, subject.HasAnyRole("admin", "viewer"))
	assert.True(t, subject.HasAnyRole("viewer", "editor"))
	assert.False(t, subject.HasAnyRole("viewer", "guest"))
}

func TestAccessSubject_HasAllRoles(t *testing.T) {
	subject := NewAccessSubject("usr_123").WithRoles("admin", "editor", "viewer")

	assert.True(t, subject.HasAllRoles("admin", "editor"))
	assert.True(t, subject.HasAllRoles("admin"))
	assert.False(t, subject.HasAllRoles("admin", "superuser"))
}

func TestAccessSubject_HasScope(t *testing.T) {
	subject := NewAccessSubject("usr_123").WithScopes("read", "write")

	assert.True(t, subject.HasScope("read"))
	assert.False(t, subject.HasScope("delete"))
}

func TestAccessSubject_HasGroup(t *testing.T) {
	subject := NewAccessSubject("usr_123").WithGroups("team-a", "team-b")

	assert.True(t, subject.HasGroup("team-a"))
	assert.False(t, subject.HasGroup("team-c"))
}

func TestAccessSubject_IsAnonymous(t *testing.T) {
	t.Run("nil subject is anonymous", func(t *testing.T) {
		var subject *AccessSubject
		assert.True(t, subject.IsAnonymous())
	})

	t.Run("empty ID is anonymous", func(t *testing.T) {
		subject := &AccessSubject{}
		assert.True(t, subject.IsAnonymous())
	})

	t.Run("anonymous type is anonymous", func(t *testing.T) {
		subject := NewAccessSubject("anon").WithType(SubjectTypeAnonymous)
		assert.True(t, subject.IsAnonymous())
	})

	t.Run("regular subject is not anonymous", func(t *testing.T) {
		subject := NewAccessSubject("usr_123")
		assert.False(t, subject.IsAnonymous())
	})
}

func TestAccessRequest_Builder(t *testing.T) {
	subject := NewAccessSubject("usr_123")
	tmpl := &StoredTemplate{ID: "tmpl_abc", Name: "greeting"}

	req := NewAccessRequest(OpExecute, "greeting", subject).
		WithTemplateID("tmpl_xyz").
		WithResource(tmpl).
		WithExecutionData(map[string]any{"user": "Alice"}).
		WithMetadata("request_id", "req_123")

	assert.Equal(t, OpExecute, req.Operation)
	assert.Equal(t, "greeting", req.TemplateName)
	assert.Equal(t, subject, req.Subject)
	assert.Equal(t, tmpl.ID, req.TemplateID) // Updated by WithResource
	assert.Equal(t, "Alice", req.ExecutionData["user"])
	assert.Equal(t, "req_123", req.Metadata["request_id"])
}

func TestAccessDecision_Constructors(t *testing.T) {
	t.Run("Allow", func(t *testing.T) {
		d := Allow("test reason")
		assert.True(t, d.Allowed)
		assert.Equal(t, "test reason", d.Reason)
	})

	t.Run("Deny", func(t *testing.T) {
		d := Deny("test reason")
		assert.False(t, d.Allowed)
		assert.Equal(t, "test reason", d.Reason)
	})

	t.Run("AllowWithConditions", func(t *testing.T) {
		d := AllowWithConditions("test", "condition1", "condition2")
		assert.True(t, d.Allowed)
		assert.Equal(t, []string{"condition1", "condition2"}, d.Conditions)
	})

	t.Run("AllowWithExpiry", func(t *testing.T) {
		expiry := time.Now().Add(time.Hour)
		d := AllowWithExpiry("test", expiry)
		assert.True(t, d.Allowed)
		assert.Equal(t, expiry, *d.ExpiresAt)
	})
}

func TestAllowAllChecker(t *testing.T) {
	checker := &AllowAllChecker{}
	ctx := context.Background()

	t.Run("Check allows all", func(t *testing.T) {
		req := NewAccessRequest(OpExecute, "test", NewAccessSubject("usr_123"))
		decision, err := checker.Check(ctx, req)
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
	})

	t.Run("BatchCheck allows all", func(t *testing.T) {
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "a", nil),
			NewAccessRequest(OpExecute, "b", nil),
			NewAccessRequest(OpDelete, "c", nil),
		}
		decisions, err := checker.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		for _, d := range decisions {
			assert.True(t, d.Allowed)
		}
	})
}

func TestDenyAllChecker(t *testing.T) {
	checker := NewDenyAllChecker("maintenance mode")
	ctx := context.Background()

	t.Run("Check denies all", func(t *testing.T) {
		req := NewAccessRequest(OpExecute, "test", NewAccessSubject("usr_123"))
		decision, err := checker.Check(ctx, req)
		require.NoError(t, err)
		assert.False(t, decision.Allowed)
		assert.Equal(t, "maintenance mode", decision.Reason)
	})

	t.Run("BatchCheck denies all", func(t *testing.T) {
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "a", nil),
			NewAccessRequest(OpExecute, "b", nil),
		}
		decisions, err := checker.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		for _, d := range decisions {
			assert.False(t, d.Allowed)
		}
	})

	t.Run("default reason", func(t *testing.T) {
		checker := NewDenyAllChecker("")
		decision, _ := checker.Check(ctx, NewAccessRequest(OpRead, "test", nil))
		assert.Equal(t, "deny all checker", decision.Reason)
	})
}

func TestChainedChecker(t *testing.T) {
	ctx := context.Background()

	t.Run("all allow returns allow", func(t *testing.T) {
		checker := MustChainedChecker(&AllowAllChecker{}, &AllowAllChecker{})
		decision, err := checker.Check(ctx, NewAccessRequest(OpRead, "test", nil))
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
	})

	t.Run("first deny stops chain", func(t *testing.T) {
		checker := MustChainedChecker(
			NewDenyAllChecker("first"),
			&AllowAllChecker{},
		)
		decision, err := checker.Check(ctx, NewAccessRequest(OpRead, "test", nil))
		require.NoError(t, err)
		assert.False(t, decision.Allowed)
		assert.Equal(t, "first", decision.Reason)
	})

	t.Run("empty chain returns error", func(t *testing.T) {
		_, err := NewChainedChecker()
		require.Error(t, err)
	})

	t.Run("AddChecker", func(t *testing.T) {
		checker := MustChainedChecker(&AllowAllChecker{})
		checker.AddChecker(NewDenyAllChecker("added"))
		decision, _ := checker.Check(ctx, NewAccessRequest(OpRead, "test", nil))
		assert.False(t, decision.Allowed)
	})
}

func TestAnyOfChecker(t *testing.T) {
	ctx := context.Background()

	t.Run("any allow returns allow", func(t *testing.T) {
		checker := MustAnyOfChecker(
			NewDenyAllChecker("first"),
			&AllowAllChecker{},
		)
		decision, err := checker.Check(ctx, NewAccessRequest(OpRead, "test", nil))
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
	})

	t.Run("all deny returns deny", func(t *testing.T) {
		checker := MustAnyOfChecker(
			NewDenyAllChecker("first"),
			NewDenyAllChecker("second"),
		)
		decision, err := checker.Check(ctx, NewAccessRequest(OpRead, "test", nil))
		require.NoError(t, err)
		assert.False(t, decision.Allowed)
	})

	t.Run("empty returns error", func(t *testing.T) {
		_, err := NewAnyOfChecker()
		require.Error(t, err)
	})
}

func TestCachedChecker(t *testing.T) {
	ctx := context.Background()

	t.Run("caches decisions", func(t *testing.T) {
		callCount := 0
		inner := &countingChecker{
			allow:     true,
			onCheck:   func() { callCount++ },
		}

		checker := NewCachedChecker(inner, CachedCheckerConfig{
			TTL:        time.Hour,
			MaxEntries: 100,
		})

		req := NewAccessRequest(OpRead, "test", NewAccessSubject("usr_123"))

		// First call
		_, _ = checker.Check(ctx, req)
		assert.Equal(t, 1, callCount)

		// Second call should use cache
		_, _ = checker.Check(ctx, req)
		assert.Equal(t, 1, callCount)

		stats := checker.Stats()
		assert.Equal(t, 1, stats.ValidEntries)
	})

	t.Run("expires after TTL", func(t *testing.T) {
		callCount := 0
		inner := &countingChecker{
			allow:   true,
			onCheck: func() { callCount++ },
		}

		checker := NewCachedChecker(inner, CachedCheckerConfig{
			TTL:        1 * time.Millisecond,
			MaxEntries: 100,
		})

		req := NewAccessRequest(OpRead, "test", NewAccessSubject("usr_123"))

		_, _ = checker.Check(ctx, req)
		assert.Equal(t, 1, callCount)

		time.Sleep(10 * time.Millisecond)

		_, _ = checker.Check(ctx, req)
		assert.Equal(t, 2, callCount)
	})

	t.Run("Invalidate removes entry", func(t *testing.T) {
		inner := &AllowAllChecker{}
		checker := NewCachedChecker(inner, DefaultCachedCheckerConfig())

		req := NewAccessRequest(OpRead, "test", NewAccessSubject("usr_123"))
		_, _ = checker.Check(ctx, req)

		assert.Equal(t, 1, checker.Stats().Entries)

		checker.Invalidate(req)
		assert.Equal(t, 0, checker.Stats().Entries)
	})

	t.Run("InvalidateAll clears cache", func(t *testing.T) {
		inner := &AllowAllChecker{}
		checker := NewCachedChecker(inner, DefaultCachedCheckerConfig())

		for i := 0; i < 5; i++ {
			req := NewAccessRequest(OpRead, "test"+intToStr(i), NewAccessSubject("usr_123"))
			_, _ = checker.Check(ctx, req)
		}

		assert.Equal(t, 5, checker.Stats().Entries)

		checker.InvalidateAll()
		assert.Equal(t, 0, checker.Stats().Entries)
	})
}

func TestOperationChecker(t *testing.T) {
	ctx := context.Background()
	checker := NewOperationChecker(OpRead, OpList)

	t.Run("allows configured operations", func(t *testing.T) {
		decision, _ := checker.Check(ctx, NewAccessRequest(OpRead, "test", nil))
		assert.True(t, decision.Allowed)

		decision, _ = checker.Check(ctx, NewAccessRequest(OpList, "test", nil))
		assert.True(t, decision.Allowed)
	})

	t.Run("denies other operations", func(t *testing.T) {
		decision, _ := checker.Check(ctx, NewAccessRequest(OpExecute, "test", nil))
		assert.False(t, decision.Allowed)

		decision, _ = checker.Check(ctx, NewAccessRequest(OpDelete, "test", nil))
		assert.False(t, decision.Allowed)
	})
}

func TestTenantChecker(t *testing.T) {
	ctx := context.Background()

	t.Run("denies missing tenant", func(t *testing.T) {
		checker := NewTenantChecker()
		decision, _ := checker.Check(ctx, NewAccessRequest(OpRead, "test", NewAccessSubject("usr_123")))
		assert.False(t, decision.Allowed)
	})

	t.Run("allows matching tenant", func(t *testing.T) {
		checker := NewTenantChecker()
		subject := NewAccessSubject("usr_123").WithTenant("tenant_a")
		tmpl := &StoredTemplate{Name: "test", TenantID: "tenant_a"}
		req := NewAccessRequest(OpRead, "test", subject).WithResource(tmpl)

		decision, _ := checker.Check(ctx, req)
		assert.True(t, decision.Allowed)
	})

	t.Run("denies mismatched tenant", func(t *testing.T) {
		checker := NewTenantChecker()
		subject := NewAccessSubject("usr_123").WithTenant("tenant_a")
		tmpl := &StoredTemplate{Name: "test", TenantID: "tenant_b"}
		req := NewAccessRequest(OpRead, "test", subject).WithResource(tmpl)

		decision, _ := checker.Check(ctx, req)
		assert.False(t, decision.Allowed)
	})

	t.Run("system tenant can access all", func(t *testing.T) {
		checker := NewTenantChecker().WithSystemTenant("system")
		subject := NewAccessSubject("usr_123").WithTenant("system")
		tmpl := &StoredTemplate{Name: "test", TenantID: "tenant_b"}
		req := NewAccessRequest(OpRead, "test", subject).WithResource(tmpl)

		decision, _ := checker.Check(ctx, req)
		assert.True(t, decision.Allowed)
	})
}

func TestRoleChecker(t *testing.T) {
	ctx := context.Background()

	t.Run("allows with required role", func(t *testing.T) {
		checker := NewRoleChecker().
			WithOperationRoles(OpExecute, "executor", "admin")

		subject := NewAccessSubject("usr_123").WithRoles("executor")
		decision, _ := checker.Check(ctx, NewAccessRequest(OpExecute, "test", subject))
		assert.True(t, decision.Allowed)
	})

	t.Run("denies without required role", func(t *testing.T) {
		checker := NewRoleChecker().
			WithOperationRoles(OpExecute, "admin")

		subject := NewAccessSubject("usr_123").WithRoles("viewer")
		decision, _ := checker.Check(ctx, NewAccessRequest(OpExecute, "test", subject))
		assert.False(t, decision.Allowed)
	})

	t.Run("uses default roles", func(t *testing.T) {
		checker := NewRoleChecker().WithDefaultRoles("authenticated")

		subject := NewAccessSubject("usr_123").WithRoles("authenticated")
		decision, _ := checker.Check(ctx, NewAccessRequest(OpRead, "test", subject))
		assert.True(t, decision.Allowed)
	})

	t.Run("allows when no roles required", func(t *testing.T) {
		checker := NewRoleChecker()
		decision, _ := checker.Check(ctx, NewAccessRequest(OpRead, "test", NewAccessSubject("usr_123")))
		assert.True(t, decision.Allowed)
	})

	t.Run("requires all roles when configured", func(t *testing.T) {
		checker := NewRoleChecker().
			WithOperationRoles(OpExecute, "admin", "executor")
		checker.RequireAllRoles = true

		// Has only one role
		subject := NewAccessSubject("usr_123").WithRoles("admin")
		decision, _ := checker.Check(ctx, NewAccessRequest(OpExecute, "test", subject))
		assert.False(t, decision.Allowed)

		// Has both roles
		subject = NewAccessSubject("usr_123").WithRoles("admin", "executor")
		decision, _ = checker.Check(ctx, NewAccessRequest(OpExecute, "test", subject))
		assert.True(t, decision.Allowed)
	})
}

func TestAccessError(t *testing.T) {
	t.Run("error message formatting", func(t *testing.T) {
		err := &AccessError{
			Message:   "access denied",
			Operation: OpExecute,
			Template:  "greeting",
		}
		assert.Contains(t, err.Error(), "access denied")
		assert.Contains(t, err.Error(), "greeting")
		assert.Contains(t, err.Error(), "execute")
	})

	t.Run("with cause", func(t *testing.T) {
		cause := &AccessError{Message: "inner error"}
		err := &AccessError{
			Message: "outer error",
			Cause:   cause,
		}
		assert.Contains(t, err.Error(), "outer error")
		assert.Contains(t, err.Error(), "inner error")
		assert.Equal(t, cause, err.Unwrap())
	})
}

// countingChecker is a test helper that counts calls.
type countingChecker struct {
	allow   bool
	onCheck func()
}

func (c *countingChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	if c.onCheck != nil {
		c.onCheck()
	}
	if c.allow {
		return Allow("counting checker"), nil
	}
	return Deny("counting checker"), nil
}

func (c *countingChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i := range reqs {
		decisions[i], _ = c.Check(ctx, reqs[i])
	}
	return decisions, nil
}
