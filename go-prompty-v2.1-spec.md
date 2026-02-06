# go-prompty v2.1 Specification

> **Purpose**: Extend go-prompty to support the Agent Skills specification, composable agents, and multi-turn message templates.

## 1. Overview

### 1.1 Goals

1. **Agent Skills Compatibility**: Full support for [agentskills.io](https://agentskills.io) specification
2. **Agent Definition**: First-class support for defining AI agents with skills, tools, and constraints
3. **Composable Prompts**: Include, reference, and override system across document types
4. **Multi-Turn Templates**: Message-level templates for conversation structure
5. **Catalog Generation**: Auto-generate skill and tool catalogs for system prompts
6. **Namespaced Execution Config**: Provider-specific settings under `execution` namespace
7. **Backward Compatibility**: v1.x and v2.0 prompts should parse (with deprecation warnings)

### 1.2 Document Types

| Type | Purpose | Has Resources | Has Skills | Has Tools |
|------|---------|---------------|------------|-----------|
| `prompt` | Bare prompt template | No | No | No |
| `skill` | Packaged expertise (default) | Yes | No | No |
| `agent` | Autonomous entity | Yes | Yes | Yes |

### 1.3 Non-Goals (v2.1)

- Runtime execution (handled by skope/aigentchat)
- Multi-agent orchestration (handled by aigentflow)
- MCP server implementation
- Conversation state management

---

## 2. Composition Model

### 2.1 The Three Layers

```
┌─────────────────────────────────────────────────────────────┐
│                     RUNTIME REQUEST                          │
│  (input variables, execution overrides, skill activation)   │
└─────────────────────────────┬───────────────────────────────┘
                              │ overrides
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      ACTIVE SKILL                            │
│  (skill-specific instructions, execution config overrides)  │
└─────────────────────────────┬───────────────────────────────┘
                              │ extends/injects into
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                         AGENT                                │
│  (identity, base system prompt, tools, default execution)   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Override Precedence

For execution configuration:

```
HIGHEST  │  runtime request        (execution overrides in API call)
         │  active skill           (skill's execution config)
         │  agent definition       (agent's execution config)
LOWEST   │  system defaults        (go-prompty defaults)
```

**Merge Rule**: Shallow merge at each level. More specific values win.

Example:
```yaml
# Agent defaults
execution:
  provider: anthropic
  model: claude-sonnet-4-20250514
  temperature: 0.5
  max_tokens: 4096

# Skill override
execution:
  temperature: 0.2

# Runtime override
# Request: { "execution": { "max_tokens": 8192 } }

# Resolved:
execution:
  provider: anthropic           # from agent
  model: claude-sonnet-4-20250514  # from agent
  temperature: 0.2              # from skill
  max_tokens: 8192              # from runtime
```

### 2.3 Include Syntax

| Syntax | Description | Example |
|--------|-------------|---------|
| `{{include "self"}}` | Current document's body | System prompt self-reference |
| `{{include "prompt:slug"}}` | Include prompt by slug | `{{include "prompt:security-base"}}` |
| `{{include "skill:slug"}}` | Include skill body by slug | `{{include "skill:brave-red-falcon"}}` |
| `{{include "skill:slug@v2"}}` | Include specific version | `{{include "skill:brave-red-falcon@v2"}}` |
| `{{include "agent:current"}}` | Parent agent's compiled prompt | Used in skill message templates |

### 2.4 Catalog Syntax

| Syntax | Description |
|--------|-------------|
| `{{skills_catalog}}` | List of available skills (default format) |
| `{{skills_catalog format="detailed"}}` | Detailed skill descriptions |
| `{{skills_catalog format="compact"}}` | Minimal skill list |
| `{{tools_catalog}}` | List of available tools (default format) |
| `{{tools_catalog format="function_calling"}}` | JSON schema format for function calling |
| `{{tools_catalog format="compact"}}` | Minimal tool list |

### 2.5 Variable Syntax

| Syntax | Description |
|--------|-------------|
| `{{.context.key}}` | Static context variable |
| `{{.input.key}}` | Runtime input variable |
| `{{.examples.key}}` | Example data for few-shot |
| `{{.meta.name}}` | Document metadata |

---

## 3. Frontmatter Schema

### 3.1 Complete v2.1 Schema (Agent)

```yaml
---
# ============================================================
# DOCUMENT TYPE
# ============================================================
type: agent  # 'prompt' | 'skill' (default) | 'agent'

# ============================================================
# AGENT SKILLS STANDARD FIELDS (agentskills.io compatible)
# ============================================================

# REQUIRED: Unique identifier
# Constraints: max 64 chars, lowercase letters, numbers, hyphens
# Must not start or end with hyphen
name: security-auditor

# REQUIRED: What this agent does and when to use it
# Constraints: max 1024 chars, non-empty
description: |
  Autonomous security engineer that reviews code, identifies
  vulnerabilities, and provides remediation guidance.

# OPTIONAL: License identifier or filename
license: Apache-2.0

# OPTIONAL: Environment requirements
compatibility: "requires: semgrep, python3"

# OPTIONAL: Arbitrary key-value metadata
metadata:
  author: toni
  version: "1.0.0"
  tags: [security, code-review, agent]

# OPTIONAL (EXPERIMENTAL): Space-delimited allowed tools
allowed_tools: "Bash(semgrep:*) Read"

# ============================================================
# EXECUTION CONFIG (provider, model, inference settings)
# ============================================================

execution:
  # Provider identifier
  provider: anthropic  # openai | anthropic | google | vllm | custom

  # Model identifier (provider-specific)
  model: claude-sonnet-4-20250514

  # Sampling parameters
  temperature: 0.3
  max_tokens: 8192
  top_p: 0.95
  top_k: 40

  # Stop sequences
  stop_sequences:
    - "END_ANALYSIS"

  # Anthropic extended thinking
  thinking:
    enabled: true
    budget_tokens: 4096

  # Structured output
  response_format:
    type: json_schema
    strict: true
    name: SecurityAnalysisResult
    schema:
      type: object
      properties:
        summary: { type: string }
        findings: { type: array, items: { type: object } }
        risk_score: { type: integer }
      required: [summary, findings, risk_score]

  # vLLM guided decoding
  guided_decoding:
    type: json
    backend: outlines

  # Provider-specific passthrough
  provider_options:
    anthropic:
      metadata:
        user_id: "{{.input.user_id}}"

# ============================================================
# AGENT-SPECIFIC: SKILLS
# ============================================================

skills:
  # Reference by slug (latest version)
  - ref: brave-red-falcon

  # Reference with version pin
  - ref: swift-blue-horizon@v2

  # Reference with overrides
  - ref: calm-green-river
    execution:
      temperature: 0.1      # Override for this skill
    injection: none         # none | system_prompt | user_context
    marker: "{{active_skill}}"  # Injection point (if system_prompt)

  # Inline skill definition
  - inline:
      name: quick-scan
      description: "Fast security scan for critical issues only"
      body: |
        Perform a rapid scan focusing only on:
        - SQL injection
        - Authentication bypass
        - Remote code execution

# ============================================================
# AGENT-SPECIFIC: TOOLS
# ============================================================

tools:
  # MCP servers
  mcp:
    - name: filesystem
      uri: "mcp://github.com/modelcontextprotocol/servers/filesystem"
      config:
        allowed_paths: ["{{.context.workspace}}"]
      allowed_operations: [read, list]

    - name: github
      uri: "mcp://github.com/modelcontextprotocol/servers/github"
      config:
        repo: "{{.context.repo}}"

  # Function definitions (for function calling / tool use)
  functions:
    - name: run_static_analysis
      description: "Execute static analysis tool on codebase"
      parameters:
        type: object
        properties:
          tool:
            type: string
            enum: [semgrep, bandit, gosec, eslint-security]
            description: "Analysis tool to run"
          path:
            type: string
            description: "Path to analyze (file or directory)"
          config:
            type: string
            description: "Optional config file path"
        required: [tool, path]
      returns:
        type: object
        properties:
          findings: { type: array }
          exit_code: { type: integer }

    - name: search_vulnerability_db
      description: "Search CVE/NVD database for known vulnerabilities"
      parameters:
        type: object
        properties:
          query: { type: string, description: "Search query" }
          cpe: { type: string, description: "CPE identifier" }
          severity: { type: string, enum: [critical, high, medium, low] }
        required: [query]

# ============================================================
# AGENT-SPECIFIC: CONTEXT
# ============================================================

context:
  # Static context (always available)
  company: "vAudience.AI"
  standards: "OWASP Top 10, CWE/SANS Top 25"
  workspace: "/workspace"

  # Dynamic context (can reference input)
  current_user: "{{.input.user_name}}"
  target_repo: "{{.input.repo}}"

# ============================================================
# AGENT-SPECIFIC: CONSTRAINTS
# ============================================================

constraints:
  # Behavioral: Injected into system prompt
  behavioral:
    - "Never execute code without explicit user confirmation"
    - "Focus on security issues; ignore style unless security-relevant"
    - "Always provide severity ratings (critical/high/medium/low/info)"
    - "Include remediation guidance for every finding"

  # Operational: Enforced by runtime
  operational:
    max_iterations: 20
    max_tool_calls_per_turn: 10
    require_confirmation: [write, execute, delete]
    forbidden_tools: [rm, sudo, curl]

# ============================================================
# AGENT-SPECIFIC: MESSAGE TEMPLATES
# ============================================================

messages:
  # System message (typically includes body via {{include "self"}})
  - role: system
    content: "{{include \"self\"}}"

  # Optional few-shot examples
  - role: user
    content: "Review this code:\n```python\n{{.examples.vuln_code}}\n```"
  - role: assistant
    content: "{{.examples.vuln_analysis}}"

  # User turn template
  - role: user
    content: "{{.input.message}}"

# ============================================================
# SKOPE MANAGEMENT
# ============================================================

skope:
  slug: vigilant-steel-guardian
  forked_from: null
  created_at: "2025-02-01T10:30:00Z"
  created_by: usr_abc123
  updated_at: "2025-02-04T14:22:00Z"
  updated_by: usr_abc123
  version_number: 3
  visibility: private
  projects:
    - prj_security
    - prj_agents
  references:
    - brave-red-falcon
    - swift-blue-horizon

---

# Security Auditor

You are a senior security engineer at {{.context.company}}. Your mission is to
identify security vulnerabilities and provide actionable remediation guidance.

## Your Identity

- **Role**: Lead Security Engineer
- **Expertise**: Application security, secure code review, vulnerability assessment
- **Approach**: Thorough, methodical, pragmatic (focus on real risks)

## Security Standards

You evaluate all code against:
- {{.context.standards}}
- Language-specific security best practices
- The principle of least privilege

## Your Capabilities

### Skills

You have specialized expertise in these areas:

{{skills_catalog format="detailed"}}

When a task aligns with a skill, follow its procedures precisely.

### Tools

You can use these tools to assist your analysis:

{{tools_catalog format="function_calling"}}

Use tools when you need:
- Static analysis results
- Vulnerability database lookups
- File system access for code review

## Operating Procedures

1. **Understand the request**: Clarify scope if ambiguous
2. **Select approach**: Determine if a skill applies, or handle directly
3. **Analyze systematically**: Thoroughness prevents missed vulnerabilities
4. **Use tools appropriately**: Augment analysis with static analysis when helpful
5. **Report clearly**: Severity, location, description, remediation for each finding

## Constraints

{{#each constraints.behavioral}}
- {{.}}
{{/each}}

## Output Format

Unless otherwise specified, structure your findings as:

```markdown
## Summary
[Executive summary: X critical, Y high, Z medium findings]

## Findings

### [SEVERITY] Finding Title
- **Location**: file:line
- **Description**: What's wrong
- **Impact**: What could happen
- **Remediation**: How to fix
- **References**: CWE/CVE if applicable
```
```

### 3.2 Skill Schema (Reference)

Skills remain unchanged from v2.0, but can now be referenced by agents:

```yaml
---
type: skill  # Optional, default

name: security-code-review
description: |
  Reviews code for security vulnerabilities including injection attacks,
  authentication flaws, and data exposure risks.

license: Apache-2.0
metadata:
  author: toni
  tags: [security, code-review]

execution:
  provider: anthropic
  model: claude-sonnet-4-20250514
  temperature: 0.2

skope:
  slug: brave-red-falcon
  visibility: shared

---

# Security Code Review

## Step 1: Input Validation Analysis

Check all user inputs for:
- SQL injection vulnerabilities
- Cross-site scripting (XSS)
- Command injection
- Path traversal

## Step 2: Authentication Review

Verify:
- Password handling (hashing, storage)
- Session management
- Token validation
- MFA implementation

## Step 3: Data Exposure Check

Look for:
- Sensitive data in logs
- Unencrypted storage
- Excessive data in responses
- Missing access controls

...
```

### 3.3 Prompt Schema (Bare Template)

Prompts are simple templates without resources or agent capabilities:

```yaml
---
type: prompt

name: security-guidelines-base
description: Base security guidelines included in multiple agents/skills

metadata:
  author: security-team
  version: "2.0"

skope:
  slug: firm-gold-standard
  visibility: shared

---

## Security Guidelines

All security assessments must follow these principles:

1. **Defense in Depth**: Multiple layers of security controls
2. **Least Privilege**: Minimal permissions required
3. **Fail Secure**: Errors should not bypass security
4. **Input Validation**: Never trust user input
5. **Output Encoding**: Context-appropriate encoding
```

---

## 4. Type Definitions

### 4.1 Core Types

```go
// prompty/types.go

package prompty

import "time"

// DocumentType identifies the type of prompty document
type DocumentType string

const (
    DocumentTypePrompt DocumentType = "prompt"
    DocumentTypeSkill  DocumentType = "skill"  // Default
    DocumentTypeAgent  DocumentType = "agent"
)

// Prompt represents a go-prompty v2.1 document
type Prompt struct {
    // Document type (defaults to "skill")
    Type DocumentType `yaml:"type,omitempty"`

    // Agent Skills standard fields
    Name          string         `yaml:"name" validate:"required,max=64,slug"`
    Description   string         `yaml:"description" validate:"required,max=1024"`
    License       string         `yaml:"license,omitempty" validate:"max=100"`
    Compatibility string         `yaml:"compatibility,omitempty" validate:"max=500"`
    Metadata      map[string]any `yaml:"metadata,omitempty"`
    AllowedTools  string         `yaml:"allowed_tools,omitempty"`

    // Execution configuration
    Execution *ExecutionConfig `yaml:"execution,omitempty"`

    // Agent-specific fields
    Skills      []SkillRef        `yaml:"skills,omitempty"`
    Tools       *ToolsConfig      `yaml:"tools,omitempty"`
    Context     map[string]any    `yaml:"context,omitempty"`
    Constraints *ConstraintsConfig `yaml:"constraints,omitempty"`
    Messages    []MessageTemplate `yaml:"messages,omitempty"`

    // Skope management
    Skope *SkopeConfig `yaml:"skope,omitempty"`

    // Body content (markdown after frontmatter)
    Body string `yaml:"-"`
}

// IsAgent returns true if this is an agent document
func (p *Prompt) IsAgent() bool {
    return p.Type == DocumentTypeAgent
}

// IsSkill returns true if this is a skill document
func (p *Prompt) IsSkill() bool {
    return p.Type == DocumentTypeSkill || p.Type == ""
}

// IsPrompt returns true if this is a bare prompt document
func (p *Prompt) IsPrompt() bool {
    return p.Type == DocumentTypePrompt
}
```

### 4.2 Execution Config

```go
// prompty/execution.go

// ExecutionConfig contains LLM execution parameters
type ExecutionConfig struct {
    Provider        string             `yaml:"provider" validate:"omitempty,oneof=openai anthropic google vllm custom"`
    Model           string             `yaml:"model"`
    Temperature     *float64           `yaml:"temperature,omitempty" validate:"omitempty,gte=0,lte=2"`
    MaxTokens       *int               `yaml:"max_tokens,omitempty" validate:"omitempty,gte=1"`
    TopP            *float64           `yaml:"top_p,omitempty" validate:"omitempty,gte=0,lte=1"`
    TopK            *int               `yaml:"top_k,omitempty" validate:"omitempty,gte=1"`
    StopSequences   []string           `yaml:"stop_sequences,omitempty"`
    Thinking        *ThinkingConfig    `yaml:"thinking,omitempty"`
    ResponseFormat  *ResponseFormat    `yaml:"response_format,omitempty"`
    GuidedDecoding  *GuidedDecoding    `yaml:"guided_decoding,omitempty"`
    ProviderOptions map[string]any     `yaml:"provider_options,omitempty"`
}

// Merge combines two ExecutionConfigs, with other taking precedence
func (e *ExecutionConfig) Merge(other *ExecutionConfig) *ExecutionConfig {
    if other == nil {
        return e
    }
    if e == nil {
        return other
    }

    result := &ExecutionConfig{
        Provider:        coalesce(other.Provider, e.Provider),
        Model:           coalesce(other.Model, e.Model),
        Temperature:     coalescePtr(other.Temperature, e.Temperature),
        MaxTokens:       coalescePtr(other.MaxTokens, e.MaxTokens),
        TopP:            coalescePtr(other.TopP, e.TopP),
        TopK:            coalescePtr(other.TopK, e.TopK),
        StopSequences:   coalesceSlice(other.StopSequences, e.StopSequences),
        Thinking:        coalesceStruct(other.Thinking, e.Thinking),
        ResponseFormat:  coalesceStruct(other.ResponseFormat, e.ResponseFormat),
        GuidedDecoding:  coalesceStruct(other.GuidedDecoding, e.GuidedDecoding),
        ProviderOptions: mergeMaps(e.ProviderOptions, other.ProviderOptions),
    }

    return result
}

// ThinkingConfig for Anthropic extended thinking
type ThinkingConfig struct {
    Enabled      bool `yaml:"enabled"`
    BudgetTokens int  `yaml:"budget_tokens,omitempty" validate:"omitempty,gte=1"`
}

// ResponseFormat for structured output
type ResponseFormat struct {
    Type   string         `yaml:"type" validate:"required,oneof=json_object json_schema text"`
    Strict bool           `yaml:"strict,omitempty"`
    Name   string         `yaml:"name,omitempty"`
    Schema map[string]any `yaml:"schema,omitempty"`
}

// GuidedDecoding for vLLM
type GuidedDecoding struct {
    Type    string   `yaml:"type" validate:"required,oneof=json regex choice grammar"`
    Schema  any      `yaml:"schema,omitempty"`
    Pattern string   `yaml:"pattern,omitempty"`
    Choices []string `yaml:"choices,omitempty"`
    Grammar string   `yaml:"grammar,omitempty"`
    Backend string   `yaml:"backend,omitempty" validate:"omitempty,oneof=outlines lm-format-enforcer"`
}
```

### 4.3 Agent-Specific Types

```go
// prompty/agent.go

// SkillRef references a skill for use in an agent
type SkillRef struct {
    // Reference by slug (mutually exclusive with Inline)
    Ref string `yaml:"ref,omitempty"` // "slug" or "slug@v2"

    // Or inline definition
    Inline *InlineSkill `yaml:"inline,omitempty"`

    // Execution override when this skill is active
    Execution *ExecutionConfig `yaml:"execution,omitempty"`

    // How to inject skill when activated
    Injection SkillInjection `yaml:"injection,omitempty"`

    // Marker for injection (if injection == system_prompt)
    Marker string `yaml:"marker,omitempty"`
}

// Validate validates a SkillRef
func (s *SkillRef) Validate() error {
    if s.Ref == "" && s.Inline == nil {
        return ErrSkillRefEmpty
    }
    if s.Ref != "" && s.Inline != nil {
        return ErrSkillRefAmbiguous
    }
    return nil
}

// GetSlug returns the slug from a ref (without version)
func (s *SkillRef) GetSlug() string {
    if s.Ref == "" {
        return ""
    }
    parts := strings.Split(s.Ref, "@")
    return parts[0]
}

// GetVersion returns the pinned version, or 0 for latest
func (s *SkillRef) GetVersion() int {
    if s.Ref == "" {
        return 0
    }
    parts := strings.Split(s.Ref, "@v")
    if len(parts) == 2 {
        v, _ := strconv.Atoi(parts[1])
        return v
    }
    return 0
}

// SkillInjection defines how a skill is injected when activated
type SkillInjection string

const (
    SkillInjectionNone         SkillInjection = "none"          // Default: catalog only
    SkillInjectionSystemPrompt SkillInjection = "system_prompt" // Append to system
    SkillInjectionUserContext  SkillInjection = "user_context"  // Prepend to user
)

// InlineSkill defines a skill inline within an agent
type InlineSkill struct {
    Name        string `yaml:"name" validate:"required,max=64"`
    Description string `yaml:"description" validate:"required,max=1024"`
    Body        string `yaml:"body" validate:"required"`
}

// ToolsConfig defines tools available to an agent
type ToolsConfig struct {
    MCP       []MCPServer   `yaml:"mcp,omitempty"`
    Functions []FunctionDef `yaml:"functions,omitempty"`
}

// HasTools returns true if any tools are defined
func (t *ToolsConfig) HasTools() bool {
    if t == nil {
        return false
    }
    return len(t.MCP) > 0 || len(t.Functions) > 0
}

// MCPServer references an MCP server
type MCPServer struct {
    Name              string         `yaml:"name" validate:"required"`
    URI               string         `yaml:"uri" validate:"required"`
    Config            map[string]any `yaml:"config,omitempty"`
    AllowedOperations []string       `yaml:"allowed_operations,omitempty"`
}

// FunctionDef defines a callable function/tool
type FunctionDef struct {
    Name        string         `yaml:"name" validate:"required"`
    Description string         `yaml:"description" validate:"required"`
    Parameters  map[string]any `yaml:"parameters" validate:"required"` // JSON Schema
    Returns     map[string]any `yaml:"returns,omitempty"`              // JSON Schema
}

// ToOpenAITool converts to OpenAI function calling format
func (f *FunctionDef) ToOpenAITool() map[string]any {
    return map[string]any{
        "type": "function",
        "function": map[string]any{
            "name":        f.Name,
            "description": f.Description,
            "parameters":  f.Parameters,
        },
    }
}

// ToAnthropicTool converts to Anthropic tool format
func (f *FunctionDef) ToAnthropicTool() map[string]any {
    return map[string]any{
        "name":         f.Name,
        "description":  f.Description,
        "input_schema": f.Parameters,
    }
}

// ConstraintsConfig defines agent constraints
type ConstraintsConfig struct {
    Behavioral  []string                `yaml:"behavioral,omitempty"`
    Operational *OperationalConstraints `yaml:"operational,omitempty"`
}

// OperationalConstraints are enforced by runtime
type OperationalConstraints struct {
    MaxIterations       int      `yaml:"max_iterations,omitempty"`
    MaxToolCallsPerTurn int      `yaml:"max_tool_calls_per_turn,omitempty"`
    RequireConfirmation []string `yaml:"require_confirmation,omitempty"`
    ForbiddenTools      []string `yaml:"forbidden_tools,omitempty"`
}

// MessageTemplate defines a message in a conversation
type MessageTemplate struct {
    Role    string `yaml:"role" validate:"required,oneof=system user assistant"`
    Content string `yaml:"content" validate:"required"`
}
```

### 4.4 Skope Config

```go
// prompty/skope.go

// SkopeConfig contains skope-specific management fields
type SkopeConfig struct {
    Slug          string    `yaml:"slug,omitempty" validate:"omitempty,max=64,slugid"`
    ForkedFrom    string    `yaml:"forked_from,omitempty"`
    CreatedAt     time.Time `yaml:"created_at,omitempty"`
    CreatedBy     string    `yaml:"created_by,omitempty"`
    UpdatedAt     time.Time `yaml:"updated_at,omitempty"`
    UpdatedBy     string    `yaml:"updated_by,omitempty"`
    VersionNumber int       `yaml:"version_number,omitempty"`
    Visibility    string    `yaml:"visibility,omitempty" validate:"omitempty,oneof=private shared public"`
    Projects      []string  `yaml:"projects,omitempty"`
    References    []string  `yaml:"references,omitempty"`
}
```

---

## 5. Parsing and Serialization

### 5.1 Parser

```go
// prompty/parser.go

package prompty

import (
    "bytes"
    "errors"
    "fmt"
    "os"
    "strings"

    "gopkg.in/yaml.v3"
)

var (
    ErrNoFrontmatter      = errors.New("no frontmatter found")
    ErrInvalidFrontmatter = errors.New("invalid frontmatter format")
    ErrNotAnAgent         = errors.New("document is not an agent")
    ErrSkillNotFound      = errors.New("skill not found")
    ErrSkillRefEmpty      = errors.New("skill ref must have either ref or inline")
    ErrSkillRefAmbiguous  = errors.New("skill ref cannot have both ref and inline")
    ErrCircularReference  = errors.New("circular reference detected")
)

const frontmatterDelimiter = "---"

// Parse parses a go-prompty v2.1 document from bytes
func Parse(data []byte) (*Prompt, error) {
    content := string(data)

    // Find frontmatter boundaries
    trimmed := strings.TrimSpace(content)
    if !strings.HasPrefix(trimmed, frontmatterDelimiter) {
        return nil, ErrNoFrontmatter
    }

    // Split on delimiter
    parts := strings.SplitN(trimmed, frontmatterDelimiter, 3)
    if len(parts) < 3 {
        return nil, ErrInvalidFrontmatter
    }

    frontmatterStr := strings.TrimSpace(parts[1])
    body := strings.TrimSpace(parts[2])

    // Parse YAML frontmatter
    var prompt Prompt
    if err := yaml.Unmarshal([]byte(frontmatterStr), &prompt); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
    }

    // Set default type
    if prompt.Type == "" {
        prompt.Type = DocumentTypeSkill
    }

    prompt.Body = body

    // Validate
    if err := prompt.Validate(); err != nil {
        return nil, err
    }

    return &prompt, nil
}

// ParseFile parses from file path
func ParseFile(path string) (*Prompt, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    return Parse(data)
}

// MustParse panics on error (for testing)
func MustParse(data []byte) *Prompt {
    p, err := Parse(data)
    if err != nil {
        panic(err)
    }
    return p
}
```

### 5.2 Serializer

```go
// prompty/serialize.go

package prompty

import (
    "bytes"

    "gopkg.in/yaml.v3"
)

// SerializeOptions controls serialization behavior
type SerializeOptions struct {
    IncludeExecution bool // Include execution block
    IncludeSkope     bool // Include skope block
    IncludeAgentFields bool // Include skills, tools, etc.
    Indent           int  // YAML indentation (default 2)
}

// DefaultSerializeOptions returns standard options
func DefaultSerializeOptions() SerializeOptions {
    return SerializeOptions{
        IncludeExecution:   true,
        IncludeSkope:       true,
        IncludeAgentFields: true,
        Indent:             2,
    }
}

// AgentSkillsExportOptions returns options for Agent Skills export
func AgentSkillsExportOptions() SerializeOptions {
    return SerializeOptions{
        IncludeExecution:   false,
        IncludeSkope:       false,
        IncludeAgentFields: false, // Agent-specific fields not in spec
        Indent:             2,
    }
}

// Serialize converts a Prompt to bytes
func (p *Prompt) Serialize(opts SerializeOptions) ([]byte, error) {
    fm := make(map[string]any)

    // Type (omit if skill/default)
    if p.Type != "" && p.Type != DocumentTypeSkill {
        fm["type"] = string(p.Type)
    }

    // Agent Skills standard fields
    fm["name"] = p.Name
    fm["description"] = p.Description
    if p.License != "" {
        fm["license"] = p.License
    }
    if p.Compatibility != "" {
        fm["compatibility"] = p.Compatibility
    }
    if len(p.Metadata) > 0 {
        fm["metadata"] = p.Metadata
    }
    if p.AllowedTools != "" {
        fm["allowed_tools"] = p.AllowedTools
    }

    // Execution config
    if opts.IncludeExecution && p.Execution != nil {
        fm["execution"] = p.Execution
    }

    // Agent-specific fields
    if opts.IncludeAgentFields && p.IsAgent() {
        if len(p.Skills) > 0 {
            fm["skills"] = p.Skills
        }
        if p.Tools != nil && p.Tools.HasTools() {
            fm["tools"] = p.Tools
        }
        if len(p.Context) > 0 {
            fm["context"] = p.Context
        }
        if p.Constraints != nil {
            fm["constraints"] = p.Constraints
        }
        if len(p.Messages) > 0 {
            fm["messages"] = p.Messages
        }
    }

    // Skope config
    if opts.IncludeSkope && p.Skope != nil {
        fm["skope"] = p.Skope
    }

    // Serialize YAML
    var buf bytes.Buffer
    encoder := yaml.NewEncoder(&buf)
    encoder.SetIndent(opts.Indent)
    if err := encoder.Encode(fm); err != nil {
        return nil, err
    }

    // Compose final document
    var result bytes.Buffer
    result.WriteString("---\n")
    result.Write(buf.Bytes())
    result.WriteString("---\n\n")
    result.WriteString(p.Body)

    return result.Bytes(), nil
}
```

---

## 6. Compilation Pipeline

### 6.1 Resolver Interface

```go
// prompty/resolver.go

package prompty

// Resolver resolves references to prompts, skills, and agents
type Resolver interface {
    // ResolvePrompt returns a prompt by slug
    ResolvePrompt(slug string) (*Prompt, error)

    // ResolveSkill returns a skill by slug (with optional version)
    ResolveSkill(ref string) (*Prompt, error)

    // ResolveAgent returns an agent by slug
    ResolveAgent(slug string) (*Prompt, error)
}

// NoopResolver returns errors for all resolutions
type NoopResolver struct{}

func (r *NoopResolver) ResolvePrompt(slug string) (*Prompt, error) {
    return nil, fmt.Errorf("cannot resolve prompt: %s", slug)
}

func (r *NoopResolver) ResolveSkill(ref string) (*Prompt, error) {
    return nil, fmt.Errorf("cannot resolve skill: %s", ref)
}

func (r *NoopResolver) ResolveAgent(slug string) (*Prompt, error) {
    return nil, fmt.Errorf("cannot resolve agent: %s", slug)
}
```

### 6.2 Catalog Generation

```go
// prompty/catalog.go

package prompty

import (
    "fmt"
    "strings"
)

// CatalogFormat specifies the output format for catalogs
type CatalogFormat string

const (
    CatalogFormatDefault         CatalogFormat = ""
    CatalogFormatDetailed        CatalogFormat = "detailed"
    CatalogFormatCompact         CatalogFormat = "compact"
    CatalogFormatFunctionCalling CatalogFormat = "function_calling"
)

// GenerateSkillsCatalog creates a skills catalog from skill refs
func GenerateSkillsCatalog(skills []SkillRef, resolver Resolver, format CatalogFormat) string {
    if len(skills) == 0 {
        return "_No skills available._"
    }

    var sb strings.Builder

    for _, ref := range skills {
        var name, description string

        if ref.Inline != nil {
            name = ref.Inline.Name
            description = ref.Inline.Description
        } else if ref.Ref != "" {
            skill, err := resolver.ResolveSkill(ref.Ref)
            if err != nil {
                name = ref.GetSlug()
                description = fmt.Sprintf("_Error loading skill: %v_", err)
            } else {
                name = skill.Name
                description = skill.Description
            }
        }

        switch format {
        case CatalogFormatDetailed:
            sb.WriteString(fmt.Sprintf("### %s\n", name))
            if ref.Ref != "" {
                sb.WriteString(fmt.Sprintf("**Slug**: %s\n", ref.GetSlug()))
                if v := ref.GetVersion(); v > 0 {
                    sb.WriteString(fmt.Sprintf("**Version**: v%d (pinned)\n", v))
                } else {
                    sb.WriteString("**Version**: latest\n")
                }
            }
            sb.WriteString(fmt.Sprintf("**Description**: %s\n\n", description))

        case CatalogFormatCompact:
            sb.WriteString(fmt.Sprintf("- **%s**: %s\n", name, truncate(description, 80)))

        default: // Default format
            sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", name, description))
        }
    }

    return sb.String()
}

// GenerateToolsCatalog creates a tools catalog
func GenerateToolsCatalog(tools *ToolsConfig, format CatalogFormat) string {
    if tools == nil || !tools.HasTools() {
        return "_No tools available._"
    }

    var sb strings.Builder

    // MCP servers
    if len(tools.MCP) > 0 {
        if format != CatalogFormatCompact {
            sb.WriteString("#### MCP Servers\n\n")
        }
        for _, mcp := range tools.MCP {
            switch format {
            case CatalogFormatCompact:
                sb.WriteString(fmt.Sprintf("- **%s** (MCP)\n", mcp.Name))
            default:
                sb.WriteString(fmt.Sprintf("- **%s**: `%s`\n", mcp.Name, mcp.URI))
                if len(mcp.AllowedOperations) > 0 {
                    sb.WriteString(fmt.Sprintf("  - Allowed: %s\n", strings.Join(mcp.AllowedOperations, ", ")))
                }
            }
        }
        sb.WriteString("\n")
    }

    // Functions
    if len(tools.Functions) > 0 {
        if format != CatalogFormatCompact {
            sb.WriteString("#### Functions\n\n")
        }
        for _, fn := range tools.Functions {
            switch format {
            case CatalogFormatFunctionCalling:
                sb.WriteString(formatFunctionSchema(fn))

            case CatalogFormatCompact:
                sb.WriteString(fmt.Sprintf("- **%s**: %s\n", fn.Name, truncate(fn.Description, 60)))

            case CatalogFormatDetailed:
                sb.WriteString(fmt.Sprintf("### %s\n", fn.Name))
                sb.WriteString(fmt.Sprintf("%s\n\n", fn.Description))
                sb.WriteString("**Parameters**:\n")
                sb.WriteString(formatParameters(fn.Parameters))
                sb.WriteString("\n")

            default:
                sb.WriteString(fmt.Sprintf("### %s\n", fn.Name))
                sb.WriteString(fmt.Sprintf("%s\n", fn.Description))
                sb.WriteString(formatParametersCompact(fn.Parameters))
                sb.WriteString("\n")
            }
        }
    }

    return sb.String()
}

func formatFunctionSchema(fn FunctionDef) string {
    // Returns JSON schema format suitable for function calling
    return fmt.Sprintf("```json\n%s\n```\n\n", mustMarshalIndent(fn.ToOpenAITool()))
}

func formatParameters(params map[string]any) string {
    props, _ := params["properties"].(map[string]any)
    required, _ := params["required"].([]any)
    reqSet := make(map[string]bool)
    for _, r := range required {
        reqSet[r.(string)] = true
    }

    var sb strings.Builder
    for name, schema := range props {
        s := schema.(map[string]any)
        typ, _ := s["type"].(string)
        desc, _ := s["description"].(string)
        req := ""
        if reqSet[name] {
            req = " (required)"
        }
        sb.WriteString(fmt.Sprintf("- `%s` (%s%s): %s\n", name, typ, req, desc))
    }
    return sb.String()
}
```

### 6.3 Template Compilation

```go
// prompty/compile.go

package prompty

import (
    "bytes"
    "fmt"
    "regexp"
    "strings"
    "text/template"
)

// Patterns for template directives
var (
    includePattern = regexp.MustCompile(`\{\{\s*include\s+"([^"]+)"\s*\}\}`)
    catalogPattern = regexp.MustCompile(`\{\{(skills_catalog|tools_catalog)(?:\s+format="([^"]+)")?\}\}`)
)

// CompileOptions controls compilation behavior
type CompileOptions struct {
    Resolver      Resolver
    SkillsCatalog string // Pre-generated skills catalog
    ToolsCatalog  string // Pre-generated tools catalog
}

// CompiledPrompt is a fully resolved prompt ready for execution
type CompiledPrompt struct {
    Messages    []CompiledMessage
    Execution   *ExecutionConfig
    Tools       *ToolsConfig
    Constraints *OperationalConstraints
}

// CompiledMessage is a resolved message
type CompiledMessage struct {
    Role    string
    Content string
}

// Compile compiles a prompt/skill with input variables
func (p *Prompt) Compile(input map[string]any, opts CompileOptions) (string, error) {
    return compileTemplate(p.Body, buildTemplateData(p, input, opts), opts.Resolver)
}

// CompileAgent compiles an agent document to executable form
func (p *Prompt) CompileAgent(input map[string]any, opts CompileOptions) (*CompiledPrompt, error) {
    if !p.IsAgent() {
        return nil, ErrNotAnAgent
    }

    // Build template data
    data := buildTemplateData(p, input, opts)

    // Generate catalogs if not provided
    if opts.SkillsCatalog == "" {
        opts.SkillsCatalog = GenerateSkillsCatalog(p.Skills, opts.Resolver, CatalogFormatDefault)
    }
    if opts.ToolsCatalog == "" {
        opts.ToolsCatalog = GenerateToolsCatalog(p.Tools, CatalogFormatDefault)
    }

    // Compile body (system prompt)
    systemPrompt, err := compileTemplate(p.Body, data, opts.Resolver)
    if err != nil {
        return nil, fmt.Errorf("compiling body: %w", err)
    }

    // Build messages
    messages := p.Messages
    if len(messages) == 0 {
        // Default message structure
        messages = []MessageTemplate{
            {Role: "system", Content: `{{include "self"}}`},
            {Role: "user", Content: "{{.input.message}}"},
        }
    }

    compiledMessages := make([]CompiledMessage, 0, len(messages))
    for _, m := range messages {
        content := m.Content

        // Handle special includes
        if strings.Contains(content, `{{include "self"}}`) {
            content = strings.ReplaceAll(content, `{{include "self"}}`, systemPrompt)
        }

        // Compile remaining template
        compiled, err := compileTemplate(content, data, opts.Resolver)
        if err != nil {
            return nil, fmt.Errorf("compiling message: %w", err)
        }

        compiledMessages = append(compiledMessages, CompiledMessage{
            Role:    m.Role,
            Content: compiled,
        })
    }

    // Build result
    result := &CompiledPrompt{
        Messages:  compiledMessages,
        Execution: p.Execution,
        Tools:     p.Tools,
    }

    if p.Constraints != nil {
        result.Constraints = p.Constraints.Operational
    }

    return result, nil
}

// ActivateSkill compiles an agent with a specific skill activated
func (p *Prompt) ActivateSkill(skillRef string, input map[string]any, opts CompileOptions) (*CompiledPrompt, error) {
    if !p.IsAgent() {
        return nil, ErrNotAnAgent
    }

    // Find skill reference
    var found *SkillRef
    for i := range p.Skills {
        if p.Skills[i].GetSlug() == skillRef || p.Skills[i].Ref == skillRef {
            found = &p.Skills[i]
            break
        }
    }
    if found == nil {
        return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, skillRef)
    }

    // Compile base agent
    compiled, err := p.CompileAgent(input, opts)
    if err != nil {
        return nil, err
    }

    // Load skill content
    var skillBody string
    if found.Inline != nil {
        skillBody = found.Inline.Body
    } else {
        skill, err := opts.Resolver.ResolveSkill(found.Ref)
        if err != nil {
            return nil, err
        }
        skillBody, err = compileTemplate(skill.Body, buildTemplateData(p, input, opts), opts.Resolver)
        if err != nil {
            return nil, err
        }
    }

    // Apply injection
    switch found.Injection {
    case SkillInjectionSystemPrompt:
        for i, m := range compiled.Messages {
            if m.Role == "system" {
                if found.Marker != "" && strings.Contains(m.Content, found.Marker) {
                    compiled.Messages[i].Content = strings.Replace(m.Content, found.Marker, skillBody, 1)
                } else {
                    compiled.Messages[i].Content += "\n\n## Active Skill\n\n" + skillBody
                }
                break
            }
        }

    case SkillInjectionUserContext:
        for i := len(compiled.Messages) - 1; i >= 0; i-- {
            if compiled.Messages[i].Role == "user" {
                compiled.Messages[i].Content = skillBody + "\n\n" + compiled.Messages[i].Content
                break
            }
        }
    }

    // Apply skill execution overrides
    if found.Execution != nil {
        compiled.Execution = compiled.Execution.Merge(found.Execution)
    }

    return compiled, nil
}

func buildTemplateData(p *Prompt, input map[string]any, opts CompileOptions) map[string]any {
    data := map[string]any{
        "input": input,
        "meta": map[string]any{
            "name":        p.Name,
            "description": p.Description,
            "type":        string(p.Type),
        },
    }

    // Add context
    if p.Context != nil {
        ctx := make(map[string]any)
        for k, v := range p.Context {
            // Context values can themselves be templates
            if s, ok := v.(string); ok && strings.Contains(s, "{{") {
                compiled, _ := compileSimpleTemplate(s, data)
                ctx[k] = compiled
            } else {
                ctx[k] = v
            }
        }
        data["context"] = ctx
    }

    // Add constraints
    if p.Constraints != nil {
        data["constraints"] = p.Constraints
    }

    // Add catalogs
    data["skills_catalog"] = opts.SkillsCatalog
    data["tools_catalog"] = opts.ToolsCatalog

    return data
}

func compileTemplate(tmplStr string, data map[string]any, resolver Resolver) (string, error) {
    // First pass: resolve includes
    result := includePattern.ReplaceAllStringFunc(tmplStr, func(match string) string {
        parts := includePattern.FindStringSubmatch(match)
        if len(parts) < 2 {
            return match
        }
        ref := parts[1]

        // Handle different include types
        switch {
        case ref == "self":
            return match // Keep for later handling

        case strings.HasPrefix(ref, "prompt:"):
            slug := strings.TrimPrefix(ref, "prompt:")
            if resolver != nil {
                if p, err := resolver.ResolvePrompt(slug); err == nil {
                    return p.Body
                }
            }
            return fmt.Sprintf("[ERROR: prompt not found: %s]", slug)

        case strings.HasPrefix(ref, "skill:"):
            skillRef := strings.TrimPrefix(ref, "skill:")
            if resolver != nil {
                if s, err := resolver.ResolveSkill(skillRef); err == nil {
                    return s.Body
                }
            }
            return fmt.Sprintf("[ERROR: skill not found: %s]", skillRef)

        case strings.HasPrefix(ref, "agent:"):
            // Special case for parent agent reference
            return match

        default:
            // Try as skill slug
            if resolver != nil {
                if s, err := resolver.ResolveSkill(ref); err == nil {
                    return s.Body
                }
            }
            return fmt.Sprintf("[ERROR: reference not found: %s]", ref)
        }
    })

    // Second pass: resolve catalogs
    result = catalogPattern.ReplaceAllStringFunc(result, func(match string) string {
        parts := catalogPattern.FindStringSubmatch(match)
        if len(parts) < 2 {
            return match
        }
        catalogType := parts[1]
        format := CatalogFormat("")
        if len(parts) > 2 {
            format = CatalogFormat(parts[2])
        }

        switch catalogType {
        case "skills_catalog":
            if s, ok := data["skills_catalog"].(string); ok {
                return s
            }
        case "tools_catalog":
            if s, ok := data["tools_catalog"].(string); ok {
                return s
            }
        }
        return match
    })

    // Third pass: Go template execution
    return compileSimpleTemplate(result, data)
}

func compileSimpleTemplate(tmplStr string, data map[string]any) (string, error) {
    tmpl, err := template.New("prompt").Funcs(DefaultFuncs()).Parse(tmplStr)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", err
    }

    return buf.String(), nil
}
```

### 6.4 Template Functions

```go
// prompty/funcs.go

package prompty

import (
    "encoding/json"
    "strings"
    "text/template"
)

// DefaultFuncs returns standard template functions
func DefaultFuncs() template.FuncMap {
    return template.FuncMap{
        // String functions
        "upper":      strings.ToUpper,
        "lower":      strings.ToLower,
        "title":      strings.Title,
        "trim":       strings.TrimSpace,
        "replace":    strings.ReplaceAll,
        "contains":   strings.Contains,
        "hasPrefix":  strings.HasPrefix,
        "hasSuffix":  strings.HasSuffix,
        "split":      strings.Split,
        "join":       strings.Join,
        "truncate":   truncate,

        // JSON functions
        "toJSON": func(v any) string {
            b, _ := json.Marshal(v)
            return string(b)
        },
        "toPrettyJSON": func(v any) string {
            b, _ := json.MarshalIndent(v, "", "  ")
            return string(b)
        },
        "fromJSON": func(s string) any {
            var v any
            json.Unmarshal([]byte(s), &v)
            return v
        },

        // Default/coalesce
        "default": func(def, val any) any {
            if val == nil || val == "" {
                return def
            }
            return val
        },
        "coalesce": func(vals ...any) any {
            for _, v := range vals {
                if v != nil && v != "" {
                    return v
                }
            }
            return nil
        },

        // List helpers
        "first": func(list []any) any {
            if len(list) > 0 {
                return list[0]
            }
            return nil
        },
        "last": func(list []any) any {
            if len(list) > 0 {
                return list[len(list)-1]
            }
            return nil
        },
        "list": func(items ...any) []any {
            return items
        },

        // Conditional
        "ternary": func(cond bool, t, f any) any {
            if cond {
                return t
            }
            return f
        },
    }
}

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-3] + "..."
}
```

---

## 7. Import/Export

### 7.1 Agent Skills Export

```go
// prompty/export.go

