package audit

import (
	"fmt"
)

type Suite struct {
	Name          string            `json:"name"`
	Enable        bool              `json:"enabled"`
	Configuration map[string]string `json:"configuration"`
	Core          bool              `json:"core"`

	checkRan     int
	checkPassed  int
	checkFailed  int
	checkSkipped int
	checkUnknown int

	results []*CheckResult
	run     *Run
	log     Logger
	checks  map[string]*Check
}

func (s *Suite) assertions() int {
	a := 0
	for _, res := range s.results {
		a += res.AssertionCount
	}

	return a
}

func (s *Suite) checksRan() int {
	return s.checkRan
}

func (s *Suite) checksFailed() int {
	return s.checkFailed
}

func (s *Suite) checksPassed() int {
	return s.checkPassed
}

func (s *Suite) outcome() CheckOutcome {
	if len(s.results) == 0 {
		return UnknownOutcome
	}

	for _, r := range s.results {
		if r.Outcome == ErrorOutcome {
			return ErrorOutcome
		}
	}

	return OKOutcome
}

func (s *Suite) registerCheck(check *Check) error {
	_, ok := s.checks[check.Name]
	if ok {
		return fmt.Errorf("already registered")
	}

	s.checks[check.Name] = check

	return nil
}

func (s *Suite) recordOutcomes(result *CheckResult) {
	if result.Skipped {
		s.checkSkipped++
	}

	switch result.Outcome {
	case UnknownOutcome:
		s.checkUnknown++
	case OKOutcome:
		s.checkPassed++
	case ErrorOutcome:
		s.checkFailed++
	}
}

func (s *Suite) runChecks() CheckOutcome {
	s.log.Infof("Starting suite %s with %d checks", s.Name, len(s.checks))

	for _, check := range s.checks {
		if s.run.isCheckSkipped(check.Name) {
			result := check.newResult(s.log)
			result.Skipped = true
			s.checkSkipped++
			result.finalize()
			s.results = append(s.results, result)

			continue
		}

		s.log.Infof("Starting check %s", check.Name)
		s.checkRan++

		if check.Clustered {
			result := check.newResult(s.log)
			err := check.clusterCheck(s.run.Servers, result, s.log)
			if err != nil {
				result.Outcome = ErrorOutcome
				result.Error = err.Error()
			}

			// record times etc
			result.finalize()

			s.recordOutcomes(result)

			s.results = append(s.results, result)
		} else {
			for _, srv := range s.run.Servers {
				result := check.newResult(s.log)
				result.ServerID = srv.Info.ID
				result.ServerName = srv.Info.Name
				result.ServerCluster = srv.Info.Cluster

				err := check.serverCheck(srv, result, s.log)
				if err != nil {
					result.Outcome = ErrorOutcome
					result.Error = err.Error()
				}

				// record times etc
				result.finalize()

				s.recordOutcomes(result)

				s.results = append(s.results, result)
			}
		}
	}

	msg := fmt.Sprintf("Finished suite %s with %d checks and %d assertions: skipped: %d failed: %d ok: %d", s.Name, len(s.checks), s.assertions(), s.checkSkipped, s.checkFailed, s.checkPassed)
	if s.checkFailed > 0 {
		s.log.Errorf(msg)
	} else {
		s.log.Infof(msg)
	}

	return s.outcome()
}
