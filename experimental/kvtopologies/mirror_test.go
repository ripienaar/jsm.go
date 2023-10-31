package kvtopologies

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestCreateMirror(t *testing.T) {
	srv, nc, js := startJSServer(t)
	defer srv.Shutdown()
	defer nc.Flush()

	var orders jetstream.KeyValue

	setup := func(t *testing.T) {
		t.Helper()

		var err error
		orders, err = js.CreateKeyValue(context.TODO(), jetstream.KeyValueConfig{
			Bucket: "ORDERS",
		})
		if err != nil {
			t.Fatalf("orders creation failed: %v", err)
		}

		_, err = orders.Put(context.TODO(), "HELLO", []byte("world"))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}

	t.Run("Validations", func(t *testing.T) {
		err := CreateMirror(context.TODO(), js, MirrorConfig{Name: "KV_MIRROR", OriginBucket: "x"})
		if !errors.Is(err, ErrMirrorNameIsKV) {
			t.Fatalf("Expected ErrMirrorNameIsKV, got: %v", err)
		}
	})

	t.Run("Unfiltered", func(t *testing.T) {
		setup(t)

		err := CreateMirror(context.TODO(), js, MirrorConfig{Name: "ORDERS_MIRROR", OriginBucket: "KV_ORDERS"})
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		mirror, err := js.Stream(context.TODO(), "ORDERS_MIRROR")
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}
		nfo, err := mirror.Info(context.TODO())
		if err != nil {
			t.Fatalf("mirror info failed: %v", err)
		}
		if nfo.State.Msgs != 1 {
			t.Fatalf("expected 1 message got %v", nfo.State.Msgs)
		}

		// delete the origin so that the mirror answers the direct get thus testing it works
		err = js.DeleteStream(context.TODO(), "KV_ORDERS")
		if err != nil {
			t.Fatalf("delete failed: %v", err)
		}

		e, err := orders.Get(context.TODO(), "HELLO")
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if string(e.Value()) != "world" {
			t.Fatalf("invalid value")
		}
	})

	t.Run("Filtered", func(t *testing.T) {
		setup(t)

		_, err := orders.Put(context.TODO(), "NEW.123", nil)
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}

		err = CreateMirror(context.TODO(), js, MirrorConfig{
			Name:         "ORDERS_MIRROR",
			OriginBucket: "ORDERS",
			Keys:         []string{"NEW.>"},
		})
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}

		time.Sleep(250 * time.Millisecond)

		mirror, err := js.Stream(context.TODO(), "ORDERS_MIRROR")
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}

		nfo, err := mirror.Info(context.TODO(), jetstream.WithSubjectFilter(">"))
		if err != nil {
			t.Fatalf("info failed: %v", err)
		}

		if !reflect.DeepEqual(nfo.State.Subjects, map[string]uint64{"$KV.ORDERS.NEW.123": 1}) {
			t.Fatalf("Expected $KV.ORDERS.NEW.123:1 got %v", nfo.State.Subjects)
		}
	})
}
