# Project Assessment — go-prompty

**Date**: 2026-02-10
**Version**: v2.4.0
**Language**: Go 1.24
**Standards**: `~/.claude/standards/golang.md` (v2026-02-07) + project `CLAUDE.md`
**Assessor**: Claude Code (/assess)
**Focus**: Full assessment

---

## Executive Summary

- **Build, vet, and deps are clean.** Zero build errors, zero vet issues, zero stale dependencies.
- **Test suite is comprehensive and healthy.** 3215 tests, 100% passing, no race conditions detected. Root package at 80.3% coverage.
- **One CRITICAL standards violation:** `err == context.Canceled` in expression evaluator (should use `errors.Is()`).
- **Several HIGH findings:** `fmt.Errorf` usage in postgres storage and inheritance parser should migrate to `cuserr` wrappers.
- **Missing `ADRs.md`** per golang.md standards requirement.

---

## Metrics

| Metric | Value |
|--------|-------|
| Build | PASS |
| Vet issues | 0 |
| Lint issues | 2 (errcheck in test file) |
| Test count | 3215 |
| Tests passing | 3215 |
| Tests failed | 0 |
| Coverage (combined) | 72.4% |
| Coverage (root) | 80.3% |
| Coverage (internal) | 91.1% |
| Coverage (cmd) | 91.3% |
| TODO/FIXME count | 0 |
| Standards violations | 2 (errors.Is, missing ADRs.md) |
| Stale deps | 0 |

*First assessment — no previous data for trend comparison.*

---

## Health Scorecard

| Category | Score | Evidence |
|----------|-------|----------|
| Build & Lint | 9/10 | Zero build/vet issues. 2 minor errcheck findings in test code only. |
| Test Health | 9/10 | 3215/3215 passing, race-clean. 80.3% root coverage (exceeds 80% threshold). 5 functions below 50% but all are edge-case code paths. |
| Code Quality | 7/10 | 1 CRITICAL (error comparison), 4 HIGH (fmt.Errorf inconsistency), 3 MEDIUM, 4 LOW. Total: 12 findings from code review. |
| Standards Compliance | 8/10 | 9/11 checks passed. 2 violations: `errors.Is()` usage, missing ADRs.md. Magic strings audit (v2.4.0) was thorough. |
| Production Readiness | 8/10 | No security issues, no secrets in code, proper mutex usage across 17 files. Missing ADRs.md is documentation gap only. |
| **Overall** | **8.2/10** | **Weighted: build 15% (9), test 25% (9), quality 25% (7), standards 20% (8), prod 15% (8)** |

---

## Critical Issues

1. **[CRITICAL] `internal/prompty.expr.evaluator.go:79,82`** — Uses `err == context.Canceled` and `err == context.DeadlineExceeded` instead of `errors.Is()`. These sentinel values should always be compared with `errors.Is()` to handle wrapped errors correctly.
   - **Fix:** Replace `err == context.Canceled` with `errors.Is(err, context.Canceled)` and same for `DeadlineExceeded`.

2. **[HIGH] `prompty.storage.postgres.go:1071,1079,1087`** — Uses `fmt.Errorf` with constant prefixes for unmarshal errors instead of `cuserr.WrapStdError()`.
   - **Fix:** Replace with `cuserr.WrapStdError(err, ErrCodeStorage, ErrMsgPostgresUnmarshalFailed).WithMetadata("field", "metadata")`.

3. **[HIGH] `prompty.storage.postgres.go:139,153,811`** — Uses `fmt.Errorf` for error wrapping in connection close and migration paths.
   - **Fix:** Wrap with cuserr error constructors.

4. **[HIGH] `internal/prompty.parser.inheritance.go:73,152,157,162,169,176`** — Uses `fmt.Errorf` for validation errors with position info.
   - **Fix:** Create dedicated inheritance error constructors using cuserr.

5. **[HIGH] `prompty.import_test.go:91,123`** — Unchecked `f.Write()` return values (errcheck linter).
   - **Fix:** `if _, err := f.Write(...); err != nil { t.Fatalf(...) }`.

---

## Quick Wins

1. **Fix `errors.Is()` in evaluator** — 2-line change in `internal/prompty.expr.evaluator.go:79,82`. (< 5 min)
2. **Fix unchecked `f.Write` in test** — 2 changes in `prompty.import_test.go`. (< 5 min)
3. **Add missing `MetaKeyLabel` constant** — `prompty.errors.go:427` uses hard-coded `"label"`. (< 5 min)
4. **Create `ADRs.md`** — Stub file documenting key architectural decisions already made (v2.1 agent model, cuserr adoption, plugin-first design). (< 30 min)

---

## Technical Debt

| Item | Effort | Impact | Priority |
|------|--------|--------|----------|
| Migrate `fmt.Errorf` in postgres storage to cuserr | S | HIGH | 1 |
| Migrate `fmt.Errorf` in inheritance parser to cuserr | S | HIGH | 2 |
| Create ADRs.md documentation | S | MEDIUM | 3 |
| Improve coverage for `resolveConfigEnvVars` (28.6%) | S | LOW | 4 |
| Improve coverage for `compareLess`/`compareGreater` (44.4%) | S | LOW | 5 |

---

## Risk Assessment

- **Deployment blockers**: None
- **Security concerns**: None — no secrets in code, no committed configs, test passwords only in E2E tests with testcontainers
- **Maintenance risks**: The `fmt.Errorf` inconsistency in postgres storage and inheritance parser creates a two-pattern error system that could confuse contributors. Should be unified to cuserr.

---

## Top 3 Recommendations

1. **Fix the CRITICAL `errors.Is()` violation** in `internal/prompty.expr.evaluator.go:79,82`. This is a correctness issue — wrapped context errors won't be detected properly.
2. **Unify error handling to cuserr** in `prompty.storage.postgres.go` and `internal/prompty.parser.inheritance.go`. Eliminate all remaining `fmt.Errorf` usage in production code to achieve a single error pattern.
3. **Create `ADRs.md`** documenting the key architectural decisions: plugin-first resolver design, `{~...~}` syntax choice, v2.1 agent model, cuserr adoption, provider serialization strategy.

---

*Assessment generated by Claude Code (/assess). Standards: `~/.claude/standards/golang.md` (v2026-02-07)*
