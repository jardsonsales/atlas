package entities

import "strings"

// StepStatus is the outcome of a single provisioning step applied to the host.
type StepStatus string

const (
	// StepApplied means the step ran successfully on the host.
	StepApplied StepStatus = "APPLIED"
	// StepSkipped means the step was intentionally not executed (e.g. the sandbox mode or a
	// disabled backend unit).
	StepSkipped StepStatus = "SKIPPED"
	// StepFailed means the step did not succeed; the run is considered unsuccessful.
	StepFailed StepStatus = "FAILED"
)

// ProvisioningStep is one action of a provisioning run (writing the vhost, reloading the web
// server, issuing the certificate, enabling the backend unit...), with its outcome and detail.
type ProvisioningStep struct {
	label  string
	status StepStatus
	detail string
}

// NewProvisioningStep creates a ProvisioningStep value object.
func NewProvisioningStep(label string, status StepStatus, detail string) ProvisioningStep {
	return ProvisioningStep{
		label:  strings.TrimSpace(label),
		status: status,
		detail: strings.TrimSpace(detail),
	}
}

func (s ProvisioningStep) Label() string      { return s.label }
func (s ProvisioningStep) Status() StepStatus { return s.status }
func (s ProvisioningStep) Detail() string     { return s.detail }

// ProvisioningResult is the ordered outcome of applying a domain provisioning plan to the host.
type ProvisioningResult struct {
	domain string
	steps  []ProvisioningStep
}

// NewProvisioningResult creates a ProvisioningResult for a domain.
func NewProvisioningResult(domain string, steps []ProvisioningStep) *ProvisioningResult {
	return &ProvisioningResult{
		domain: strings.TrimSpace(domain),
		steps:  steps,
	}
}

func (r *ProvisioningResult) Domain() string            { return r.domain }
func (r *ProvisioningResult) Steps() []ProvisioningStep { return r.steps }

// Succeeded reports whether every step was applied or intentionally skipped (no failures).
func (r *ProvisioningResult) Succeeded() bool {
	for _, step := range r.steps {
		if step.status == StepFailed {
			return false
		}
	}
	return true
}
