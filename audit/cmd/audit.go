package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nats-io/jsm.go/audit"
	"github.com/nats-io/jsm.go/natscontext"
)

func main() {
	nctx, err := natscontext.New(os.Getenv("CONTEXT"), true)
	if err != nil {
		panic(err)
	}

	opts, err := nctx.NATSOptions()
	if err != nil {
		panic(err)
	}

	nc, err := nats.Connect(nctx.ServerURL(), opts...)
	if err != nil {
		panic(err)
	}

	run, err := audit.NewRun("test",
		audit.WithNatsConnection(nc),
		audit.WithNetworkTimeout(5*time.Second),
		// audit.WithCheckConfig("tls_timeout", "threshold", "0"),
		// audit.WithCheckDisabled("tls_required"),
	)
	if err != nil {
		panic(err)
	}

	run.Run()

	fmt.Printf("checks: passed: %d failed: %d asserts: %d\n", run.ChecksPassed, run.ChecksFailed, run.AssertionsRan)
	fmt.Printf("suites: passed: %d failed: %d\n", run.SuitesPassed, run.SuitesFailed)

	rj, _ := json.Marshal(run)
	ioutil.WriteFile("run.json", rj, 0644)
}
