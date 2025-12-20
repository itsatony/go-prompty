package prompty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecureStorageEngine_New(t *testing.T) {
	t.Run("creates with default checker", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, err := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, se)
		defer se.Close()

		assert.NotNil(t, se.AccessChecker())
		assert.NotNil(t, se.Hooks())
	})

	t.Run("creates with custom checker", func(t *testing.T) {
		storage := NewMemoryStorage()
		checker := NewDenyAllChecker("custom")

		se, err := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			AccessChecker: checker,
		})
		require.NoError(t, err)
		defer se.Close()

		// Checker should deny
		ctx := context.Background()
		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

		_, err = se.ExecuteSecure(ctx, "test", nil, NewAccessSubject("usr_123"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("creates with auditor", func(t *testing.T) {
		storage := NewMemoryStorage()
		auditor := NewMemoryAuditor(0)

		se, err := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			Auditor: auditor,
		})
		require.NoError(t, err)
		defer se.Close()

		// Perform an operation
		ctx := context.Background()
		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})
		_, _ = se.ExecuteSecure(ctx, "test", nil, NewAccessSubject("usr_123"))

		// Wait for async audit
		// In real test would use synchronous auditing
	})

	t.Run("panics on nil storage", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewSecureStorageEngine(SecureStorageEngineConfig{
				StorageEngineConfig: StorageEngineConfig{
					Storage: nil,
				},
			})
		})
	})
}

func TestSecureStorageEngine_ExecuteSecure(t *testing.T) {
	ctx := context.Background()

	t.Run("allows when checker allows", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			AccessChecker: &AllowAllChecker{},
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{
			Name:   "greeting",
			Source: "Hello {~prompty.var name=\"user\" default=\"World\" /~}!",
		})

		result, err := se.ExecuteSecure(ctx, "greeting", map[string]any{"user": "Alice"}, NewAccessSubject("usr_123"))
		require.NoError(t, err)
		assert.Equal(t, "Hello Alice!", result)
	})

	t.Run("denies when checker denies", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			AccessChecker: NewDenyAllChecker("no execute"),
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

		_, err := se.ExecuteSecure(ctx, "test", nil, NewAccessSubject("usr_123"))
		require.Error(t, err)
	})

	t.Run("runs hooks", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

		var hooksCalled []string
		se.RegisterHook(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			hooksCalled = append(hooksCalled, "before")
			return nil
		})
		se.RegisterHook(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
			hooksCalled = append(hooksCalled, "after")
			return nil
		})

		_, _ = se.ExecuteSecure(ctx, "test", nil, NewAccessSubject("usr_123"))

		assert.Equal(t, []string{"before", "after"}, hooksCalled)
	})
}

func TestSecureStorageEngine_GetSecure(t *testing.T) {
	ctx := context.Background()

	t.Run("allows read when checker allows", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

		tmpl, err := se.GetSecure(ctx, "test", NewAccessSubject("usr_123"))
		require.NoError(t, err)
		assert.Equal(t, "content", tmpl.Source)
	})

	t.Run("denies when checker denies", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			AccessChecker: NewDenyAllChecker("no read"),
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

		_, err := se.GetSecure(ctx, "test", NewAccessSubject("usr_123"))
		require.Error(t, err)
	})
}

func TestSecureStorageEngine_SaveSecure(t *testing.T) {
	ctx := context.Background()

	t.Run("allows create when checker allows", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		err := se.SaveSecure(ctx, &StoredTemplate{Name: "test", Source: "content"}, NewAccessSubject("usr_123"))
		require.NoError(t, err)

		exists, _ := se.Exists(ctx, "test")
		assert.True(t, exists)
	})

	t.Run("allows update when checker allows", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})

		err := se.SaveSecure(ctx, &StoredTemplate{Name: "test", Source: "v2"}, NewAccessSubject("usr_123"))
		require.NoError(t, err)

		versions, _ := se.ListVersions(ctx, "test")
		assert.Len(t, versions, 2)
	})

	t.Run("denies when checker denies", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			AccessChecker: NewDenyAllChecker("no create"),
		})
		defer se.Close()

		err := se.SaveSecure(ctx, &StoredTemplate{Name: "test", Source: "content"}, NewAccessSubject("usr_123"))
		require.Error(t, err)
	})
}