// ExportAgentSkill exports as Agent Skills compatible format
// Note: Agent-specific fields (skills, tools, etc.) are stripped
func (p *Prompt) ExportAgentSkill() ([]byte, error) {
    return p.Serialize(AgentSkillsExportOptions())
}

// ExportFull exports with all fields preserved
func (p *Prompt) ExportFull() ([]byte, error) {
    return p.Serialize(DefaultSerializeOptions())
}

// ExportSkillDirectory exports as zip with resources
func ExportSkillDirectory(prompt *Prompt, resources map[string][]byte) ([]byte, error) {
    var buf bytes.Buffer
    zw := zip.NewWriter(&buf)

    // Determine filename based on type
    filename := "SKILL.md"
    if prompt.IsAgent() {
        filename = "AGENT.md"
    }

    // Write main file
    content, err := prompt.ExportFull()
    if err != nil {
        return nil, err
    }

    w, err := zw.Create(filename)
    if err != nil {
        return nil, err
    }
    if _, err := w.Write(content); err != nil {
        return nil, err
    }

    // Write resources
    for path, data := range resources {
        w, err := zw.Create(path)
        if err != nil {
            return nil, err
        }
        if _, err := w.Write(data); err != nil {
            return nil, err
        }
    }

    if err := zw.Close(); err != nil {
        return nil, err
    }

    return buf.Bytes(), nil
}
```

### 7.2 Import

```go
// prompty/import.go

