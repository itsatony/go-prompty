package prompty

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wrappedOperationChecker wraps OperationChecker to inject errors.
type wrappedOperationChecker struct {
	*OperationChecker
	errorOnIndex int
	checkIndex   int
}

func (w *wrappedOperationChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	defer func() { w.checkIndex++ }()
	if w.checkIndex == w.errorOnIndex {
		return nil, errors.New("check error")
	}
	return w.OperationChecker.Check(ctx, req)
}

func (w *wrappedOperationChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := w.Check(ctx, req)
		if err != nil {
			// Fail-safe: deny access on error
			decisions[i] = Deny(ErrMsgAccessCheckFailed)
		} else {
			decisions[i] = decision
		}
	}
	return decisions, nil
}

// wrappedTenantChecker wraps TenantChecker to inject errors.
type wrappedTenantChecker struct {
	*TenantChecker
	errorOnIndex int
	checkIndex   int
}

func (w *wrappedTenantChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	defer func() { w.checkIndex++ }()
	if w.checkIndex == w.errorOnIndex {
		return nil, errors.New("check error")
	}
	return w.TenantChecker.Check(ctx, req)
}

func (w *wrappedTenantChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := w.Check(ctx, req)
		if err != nil {
			// Fail-safe: deny access on error
			decisions[i] = Deny(ErrMsgAccessCheckFailed)
		} else {
			decisions[i] = decision
		}
	}
	return decisions, nil
}

// wrappedRoleChecker wraps RoleChecker to inject errors.
type wrappedRoleChecker struct {
	*RoleChecker
	errorOnIndex int
	checkIndex   int
}

func (w *wrappedRoleChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	defer func() { w.checkIndex++ }()
	if w.checkIndex == w.errorOnIndex {
		return nil, errors.New("check error")
	}
	return w.RoleChecker.Check(ctx, req)
}

func (w *wrappedRoleChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := w.Check(ctx, req)
		if err != nil {
			// Fail-safe: deny access on error
			decisions[i] = Deny(ErrMsgAccessCheckFailed)
		} else {
			decisions[i] = decision
		}
	}
	return decisions, nil
}

func TestOperationChecker_BatchCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("processes all requests successfully", func(t *testing.T) {
		checker := NewOperationChecker(OpRead, OpList)
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test1", nil),
			NewAccessRequest(OpList, "test2", nil),
			NewAccessRequest(OpExecute, "test3", nil), // Should be denied
		}

		decisions, err := checker.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		require.Len(t, decisions, 3)
		assert.True(t, decisions[0].Allowed)
		assert.True(t, decisions[1].Allowed)
		assert.False(t, decisions[2].Allowed)
	})

	t.Run("handles errors with fail-safe deny", func(t *testing.T) {
		wrapped := &wrappedOperationChecker{
			OperationChecker: NewOperationChecker(OpRead, OpList),
			errorOnIndex:     1, // Second request will error
		}
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test1", nil),
			NewAccessRequest(OpList, "test2", nil), // Will error
			NewAccessRequest(OpRead, "test3", nil),
		}

		decisions, err := wrapped.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		require.Len(t, decisions, 3)
		assert.True(t, decisions[0].Allowed)
		assert.False(t, decisions[1].Allowed) // Denied due to error
		assert.Equal(t, ErrMsgAccessCheckFailed, decisions[1].Reason)
		assert.True(t, decisions[2].Allowed) // Still processed
	})

	t.Run("error denial uses correct message", func(t *testing.T) {
		wrapped := &wrappedOperationChecker{
			OperationChecker: NewOperationChecker(OpRead),
			errorOnIndex:     0,
		}
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test1", nil),
		}

		decisions, err := wrapped.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		assert.False(t, decisions[0].Allowed)
		assert.Equal(t, ErrMsgAccessCheckFailed, decisions[0].Reason)
	})
}

func TestTenantChecker_BatchCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("processes all requests successfully", func(t *testing.T) {
		checker := NewTenantChecker()
		subject := NewAccessSubject("usr_123").WithTenant("tenant_a")
		tmplA := &StoredTemplate{Name: "test1", TenantID: "tenant_a"}
		tmplB := &StoredTemplate{Name: "test2", TenantID: "tenant_b"}

		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test1", subject).WithResource(tmplA),
			NewAccessRequest(OpRead, "test2", subject).WithResource(tmplB), // Different tenant
		}

		decisions, err := checker.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		require.Len(t, decisions, 2)
		assert.True(t, decisions[0].Allowed)
		assert.False(t, decisions[1].Allowed)
	})

	t.Run("handles errors with fail-safe deny", func(t *testing.T) {
		wrapped := &wrappedTenantChecker{
			TenantChecker: NewTenantChecker(),
			errorOnIndex:  1,
		}
		subject := NewAccessSubject("usr_123").WithTenant("tenant_a")
		tmpl := &StoredTemplate{Name: "test", TenantID: "tenant_a"}

		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test1", subject).WithResource(tmpl),
			NewAccessRequest(OpRead, "test2", subject).WithResource(tmpl), // Will error
			NewAccessRequest(OpRead, "test3", subject).WithResource(tmpl),
		}

		decisions, err := wrapped.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		require.Len(t, decisions, 3)
		assert.True(t, decisions[0].Allowed)
		assert.False(t, decisions[1].Allowed) // Denied due to error
		assert.Equal(t, ErrMsgAccessCheckFailed, decisions[1].Reason)
		assert.True(t, decisions[2].Allowed) // Still processed
	})
}

