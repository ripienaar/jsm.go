package jsm_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	srv "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"

	"github.com/nats-io/jsm.go"
)

func startCluster(t *testing.T, user string, pass string) (*server.Server, *server.Server, *nats.Conn, *jsm.Manager) {
	s1, _ := srv.RunServerWithConfig("testdata/srv_a.conf")
	s2, _ := srv.RunServerWithConfig("testdata/srv_b.conf")

	for i := 0; i < 10; i++ {
		rz, err := s1.Routez(nil)
		if err != nil {
			continue
		}

		if rz.NumRoutes == 1 {
			break
		}

		time.Sleep(250 * time.Millisecond)

		if i == 9 {
			t.Fatalf("Cluster did not form: numRoutes: %d", rz.NumRoutes)
		}
	}

	nc, err := nats.Connect(s1.ClientURL(), nats.UserInfo(user, pass))
	if err != nil {
		t.Fatalf("could not connect to cluster: %s", err)
	}

	mgr, err := jsm.New(nc)
	if err != nil {
		t.Fatalf("manager failed: %s", err)
	}

	return s1, s2, nc, mgr
}

func TestManager_GetSubsz(t *testing.T) {
	s1, s2, nc, mgr := startCluster(t, "system", "s3cret")
	defer s1.Shutdown()
	defer s2.Shutdown()
	defer nc.Close()

	res, err := mgr.GetSubsz(2, nil)
	if err != nil {
		t.Fatalf("getsz failed: %s", err)
	}

	if res[s1.ID()] == nil || res[s1.ID()].ID != s1.ID() || res[s1.ID()].Total == 0 {
		t.Fatalf("did not receive results for %s", s1.ID())
	}

	if res[s2.ID()] == nil || res[s2.ID()].ID != s2.ID() || res[s2.ID()].Total == 0 {
		t.Fatalf("did not receive results for %s", s2.ID())
	}
}

func TestManager_GetVarz(t *testing.T) {
	s1, s2, nc, mgr := startCluster(t, "system", "s3cret")
	defer s1.Shutdown()
	defer s2.Shutdown()
	defer nc.Close()

	res, err := mgr.GetVarz(2, nil)
	if err != nil {
		t.Fatalf("getvz failed: %s", err)
	}

	if res[s1.ID()] == nil || res[s1.ID()].ID != s1.ID() || res[s1.ID()].Routes != 1 {
		t.Fatalf("did not receive results for %s", s1.ID())
	}

	if res[s2.ID()] == nil || res[s2.ID()].ID != s2.ID() || res[s1.ID()].Routes != 1 {
		t.Fatalf("did not receive results for %s", s2.ID())
	}
}
