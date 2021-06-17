package audit

import (
	"strconv"
)

func newTLSSuite(r *Run) *Suite {
	suite := &Suite{
		Name:          "tls",
		Enable:        true,
		Configuration: r.suiteConfig("tls"),
		Core:          true,
		run:           r,
		log:           r,
		checks:        make(map[string]*Check),
	}

	newTLSRequiredCheck(suite)
	newTLSVerifyCheck(suite)
	newTLSTimeoutCheck(suite)

	return suite
}

func newTLSTimeoutCheck(s *Suite) *Check {
	check := &Check{
		Name:          "tls_timeout",
		Version:       version,
		Suite:         "tls",
		Enable:        true,
		Core:          true,
		Configuration: s.run.checkConfig("tls_timeout"),
	}

	threshold := float64(2)

	t, ok := check.Configuration["threshold"]
	if ok {
		threshold, _ = strconv.ParseFloat(t, 64)
	}

	check.serverCheck = func(s *Server, result *CheckResult, log Logger) error {
		result.Assert(func() {
			if !s.Variables.TLSRequired {
				result.Skipped = true
			}
		})

		result.Assert(func() {
			if s.Variables.TLSTimeout >= threshold {
				result.Outcome = OKOutcome
			} else {
				result.ErrorF("TLS Timeout %f is too low on %s", s.Variables.TLSTimeout, s)
				result.Outcome = ErrorOutcome
			}
		})

		return nil
	}

	s.registerCheck(check)

	return check
}

func newTLSRequiredCheck(s *Suite) *Check {
	check := &Check{
		Name:    "tls_required",
		Version: version,
		Suite:   "tls",
		Enable:  true,
		Core:    true,
		serverCheck: func(s *Server, result *CheckResult, log Logger) error {
			result.Assert(func() {
				if s.Variables.TLSRequired {
					result.Outcome = OKOutcome
				} else {
					result.ErrorF("TLS is not required on %s", s)
					result.Outcome = ErrorOutcome
				}
			})

			return nil
		},
	}

	s.registerCheck(check)

	return check
}

func newTLSVerifyCheck(s *Suite) *Check {
	check := &Check{
		Name:    "tls_verified",
		Version: version,
		Suite:   "tls",
		Enable:  true,
		Core:    true,
		serverCheck: func(s *Server, result *CheckResult, log Logger) error {
			result.Assert(func() {
				if !s.Variables.TLSRequired {
					result.Skipped = true
				}
			})

			result.Assert(func() {
				if s.Variables.TLSVerify {
					result.Outcome = OKOutcome
				} else {
					result.ErrorF("TLS is not verified on %s", s)
					result.Outcome = ErrorOutcome
				}
			})

			return nil
		},
	}

	s.registerCheck(check)

	return check
}