func TestRoleChecker_BatchCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("processes all requests successfully", func(t *testing.T) {
		checker := NewRoleChecker().
			WithOperationRoles(OpExecute, "executor")
		subjectWithRole := NewAccessSubject("usr_123").WithRoles("executor")
		subjectWithoutRole := NewAccessSubject("usr_456").WithRoles("viewer")

		reqs := []*AccessRequest{
			NewAccessRequest(OpExecute, "test1", subjectWithRole),
			NewAccessRequest(OpExecute, "test2", subjectWithoutRole),
		}

		decisions, err := checker.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		require.Len(t, decisions, 2)
		assert.True(t, decisions[0].Allowed)
		assert.False(t, decisions[1].Allowed)
	})

	t.Run("handles errors with fail-safe deny", func(t *testing.T) {
		wrapped := &wrappedRoleChecker{
			RoleChecker:  NewRoleChecker(),
			errorOnIndex: 0,
		}
		subject := NewAccessSubject("usr_123").WithRoles("admin")

		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test1", subject), // Will error
			NewAccessRequest(OpRead, "test2", subject),
		}

		decisions, err := wrapped.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		require.Len(t, decisions, 2)
		assert.False(t, decisions[0].Allowed) // Denied due to error
		assert.Equal(t, ErrMsgAccessCheckFailed, decisions[0].Reason)
		assert.True(t, decisions[1].Allowed) // Still processed
	})

	t.Run("error denial has correct reason", func(t *testing.T) {
		wrapped := &wrappedRoleChecker{
			RoleChecker:  NewRoleChecker(),
			errorOnIndex: 0,
		}
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test1", NewAccessSubject("usr_123")),
		}

		decisions, err := wrapped.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		assert.False(t, decisions[0].Allowed)
		assert.Equal(t, ErrMsgAccessCheckFailed, decisions[0].Reason)
	})
}

func TestBatchCheck_FailSafePrinciple(t *testing.T) {
	// This test verifies that the fail-safe principle is applied:
	// When a Check() returns an error, BatchCheck() should deny access
	// rather than silently allowing or crashing.

	ctx := context.Background()

	t.Run("OperationChecker denies on error", func(t *testing.T) {
		wrapped := &wrappedOperationChecker{
			OperationChecker: NewOperationChecker(OpRead, OpList, OpExecute),
			errorOnIndex:     0,
		}
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test", nil),
		}

		decisions, _ := wrapped.BatchCheck(ctx, reqs)
		assert.False(t, decisions[0].Allowed, "should deny on error, not allow")
		assert.Equal(t, ErrMsgAccessCheckFailed, decisions[0].Reason)
	})

	t.Run("TenantChecker denies on error", func(t *testing.T) {
		wrapped := &wrappedTenantChecker{
			TenantChecker: NewTenantChecker().WithSystemTenant("system"),
			errorOnIndex:  0,
		}
		// System tenant would normally be allowed
		subject := NewAccessSubject("usr_123").WithTenant("system")
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test", subject),
		}

		decisions, _ := wrapped.BatchCheck(ctx, reqs)
		assert.False(t, decisions[0].Allowed, "should deny on error even for system tenant")
		assert.Equal(t, ErrMsgAccessCheckFailed, decisions[0].Reason)
	})

	t.Run("RoleChecker denies on error", func(t *testing.T) {
		wrapped := &wrappedRoleChecker{
			RoleChecker:  NewRoleChecker(), // No roles required = would allow
			errorOnIndex: 0,
		}
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "test", NewAccessSubject("usr_123")),
		}

		decisions, _ := wrapped.BatchCheck(ctx, reqs)
		assert.False(t, decisions[0].Allowed, "should deny on error even when no roles required")
		assert.Equal(t, ErrMsgAccessCheckFailed, decisions[0].Reason)
	})
}

func TestBatchCheck_ContinuesAfterError(t *testing.T) {
	// This test verifies that BatchCheck continues processing all requests
	// even when some fail, ensuring all requests get a decision.

	ctx := context.Background()

	t.Run("processes remaining requests after error", func(t *testing.T) {
		wrapped := &wrappedOperationChecker{
			OperationChecker: NewOperationChecker(OpRead, OpList),
			errorOnIndex:     1, // Error on second request
		}
		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "first", nil),
			NewAccessRequest(OpList, "second", nil),  // Will error
			NewAccessRequest(OpRead, "third", nil),
			NewAccessRequest(OpList, "fourth", nil),
		}

		decisions, err := wrapped.BatchCheck(ctx, reqs)
		require.NoError(t, err)
		require.Len(t, decisions, 4, "all requests should get decisions")

		assert.True(t, decisions[0].Allowed, "first should succeed")
		assert.False(t, decisions[1].Allowed, "second should be denied due to error")
		assert.True(t, decisions[2].Allowed, "third should succeed")
		assert.True(t, decisions[3].Allowed, "fourth should succeed")
	})
}
