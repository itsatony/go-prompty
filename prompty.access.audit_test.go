package prompty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessAuditEvent_Builder(t *testing.T) {
	subject := NewAccessSubject("usr_123")
	decision := Allow("test")
	tmpl := &StoredTemplate{ID: "tmpl_abc", Name: "greeting", Version: 2}

	event := NewAccessAuditEvent(OpExecute, "greeting", subject, decision).
		WithRequestID("req_xyz").
		WithTemplate(tmpl).
		WithDuration(100 * time.Millisecond).
		WithError(nil).
		WithMetadata("custom", "value")

	assert.Equal(t, OpExecute, event.Operation)
	assert.Equal(t, "greeting", event.TemplateName)
	assert.Equal(t, subject, event.Subject)
	assert.Equal(t, decision, event.Decision)
	assert.Equal(t, "req_xyz", event.RequestID)
	assert.Equal(t, tmpl.ID, event.TemplateID)
	assert.Equal(t, 2, event.TemplateVersion)
	assert.Equal(t, 100*time.Millisecond, event.Duration)
	assert.Equal(t, "value", event.Metadata["custom"])
}

func TestNoOpAuditor(t *testing.T) {
	auditor := &NoOpAuditor{}
	ctx := context.Background()

	err := auditor.Log(ctx, &AccessAuditEvent{})
	assert.NoError(t, err)
}

func TestMemoryAuditor(t *testing.T) {
	ctx := context.Background()

	t.Run("stores events", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)

		_ = auditor.Log(ctx, NewAccessAuditEvent(OpRead, "a", nil, Allow("ok")))
		_ = auditor.Log(ctx, NewAccessAuditEvent(OpExecute, "b", nil, Allow("ok")))

		assert.Equal(t, 2, auditor.Count())

		events := auditor.Events()
		assert.Len(t, events, 2)
		assert.Equal(t, "a", events[0].TemplateName)
		assert.Equal(t, "b", events[1].TemplateName)
	})

	t.Run("respects limit", func(t *testing.T) {
		auditor := NewMemoryAuditor(3)

		for i := 0; i < 5; i++ {
			_ = auditor.Log(ctx, NewAccessAuditEvent(OpRead, intToStr(i), nil, Allow("ok")))
		}

		assert.Equal(t, 3, auditor.Count())

		events := auditor.Events()
		// Should have the last 3
		assert.Equal(t, "2", events[0].TemplateName)
		assert.Equal(t, "3", events[1].TemplateName)
		assert.Equal(t, "4", events[2].TemplateName)
	})

	t.Run("Clear removes all", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)
		_ = auditor.Log(ctx, NewAccessAuditEvent(OpRead, "a", nil, Allow("ok")))
		_ = auditor.Log(ctx, NewAccessAuditEvent(OpRead, "b", nil, Allow("ok")))

		auditor.Clear()
		assert.Equal(t, 0, auditor.Count())
	})

	t.Run("LastEvent returns most recent", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)

		assert.Nil(t, auditor.LastEvent())

		_ = auditor.Log(ctx, NewAccessAuditEvent(OpRead, "a", nil, Allow("ok")))
		_ = auditor.Log(ctx, NewAccessAuditEvent(OpExecute, "b", nil, Allow("ok")))

		last := auditor.LastEvent()
		require.NotNil(t, last)
		assert.Equal(t, "b", last.TemplateName)
	})

	t.Run("FilteredEvents", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)
		_ = auditor.Log(ctx, NewAccessAuditEvent(OpRead, "a", nil, Allow("ok")))
		_ = auditor.Log(ctx, NewAccessAuditEvent(OpExecute, "b", nil, Allow("ok")))
		_ = auditor.Log(ctx, NewAccessAuditEvent(OpRead, "c", nil, Deny("no")))

		// Filter for reads only
		reads := auditor.FilteredEvents(func(e *AccessAuditEvent) bool {
			return e.Operation == OpRead
		})
		assert.Len(t, reads, 2)

		// Filter for denials
		denials := auditor.FilteredEvents(func(e *AccessAuditEvent) bool {
			return !e.Decision.Allowed
		})
		assert.Len(t, denials, 1)
	})
}

func TestChannelAuditor(t *testing.T) {
	ctx := context.Background()

	t.Run("sends events to channel", func(t *testing.T) {
		ch := make(chan *AccessAuditEvent, 10)
		auditor := NewChannelAuditor(ch)

		event := NewAccessAuditEvent(OpRead, "test", nil, Allow("ok"))
		err := auditor.Log(ctx, event)
		require.NoError(t, err)

		received := <-ch
		assert.Equal(t, "test", received.TemplateName)
	})

	t.Run("drops on full channel", func(t *testing.T) {
		ch := make(chan *AccessAuditEvent, 1)
		auditor := NewChannelAuditor(ch)

		// Fill channel
		_ = auditor.Log(ctx, NewAccessAuditEvent(OpRead, "first", nil, Allow("ok")))

		// Should not block
		err := auditor.Log(ctx, NewAccessAuditEvent(OpRead, "second", nil, Allow("ok")))
		assert.NoError(t, err)

		// Only first event in channel
		received := <-ch
		assert.Equal(t, "first", received.TemplateName)
	})
}