// ImportResult contains parsed document and resources
type ImportResult struct {
    Prompt    *Prompt
    Resources map[string][]byte
}

// Import handles .md or .zip import
func Import(data []byte, filename string) (*ImportResult, error) {
    if strings.HasSuffix(strings.ToLower(filename), ".zip") {
        return ImportDirectory(data)
    }
    
    prompt, err := Parse(data)
    if err != nil {
        return nil, err
    }
    
    return &ImportResult{
        Prompt:    prompt,
        Resources: make(map[string][]byte),
    }, nil
}

// ImportDirectory imports a zipped directory
func ImportDirectory(data []byte) (*ImportResult, error) {
    r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
    if err != nil {
        return nil, err
    }

    result := &ImportResult{
        Resources: make(map[string][]byte),
    }

    for _, f := range r.File {
        rc, err := f.Open()
        if err != nil {
            return nil, err
        }

        content, err := io.ReadAll(rc)
        rc.Close()
        if err != nil {
            return nil, err
        }

        name := strings.ToUpper(f.Name)
        if name == "SKILL.MD" || name == "AGENT.MD" {
            prompt, err := Parse(content)
            if err != nil {
                return nil, err
            }
            result.Prompt = prompt
        } else {
            result.Resources[f.Name] = content
        }
    }

    if result.Prompt == nil {
        return nil, ErrNoFrontmatter
    }

    return result, nil
}
```

---

## 8. Validation

```go
// prompty/validate.go

