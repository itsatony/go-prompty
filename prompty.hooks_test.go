package prompty

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHookData_Builder(t *testing.T) {
	subject := NewAccessSubject("usr_123")
	tmpl := &StoredTemplate{Name: "greeting"}

	data := NewHookData(OpExecute, "greeting", subject).
		WithTemplate(tmpl).
		WithExecutionData(map[string]any{"user": "Alice"}).
		WithResult("Hello Alice").
		WithError(nil)

	assert.Equal(t, OpExecute, data.Operation)
	assert.Equal(t, "greeting", data.TemplateName)
	assert.Equal(t, subject, data.Subject)
	assert.Equal(t, tmpl, data.Template)
	assert.Equal(t, "Alice", data.ExecutionData["user"])
	assert.Equal(t, "Hello Alice", data.Result)
}

func TestHookData_Metadata(t *testing.T) {
	data := NewHookData(OpRead, "test", nil)

	data.SetMetadata("key1", "value1")
	data.SetMetadata("key2", 42)

	v1, ok := data.GetMetadata("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", v1)

	v2, ok := data.GetMetadata("key2")
	assert.True(t, ok)
	assert.Equal(t, 42, v2)

	_, ok = data.GetMetadata("missing")
	assert.False(t, ok)
}

func TestHookRegistry_Register(t *testing.T) {
	registry := NewHookRegistry()

	called := false
	hook := func(ctx context.Context, point HookPoint, data *HookData) error {
		called = true
		return nil
	}

	registry.Register(HookBeforeExecute, hook)

	assert.Equal(t, 1, registry.Count(HookBeforeExecute))
	assert.True(t, registry.HasHooks(HookBeforeExecute))
	assert.False(t, registry.HasHooks(HookAfterExecute))

	_ = registry.Run(context.Background(), HookBeforeExecute, &HookData{})
	assert.True(t, called)
}

func TestHookRegistry_RegisterMultiple(t *testing.T) {
	registry := NewHookRegistry()

	callCount := 0
	hook := func(ctx context.Context, point HookPoint, data *HookData) error {
		callCount++
		return nil
	}

	registry.RegisterMultiple(hook, HookBeforeLoad, HookAfterLoad, HookBeforeExecute)

	assert.Equal(t, 1, registry.Count(HookBeforeLoad))
	assert.Equal(t, 1, registry.Count(HookAfterLoad))
	assert.Equal(t, 1, registry.Count(HookBeforeExecute))

	ctx := context.Background()
	data := &HookData{}
	_ = registry.Run(ctx, HookBeforeLoad, data)
	_ = registry.Run(ctx, HookAfterLoad, data)
	_ = registry.Run(ctx, HookBeforeExecute, data)

	assert.Equal(t, 3, callCount)
}

func TestHookRegistry_Clear(t *testing.T) {
	registry := NewHookRegistry()

	registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return nil
	})
	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return nil
	})

	assert.Equal(t, 1, registry.Count(HookBeforeExecute))
	assert.Equal(t, 1, registry.Count(HookAfterExecute))

	registry.Clear(HookBeforeExecute)
	assert.Equal(t, 0, registry.Count(HookBeforeExecute))
	assert.Equal(t, 1, registry.Count(HookAfterExecute))
}

func TestHookRegistry_ClearAll(t *testing.T) {
	registry := NewHookRegistry()

	registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return nil
	})
	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return nil
	})

	registry.ClearAll()
	assert.Equal(t, 0, registry.Count(HookBeforeExecute))
	assert.Equal(t, 0, registry.Count(HookAfterExecute))
}

func TestHookRegistry_BeforeHooksAbortOnError(t *testing.T) {
	registry := NewHookRegistry()

	callOrder := []string{}

	registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		callOrder = append(callOrder, "first")
		return errors.New("abort")
	})
	registry.Register(HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		callOrder = append(callOrder, "second")
		return nil
	})

	err := registry.Run(context.Background(), HookBeforeExecute, &HookData{})
	require.Error(t, err)
	assert.Equal(t, []string{"first"}, callOrder)
}