func TestFuncAuditor(t *testing.T) {
	ctx := context.Background()

	var logged *AccessAuditEvent
	auditor := NewFuncAuditor(func(ctx context.Context, event *AccessAuditEvent) error {
		logged = event
		return nil
	})

	event := NewAccessAuditEvent(OpExecute, "test", nil, Allow("ok"))
	err := auditor.Log(ctx, event)
	require.NoError(t, err)
	assert.Equal(t, event, logged)
}

func TestMultiAuditor(t *testing.T) {
	ctx := context.Background()

	t.Run("logs to all auditors", func(t *testing.T) {
		auditor1 := NewMemoryAuditor(0)
		auditor2 := NewMemoryAuditor(0)

		multi := NewMultiAuditor(auditor1, auditor2)

		event := NewAccessAuditEvent(OpRead, "test", nil, Allow("ok"))
		err := multi.Log(ctx, event)
		require.NoError(t, err)

		assert.Equal(t, 1, auditor1.Count())
		assert.Equal(t, 1, auditor2.Count())
	})

	t.Run("continues on error", func(t *testing.T) {
		auditor1 := NewFuncAuditor(func(ctx context.Context, event *AccessAuditEvent) error {
			return errors.New("fail")
		})
		auditor2 := NewMemoryAuditor(0)

		multi := NewMultiAuditor(auditor1, auditor2)

		event := NewAccessAuditEvent(OpRead, "test", nil, Allow("ok"))
		err := multi.Log(ctx, event)
		assert.Error(t, err) // Returns last error

		// But second auditor still logged
		assert.Equal(t, 1, auditor2.Count())
	})

	t.Run("AddAuditor", func(t *testing.T) {
		auditor1 := NewMemoryAuditor(0)
		auditor2 := NewMemoryAuditor(0)

		multi := NewMultiAuditor(auditor1)
		multi.AddAuditor(auditor2)

		event := NewAccessAuditEvent(OpRead, "test", nil, Allow("ok"))
		_ = multi.Log(ctx, event)

		assert.Equal(t, 1, auditor1.Count())
		assert.Equal(t, 1, auditor2.Count())
	})
}

func TestAuditingChecker(t *testing.T) {
	ctx := context.Background()

	t.Run("logs allow decisions", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)
		inner := &AllowAllChecker{}
		checker := NewAuditingChecker(inner, auditor)

		req := NewAccessRequest(OpRead, "test", NewAccessSubject("usr_123"))
		decision, err := checker.Check(ctx, req)

		require.NoError(t, err)
		assert.True(t, decision.Allowed)

		// Give async log time to complete
		time.Sleep(10 * time.Millisecond)

		events := auditor.Events()
		require.Len(t, events, 1)
		assert.Equal(t, "test", events[0].TemplateName)
		assert.Equal(t, OpRead, events[0].Operation)
		assert.True(t, events[0].Decision.Allowed)
	})

	t.Run("logs deny decisions", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)
		inner := NewDenyAllChecker("denied")
		checker := NewAuditingChecker(inner, auditor)

		req := NewAccessRequest(OpExecute, "test", NewAccessSubject("usr_123"))
		decision, err := checker.Check(ctx, req)

		require.NoError(t, err)
		assert.False(t, decision.Allowed)

		time.Sleep(10 * time.Millisecond)

		events := auditor.Events()
		require.Len(t, events, 1)
		assert.False(t, events[0].Decision.Allowed)
	})

	t.Run("BatchCheck logs each request", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)
		inner := &AllowAllChecker{}
		checker := NewAuditingChecker(inner, auditor)

		reqs := []*AccessRequest{
			NewAccessRequest(OpRead, "a", NewAccessSubject("usr_1")),
			NewAccessRequest(OpRead, "b", NewAccessSubject("usr_2")),
		}
		_, err := checker.BatchCheck(ctx, reqs)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 2, auditor.Count())
	})
}

func TestAuditHook(t *testing.T) {
	ctx := context.Background()

	t.Run("logs after hooks", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)
		hook := AuditHook(auditor)

		// Before hook should not log
		data := NewHookData(OpExecute, "test", NewAccessSubject("usr_123"))
		err := hook(ctx, HookBeforeExecute, data)
		assert.NoError(t, err)
		assert.Equal(t, 0, auditor.Count())

		// After hook should log
		err = hook(ctx, HookAfterExecute, data)
		assert.NoError(t, err)
		assert.Equal(t, 1, auditor.Count())
	})

	t.Run("logs errors", func(t *testing.T) {
		auditor := NewMemoryAuditor(0)
		hook := AuditHook(auditor)

		data := NewHookData(OpExecute, "test", NewAccessSubject("usr_123")).
			WithError(errors.New("execution failed"))

		_ = hook(ctx, HookAfterExecute, data)

		event := auditor.LastEvent()
		require.NotNil(t, event)
		assert.False(t, event.Decision.Allowed)
		assert.Contains(t, event.Decision.Reason, "failed")
	})
}