package prompty

import (
    "regexp"

    "github.com/go-playground/validator/v10"
)

var (
    slugPattern   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
    slugIDPattern = regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z]+$`)
)

// Validate performs full validation
func (p *Prompt) Validate() error {
    v := validator.New()

    // Register custom validators
    v.RegisterValidation("slug", validateSlug)
    v.RegisterValidation("slugid", validateSlugID)

    // Validate struct
    if err := v.Struct(p); err != nil {
        return err
    }

    // Type-specific validation
    if p.IsAgent() {
        return p.validateAgent()
    }

    return nil
}

func (p *Prompt) validateAgent() error {
    // Validate skill refs
    for _, s := range p.Skills {
        if err := s.Validate(); err != nil {
            return err
        }
    }

    // Validate messages
    hasSystem := false
    hasUser := false
    for _, m := range p.Messages {
        if m.Role == "system" {
            hasSystem = true
        }
        if m.Role == "user" {
            hasUser = true
        }
    }

    // If messages defined, should have at least system or user
    if len(p.Messages) > 0 && !hasSystem && !hasUser {
        return fmt.Errorf("messages must include at least system or user role")
    }

    return nil
}

func validateSlug(fl validator.FieldLevel) bool {
    return slugPattern.MatchString(fl.Field().String())
}

func validateSlugID(fl validator.FieldLevel) bool {
    return slugIDPattern.MatchString(fl.Field().String())
}

// ValidateForExecution checks if ready for LLM execution
func (p *Prompt) ValidateForExecution() error {
    if p.Execution == nil {
        return ErrNoExecutionConfig
    }
    if p.Execution.Provider == "" {
        return ErrNoProvider
    }
    if p.Execution.Model == "" {
        return ErrNoModel
    }
    return p.Validate()
}
```

