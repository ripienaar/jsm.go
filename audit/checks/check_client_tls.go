package checks

import (
	"fmt"
)

func init() {
	RegisterServerAuditor(&TLSServerAudit{})
}

type TLSServerAudit struct{}

func (a *TLSServerAudit) Name() string { return "Client TLS Required" }

func (a *TLSServerAudit) Audit(s Server, results chan *Result) error {
	if s == nil {
		return fmt.Errorf("nil server specified")
	}

	err := a.tlsRequired(s, results)
	if err != nil {
		return err
	}

	err = a.tlsVerified(s, results)
	if err != nil {
		return err
	}

	return nil
}

func (a *TLSServerAudit) tlsVerified(s Server, results chan *Result) error {
	result := &Result{
		Check:       "client_tls_verified",
		Outcome:     UnknownOutcome,
		Description: "Clients should be verified using a Certificate Authority",
	}

	if IsDisabled(result.Check) {
		result.Outcome = SkippedOutcome
		results <- result
		return nil
	}

	if s.Varz().TLSRequired {
		result.Outcome = OKOutcome
	} else {
		result.Recommendation = "Enabled TLS verification for client connections"
		result.Outcome = CriticalOutcome
	}

	results <- result

	return nil
}

func (a *TLSServerAudit) tlsRequired(s Server, results chan *Result) error {
	result := &Result{
		Check:       "client_tls_required",
		Outcome:     UnknownOutcome,
		Description: "Clients should require TLS",
	}

	if IsDisabled(result.Check) {
		result.Outcome = SkippedOutcome
		results <- result
		return nil
	}

	if s.Varz().TLSRequired {
		result.Outcome = OKOutcome
	} else {
		result.Recommendation = "Enabled TLS for client connections"
		result.Outcome = CriticalOutcome
	}

	results <- result

	return nil
}