func TestHookRegistry_AfterHooksContinueOnError(t *testing.T) {
	registry := NewHookRegistry()

	callOrder := []string{}

	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		callOrder = append(callOrder, "first")
		return errors.New("error1")
	})
	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		callOrder = append(callOrder, "second")
		return nil
	})

	err := registry.Run(context.Background(), HookAfterExecute, &HookData{})
	require.NoError(t, err) // After hooks don't return errors from Run
	assert.Equal(t, []string{"first", "second"}, callOrder)
}

func TestHookRegistry_RunWithErrors(t *testing.T) {
	registry := NewHookRegistry()

	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return errors.New("error1")
	})
	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return errors.New("error2")
	})
	registry.Register(HookAfterExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
		return nil
	})

	errs := registry.RunWithErrors(context.Background(), HookAfterExecute, &HookData{})
	assert.Len(t, errs, 2)
}

func TestLoggingHook(t *testing.T) {
	var logged []string

	hook := LoggingHook(func(point HookPoint, data *HookData) {
		logged = append(logged, string(point)+":"+data.TemplateName)
	})

	registry := NewHookRegistry()
	registry.Register(HookBeforeExecute, hook)
	registry.Register(HookAfterExecute, hook)

	ctx := context.Background()
	data := NewHookData(OpExecute, "greeting", nil)

	_ = registry.Run(ctx, HookBeforeExecute, data)
	_ = registry.Run(ctx, HookAfterExecute, data)

	assert.Equal(t, []string{
		"before_execute:greeting",
		"after_execute:greeting",
	}, logged)
}

func TestTimingHook(t *testing.T) {
	hook, getElapsed := TimingHook()

	registry := NewHookRegistry()
	registry.Register(HookBeforeExecute, hook)

	ctx := context.Background()
	data := NewHookData(OpExecute, "test", nil)

	// Before hook sets start time
	_ = registry.Run(ctx, HookBeforeExecute, data)

	// Should have elapsed time
	elapsed := getElapsed(data)
	assert.Greater(t, elapsed, int64(0))
}

func TestAccessCheckHook(t *testing.T) {
	ctx := context.Background()

	t.Run("allows when checker allows", func(t *testing.T) {
		hook := AccessCheckHook(&AllowAllChecker{})
		data := NewHookData(OpExecute, "test", NewAccessSubject("usr_123"))

		err := hook(ctx, HookBeforeExecute, data)
		assert.NoError(t, err)
	})

	t.Run("denies when checker denies", func(t *testing.T) {
		hook := AccessCheckHook(NewDenyAllChecker("denied"))
		data := NewHookData(OpExecute, "test", NewAccessSubject("usr_123"))

		err := hook(ctx, HookBeforeExecute, data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("skips after hooks", func(t *testing.T) {
		hook := AccessCheckHook(NewDenyAllChecker("denied"))
		data := NewHookData(OpExecute, "test", NewAccessSubject("usr_123"))

		// After hooks should be skipped
		err := hook(ctx, HookAfterExecute, data)
		assert.NoError(t, err)
	})
}

func TestHookError(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewHookError(HookBeforeExecute, cause)

	assert.Contains(t, err.Error(), "hook execution failed")
	assert.Contains(t, err.Error(), "before_execute")
	assert.Contains(t, err.Error(), "underlying error")
	assert.Equal(t, cause, err.Unwrap())
}

func TestIsBeforeHook(t *testing.T) {
	beforeHooks := []HookPoint{
		HookBeforeLoad,
		HookBeforeExecute,
		HookBeforeSave,
		HookBeforeDelete,
		HookBeforeValidate,
	}

	afterHooks := []HookPoint{
		HookAfterLoad,
		HookAfterExecute,
		HookAfterSave,
		HookAfterDelete,
		HookAfterValidate,
	}

	for _, point := range beforeHooks {
		assert.True(t, isBeforeHook(point), "expected %s to be before hook", point)
	}

	for _, point := range afterHooks {
		assert.False(t, isBeforeHook(point), "expected %s to be after hook", point)
	}
}