---

## 9. Constants

```go
// prompty/constants.go

package prompty

// Document types
const (
    TypePrompt = "prompt"
    TypeSkill  = "skill"
    TypeAgent  = "agent"
)

// Provider identifiers
const (
    ProviderOpenAI    = "openai"
    ProviderAnthropic = "anthropic"
    ProviderGoogle    = "google"
    ProviderVLLM      = "vllm"
    ProviderCustom    = "custom"
)

// Response format types
const (
    ResponseFormatText       = "text"
    ResponseFormatJSONObject = "json_object"
    ResponseFormatJSONSchema = "json_schema"
)

// Skill injection modes
const (
    InjectionNone         = "none"
    InjectionSystemPrompt = "system_prompt"
    InjectionUserContext  = "user_context"
)

// Catalog formats
const (
    FormatDefault         = ""
    FormatDetailed        = "detailed"
    FormatCompact         = "compact"
    FormatFunctionCalling = "function_calling"
)

// Visibility levels
const (
    VisibilityPrivate = "private"
    VisibilityShared  = "shared"
    VisibilityPublic  = "public"
)

// Field constraints
const (
    MaxNameLength          = 64
    MaxDescriptionLength   = 1024
    MaxLicenseLength       = 100
    MaxCompatibilityLength = 500
    MaxBodyTokens          = 5000 // Recommended, not enforced
)

// Error messages
const (
    ErrMsgNoFrontmatter       = "document has no frontmatter"
    ErrMsgInvalidFrontmatter  = "frontmatter is invalid"
    ErrMsgNameRequired        = "name is required"
    ErrMsgDescriptionRequired = "description is required"
    ErrMsgNotAnAgent          = "document is not an agent"
    ErrMsgSkillNotFound       = "skill not found"
    ErrMsgNoExecutionConfig   = "execution configuration required"
    ErrMsgNoProvider          = "provider is required"
    ErrMsgNoModel             = "model is required"
)
```

---

## 10. Summary: v2.0 to v2.1 Changes

| Feature | v2.0 | v2.1 |
|---------|------|------|
| Document types | skill only | prompt, skill, agent |
| Skill references | `{{ref "slug"}}` | `{{include "skill:slug"}}`, `{{include "prompt:slug"}}` |
| Agent definition | N/A | Full support |
| Tools | N/A | MCP servers, function definitions |
| Catalogs | N/A | `{{skills_catalog}}`, `{{tools_catalog}}` |
| Context | N/A | Static + dynamic injection |
| Constraints | N/A | Behavioral + operational |
| Messages | N/A | Multi-turn templates |
| Composition | Include | Include + override + merge |
| Execution merge | N/A | 3-layer precedence |

---

## 11. Migration Guide

### From v2.0 to v2.1

v2.1 is backward compatible with v2.0. Existing skills work unchanged.

To convert a skill to an agent:

1. Add `type: agent` to frontmatter
2. Add `skills:` section with skill references
3. Add `tools:` section if needed
4. Add `context:` for static variables
5. Optionally add `constraints:` and `messages:`
6. Update body to use `{{skills_catalog}}` and `{{tools_catalog}}`

---

**Document End**

*Excellence. Always.*
