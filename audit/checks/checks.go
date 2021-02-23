package checks

import (
	"log"
	"sync"

	"github.com/nats-io/nats-server/v2/server"
)

type Server interface {
	Name() string
	Id() string
	Varz() *server.Varz
}

type Outcome uint

const (
	UnknownOutcome Outcome = iota
	OKOutcome
	WarningOutcome
	CriticalOutcome
	SkippedOutcome
)

type Result struct {
	Check          string  `json:"check"`
	Outcome        Outcome `json:"outcome"`
	Description    string  `json:"description"`
	Recommendation string  `json:"recommendation"`
	URL            string  `json:"url"`
}

type ServerAuditor interface {
	Audit(Server, chan *Result) error
	Name() string
}

type FleetAuditor interface {
	Audit([]Server, chan *Result) error
	Name() string
}

var (
	serverAuditors   []ServerAuditor
	fleetAuditors    []FleetAuditor
	disabledAuditors []string
	mu               sync.Mutex
)

func DisableAuditor(a ...string) {
	mu.Lock()
	disabledAuditors = append(disabledAuditors, a...)
	mu.Unlock()
}

func IsDisabled(auditor string) bool {
	mu.Lock()
	defer mu.Unlock()

	for _, a := range disabledAuditors {
		if a == auditor {
			return true
		}
	}

	return false
}

func RegisterServerAuditor(a ServerAuditor) {
	log.Printf("Registering server auditor %s", a.Name())
	mu.Lock()
	serverAuditors = append(serverAuditors, a)
	mu.Unlock()
}

func RegisterFleetAuditor(a FleetAuditor) {
	log.Printf("Registering fleet auditor %s", a.Name())
	mu.Lock()
	fleetAuditors = append(fleetAuditors, a)
	mu.Unlock()
}
