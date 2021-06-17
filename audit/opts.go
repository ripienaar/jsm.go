package audit

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// RunOpt configures a run
type RunOpt func(*Run) error

// WithNetworkTimeout sets the timeout for access the network, defaults to 5 seconds, minimum 2 seconds
func WithNetworkTimeout(t time.Duration) RunOpt {
	return func(r *Run) error {
		if t < 2*time.Second {
			return fmt.Errorf("minimum timeout is 2 seconds")
		}

		r.mu.Lock()
		defer r.mu.Unlock()

		r.Timeout = t
		return nil
	}
}

// WithNatsConnection sets the connection to use for auditing
func WithNatsConnection(nc *nats.Conn) RunOpt {
	return func(r *Run) error {
		if nc == nil {
			return fmt.Errorf("connection is nil")
		}

		r.mu.Lock()
		defer r.mu.Unlock()

		r.nc = nc
		return nil
	}
}

// WithCheckEnabled enables a check
func WithCheckEnabled(checks ...string) RunOpt {
	return func(r *Run) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.ChecksEnabled = append(r.ChecksEnabled, checks...)

		return nil
	}
}

// WithCheckDisabled skips a check
func WithCheckDisabled(checks ...string) RunOpt {
	return func(r *Run) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.ChecksSkipped = append(r.ChecksSkipped, checks...)

		return nil
	}
}

// WithSuiteEnabled enables a suite of tests
func WithSuiteEnabled(suites ...string) RunOpt {
	return func(r *Run) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.SuitesEnabled = append(r.SuitesEnabled, suites...)

		return nil
	}
}

// WithSuiteDisabled skips a suite a tests
func WithSuiteDisabled(suites ...string) RunOpt {
	return func(r *Run) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.SuitesSkipped = append(r.SuitesSkipped, suites...)

		return nil
	}
}

// WithCheckConfig passes a check specific configuration
func WithCheckConfig(check string, item string, value string) RunOpt {
	return func(r *Run) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		_, ok := r.CheckConfiguration[check]
		if !ok {
			r.CheckConfiguration[check] = map[string]string{}
		}

		r.CheckConfiguration[check][item] = value

		return nil
	}
}

// WithSuiteConfig passes a suite specific configuration
func WithSuiteConfig(suite string, item string, value string) RunOpt {
	return func(r *Run) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		_, ok := r.SuiteConfiguration[suite]
		if !ok {
			r.SuiteConfiguration[suite] = make(map[string]string)

		}

		r.SuiteConfiguration[suite][item] = value

		return nil
	}
}

func WithLogger(logger Logger) RunOpt {
	return func(r *Run) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.logger = logger

		return nil
	}
}
