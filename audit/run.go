package audit

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
)

const version = "1.0.0"

// Run is a single run of an audit suite
type Run struct {
	Name      string        `json:"audit_name"`
	ID        string        `json:"id"`
	StartTime time.Time     `json:"time_stamp"`
	RunTime   time.Duration `json:"time"`
	Error     string        `json:"error"`

	SuiteConfiguration map[string]map[string]string `json:"suite_configuration"`
	CheckConfiguration map[string]map[string]string `json:"check_configuration"`

	Timeout           time.Duration `json:"timeout"`
	ConnectedServerID string        `json:"connected_server"`
	Servers           []*Server     `json:"servers"`

	SuitesSkipped []string                  `json:"suites_skipped"`
	SuitesEnabled []string                  `json:"suites_enabled"`
	SuitesFailed  int                       `json:"suites_failed"`
	SuitesPassed  int                       `json:"suites_passed"`
	SuitesRan     int                       `json:"suites_ran"`
	SuiteResults  map[string][]*CheckResult `json:"suite_results"`

	ChecksSkipped []string `json:"checks_skipped"`
	ChecksEnabled []string `json:"checks_enabled"`
	ChecksFailed  int      `json:"checks_failed"`
	ChecksPassed  int      `json:"checks_passed"`
	ChecksRan     int      `json:"checks_ran"`

	AssertionsRan int `json:"assertions_ran"`

	Logs []LogLine `json:"logs"`

	suites map[string]*Suite

	logger Logger
	nc     *nats.Conn
	mu     sync.Mutex
}

func NewRun(name string, opts ...RunOpt) (*Run, error) {
	run := &Run{
		Name:               name,
		ID:                 nuid.Next(),
		SuiteConfiguration: map[string]map[string]string{},
		CheckConfiguration: map[string]map[string]string{},
		Servers:            []*Server{},
		SuitesSkipped:      []string{},
		SuitesEnabled:      []string{},
		SuiteResults:       map[string][]*CheckResult{},
		ChecksSkipped:      []string{},
		ChecksEnabled:      []string{},
		Timeout:            5 * time.Second,
		suites:             make(map[string]*Suite),
		mu:                 sync.Mutex{},
	}

	for _, opt := range opts {
		err := opt(run)
		if err != nil {
			return nil, err
		}
	}

	run.suites["tls"] = newTLSSuite(run)
	run.suites["cluster"] = newClusterSuite(run)

	return run, nil
}

// Run runs the audit run updating the status as it progress
func (r *Run) Run() {
	if r.nc == nil {
		r.Error = "no nats connection given"
		return
	}

	r.StartTime = time.Now()
	r.Infof("Starting audit %s", r.Name)
	defer r.finalize()

	r.ConnectedServerID = r.nc.ConnectedServerId()

	err := r.fetchServers()
	if err != nil {
		r.Errorf("Run failed: %s", err)
		r.Error = err.Error()
		return
	}

	r.runSuites()
}

func (r *Run) runSuites() {
	for _, suite := range r.suites {
		if r.isSuiteSkipped(suite.Name) {
			r.Infof("Skipping suite %s", suite.Name)
			continue
		}

		r.SuitesRan++
		if suite.runChecks() == OKOutcome {
			r.SuitesPassed++
		} else {
			r.SuitesFailed++
		}

		r.SuiteResults[suite.Name] = suite.results

		r.ChecksFailed += suite.checksFailed()
		r.ChecksPassed += suite.checksPassed()
		r.ChecksRan += suite.checksRan()
		r.AssertionsRan += suite.assertions()
	}
}

func (r *Run) suiteConfig(suite string) map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, ok := r.SuiteConfiguration[suite]
	if ok {
		return cfg
	}

	return make(map[string]string)
}

func (r *Run) checkConfig(check string) map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, ok := r.CheckConfiguration[check]
	if ok {
		return cfg
	}

	return make(map[string]string)
}

func (r *Run) isSuiteSkipped(suite string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.SuitesSkipped {
		if s == suite {
			return true
		}
	}

	return false
}

func (r *Run) isCheckSkipped(check string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, c := range r.ChecksSkipped {
		if c == check {
			return true
		}
	}

	return false
}

func (r *Run) finalize() {
	r.RunTime = time.Since(r.StartTime)
	r.Infof("Finishing audit %s", r.Name)
}
