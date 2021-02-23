package checks

func init() {
	RegisterFleetAuditor(&FleetVersionsAudit{})
}

type FleetVersionsAudit struct{}

func (a *FleetVersionsAudit) Name() string { return "Fleet versions" }

func (a *FleetVersionsAudit) Audit(servers []Server, results chan *Result) error {
	result := &Result{
		Check:       "server_versions",
		Outcome:     UnknownOutcome,
		Description: "Servers should be at the same version",
	}

	if IsDisabled(result.Check) {
		result.Outcome = SkippedOutcome
		results <- result
		return nil
	}

	version := ""
	ok := true
	for i, s := range servers {
		if i == 0 {
			version = s.Varz().Version
			continue
		}

		if s.Varz().Version != version {
			ok = false
		}
	}

	if ok {
		result.Outcome = OKOutcome
	} else {
		result.Outcome = CriticalOutcome
		result.Recommendation = "Set all servers to the same version"
	}

	results <- result
	return nil
}
