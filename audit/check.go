package audit

import (
	"fmt"
	"time"
)

type Check struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	ScriptPath    string            `json:"path"`
	Suite         string            `json:"suite"`
	Enable        bool              `json:"enable"`
	Configuration map[string]string `json:"configuration"`
	Core          bool              `json:"core"`
	Clustered     bool              `json:"clustered"`

	serverCheck  func(s *Server, result *CheckResult, log Logger) error
	clusterCheck func(server []*Server, result *CheckResult, log Logger) error
}

type CheckOutcome string

var (
	UnknownOutcome CheckOutcome = "unknown"
	OKOutcome      CheckOutcome = "ok"
	ErrorOutcome   CheckOutcome = "error"
)

type CheckResult struct {
	CheckName      string        `json:"check_name"`
	CheckVersion   string        `json:"check_version"`
	TimeStamp      time.Time     `json:"time_stamp"`
	RunTime        time.Duration `json:"run_time"`
	Skipped        bool          `json:"skipped"`
	Error          string        `json:"error,omitempty"`
	Outcome        CheckOutcome  `json:"outcome"`
	AssertionCount int           `json:"assertions"`
	ServerName     string        `json:"server_name,omitempty"`
	ServerID       string        `json:"server_id,omitempty"`
	ServerCluster  string        `json:"cluster_name,omitempty"`
	ErrorMessages  []string      `json:"errors,omitempty"`
	Messages       []string      `json:"messages,omitempty"`
	Metadata       interface{}   `json:"metadata,omitempty"`

	log Logger
}

func (c *Check) newResult(log Logger) *CheckResult {
	return &CheckResult{
		CheckName:    c.Name,
		CheckVersion: c.Version,
		TimeStamp:    time.Now().UTC(),
		Outcome:      UnknownOutcome,
		log:          log,
	}
}

func (r *CheckResult) Assert(f func()) {
	// earlier Assert set to skip, skip all further asserts
	if r.Skipped {
		return
	}

	// earlier Assert had a critical error, skip all further asserts
	if r.Error != "" {
		return
	}

	f()

	r.AssertionCount++
}

func (r *CheckResult) ErrorF(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	r.ErrorMessages = append(r.ErrorMessages, msg)
	r.log.Errorf(msg)
}

func (r *CheckResult) finalize() {
	if r.TimeStamp.IsZero() {
		return
	}

	r.RunTime = time.Since(r.TimeStamp)
}
