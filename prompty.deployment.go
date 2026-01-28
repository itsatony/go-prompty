package prompty

import (
	"regexp"
	"slices"
)

// validLabelNameRegex is the compiled regex for label name validation.
var validLabelNameRegex = regexp.MustCompile(LabelNamePattern)

// ValidStatusTransitions defines the allowed state machine transitions.
// Key is the current status, value is the list of allowed target statuses.
var ValidStatusTransitions = map[DeploymentStatus][]DeploymentStatus{
	DeploymentStatusDraft:      {DeploymentStatusActive, DeploymentStatusArchived},
	DeploymentStatusActive:     {DeploymentStatusDeprecated, DeploymentStatusArchived},
	DeploymentStatusDeprecated: {DeploymentStatusActive, DeploymentStatusArchived},
	DeploymentStatusArchived:   {}, // Terminal state - no transitions allowed
}

// ValidateLabel validates a label name according to naming rules.
// Label names must:
// - Not be empty
// - Not exceed LabelMaxLength characters
// - Match LabelNamePattern (start with lowercase letter, then lowercase letters, digits, underscores, or hyphens)
func ValidateLabel(label string) error {
	if label == "" {
		return NewInvalidLabelNameError(label, ErrMsgLabelNameEmpty)
	}

	if len(label) > LabelMaxLength {
		return NewInvalidLabelNameError(label, ErrMsgLabelNameTooLong)
	}

	if !validLabelNameRegex.MatchString(label) {
		return NewInvalidLabelNameError(label, ErrMsgInvalidLabelFormat)
	}

	return nil
}

// CanTransitionStatus checks if a status transition is valid according to the state machine.
func CanTransitionStatus(from, to DeploymentStatus) bool {
	allowed, ok := ValidStatusTransitions[from]
	if !ok {
		return false
	}
	return slices.Contains(allowed, to)
}

// IsValid checks if the deployment status value is valid.
func (s DeploymentStatus) IsValid() bool {
	switch s {
	case DeploymentStatusDraft, DeploymentStatusActive,
		DeploymentStatusDeprecated, DeploymentStatusArchived:
		return true
	default:
		return false
	}
}

// String returns the string representation of the deployment status.
func (s DeploymentStatus) String() string {
	return string(s)
}

// AllDeploymentStatuses returns all valid deployment status values.
func AllDeploymentStatuses() []DeploymentStatus {
	return []DeploymentStatus{
		DeploymentStatusDraft,
		DeploymentStatusActive,
		DeploymentStatusDeprecated,
		DeploymentStatusArchived,
	}
}

// IsTerminal returns true if the status is a terminal state (no further transitions allowed).
func (s DeploymentStatus) IsTerminal() bool {
	return s == DeploymentStatusArchived
}

// IsUsable returns true if the status indicates the template can be used for execution.
// Draft and archived templates are not typically used in production.
func (s DeploymentStatus) IsUsable() bool {
	return s == DeploymentStatusActive || s == DeploymentStatusDeprecated
}

// ParseDeploymentStatus parses a string into a DeploymentStatus.
// Returns error if the string is not a valid status.
func ParseDeploymentStatus(s string) (DeploymentStatus, error) {
	status := DeploymentStatus(s)
	if !status.IsValid() {
		return "", NewInvalidDeploymentStatusError(s)
	}
	return status, nil
}