func TestSecureStorageEngine_DeleteSecure(t *testing.T) {
	ctx := context.Background()

	t.Run("allows delete when checker allows", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

		err := se.DeleteSecure(ctx, "test", NewAccessSubject("usr_123"))
		require.NoError(t, err)

		exists, _ := se.Exists(ctx, "test")
		assert.False(t, exists)
	})

	t.Run("denies when checker denies", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			AccessChecker: NewDenyAllChecker("no delete"),
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

		err := se.DeleteSecure(ctx, "test", NewAccessSubject("usr_123"))
		require.Error(t, err)

		// Should still exist
		exists, _ := se.Exists(ctx, "test")
		assert.True(t, exists)
	})
}

func TestSecureStorageEngine_ListSecure(t *testing.T) {
	ctx := context.Background()

	t.Run("returns only accessible templates", func(t *testing.T) {
		storage := NewMemoryStorage()

		// Checker that allows access to templates with "public" in name
		checker := &funcChecker{
			checkFn: func(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
				if req.Operation == OpList {
					return Allow("list allowed"), nil
				}
				if req.Resource != nil {
					for _, tag := range req.Resource.Tags {
						if tag == "public" {
							return Allow("public template"), nil
						}
					}
				}
				return Deny("not public"), nil
			},
		}

		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			AccessChecker: checker,
		})
		defer se.Close()

		// Save some templates
		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "public1", Source: "p1", Tags: []string{"public"}})
		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "public2", Source: "p2", Tags: []string{"public"}})
		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "private", Source: "pr", Tags: []string{"internal"}})

		// List should only return public templates
		results, err := se.ListSecure(ctx, nil, NewAccessSubject("usr_123"))
		require.NoError(t, err)
		assert.Len(t, results, 2)

		for _, tmpl := range results {
			assert.Contains(t, tmpl.Tags, "public")
		}
	})
}

func TestSecureStorageEngine_ValidateSecure(t *testing.T) {
	ctx := context.Background()

	t.Run("allows validate when checker allows", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		_ = se.StorageEngine.Save(ctx, &StoredTemplate{Name: "test", Source: "valid content"})

		result, err := se.ValidateSecure(ctx, "test", NewAccessSubject("usr_123"))
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})
}

func TestSecureStorageEngine_CheckAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("returns decision from checker", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
			AccessChecker: NewDenyAllChecker("test reason"),
		})
		defer se.Close()

		req := NewAccessRequest(OpExecute, "test", NewAccessSubject("usr_123"))
		decision, err := se.CheckAccess(ctx, req)

		require.NoError(t, err)
		assert.False(t, decision.Allowed)
		assert.Equal(t, "test reason", decision.Reason)
	})
}

func TestSecureStorageEngine_Hooks(t *testing.T) {
	t.Run("RegisterHooks registers for multiple points", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		callCount := 0
		hook := func(ctx context.Context, point HookPoint, data *HookData) error {
			callCount++
			return nil
		}

		se.RegisterHooks(hook, HookBeforeLoad, HookAfterLoad)

		assert.True(t, se.Hooks().HasHooks(HookBeforeLoad))
		assert.True(t, se.Hooks().HasHooks(HookAfterLoad))
	})

	t.Run("ClearHooks removes specific hooks", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		se.RegisterHook(HookBeforeLoad, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})
		se.RegisterHook(HookAfterLoad, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})

		se.ClearHooks(HookBeforeLoad)

		assert.False(t, se.Hooks().HasHooks(HookBeforeLoad))
		assert.True(t, se.Hooks().HasHooks(HookAfterLoad))
	})

	t.Run("ClearAllHooks removes all hooks", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, _ := NewSecureStorageEngine(SecureStorageEngineConfig{
			StorageEngineConfig: StorageEngineConfig{
				Storage: storage,
			},
		})
		defer se.Close()

		se.RegisterHook(HookBeforeLoad, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})
		se.RegisterHook(HookAfterLoad, func(ctx context.Context, point HookPoint, data *HookData) error {
			return nil
		})

		se.ClearAllHooks()

		assert.False(t, se.Hooks().HasHooks(HookBeforeLoad))
		assert.False(t, se.Hooks().HasHooks(HookAfterLoad))
	})
}

// funcChecker wraps a function as an AccessChecker for testing.
type funcChecker struct {
	checkFn func(ctx context.Context, req *AccessRequest) (*AccessDecision, error)
}

func (c *funcChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	return c.checkFn(ctx, req)
}

func (c *funcChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		var err error
		decisions[i], err = c.Check(ctx, req)
		if err != nil {
			return nil, err
		}
	}
	return decisions, nil
}
