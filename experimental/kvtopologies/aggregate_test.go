package kvtopologies

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	natsd "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestCreateAggregate(t *testing.T) {
	srv, nc, js := startJSServer(t)
	defer srv.Shutdown()
	defer nc.Flush()

	orders, err := js.CreateKeyValue(context.TODO(), jetstream.KeyValueConfig{
		Bucket: "ORDERS",
	})
	if err != nil {
		t.Fatalf("orders creation failed: %v", err)
	}

	_, err = orders.Put(context.TODO(), "HELLO", []byte("world"))
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	t.Run("Read Only", func(t *testing.T) {
		t.Cleanup(func() {
			js.DeleteStream(context.TODO(), "KV_ORDERS_NEW")
		})

		newOrders, err := CreateAggregate(context.TODO(), js, &AggregateConfig{
			Bucket: "ORDERS_NEW",
			Origins: []*AggregateOrigin{
				{
					Name: "KV_ORDERS",
				},
			},
		})
		if err != nil {
			t.Fatalf("origin creation failed: %v", err)
		}

		if newOrders.Bucket() != "ORDERS_NEW" {
			t.Fatalf("Bound to wrong bucket %v", newOrders.Bucket())
		}

		// silly sleep to let replication happen, probably a flapper
		time.Sleep(50 * time.Millisecond)

		entry, err := newOrders.Get(context.TODO(), "HELLO")
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if !bytes.Equal(entry.Value(), []byte("world")) {
			t.Fatalf("Invalid value %q", entry.Value())
		}

		_, err = newOrders.Put(context.TODO(), "X", nil)
		if !errors.Is(err, jetstream.ErrNoStreamResponse) {
			t.Fatalf("Expected ErrNoResponders got %v", err)
		}
	})

	t.Run("Read Write", func(t *testing.T) {
		t.Cleanup(func() {
			js.DeleteStream(context.TODO(), "KV_ORDERS_NEW")
		})

		newOrders, err := CreateAggregate(context.TODO(), js, &AggregateConfig{
			Bucket:   "ORDERS_NEW",
			Writable: true,
			Origins: []*AggregateOrigin{
				{
					Name: "KV_ORDERS",
				},
			},
		})
		if err != nil {
			t.Fatalf("origin creation failed: %v", err)
		}

		if newOrders.Bucket() != "ORDERS_NEW" {
			t.Fatalf("Bound to wrong bucket %v", newOrders.Bucket())
		}

		// silly sleep to let replication happen, probably a flapper
		time.Sleep(50 * time.Millisecond)

		entry, err := newOrders.Get(context.TODO(), "HELLO")
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if !bytes.Equal(entry.Value(), []byte("world")) {
			t.Fatalf("Invalid value %q", entry.Value())
		}

		_, err = newOrders.Put(context.TODO(), "X", nil)
		if err != nil {
			t.Fatalf("Expected write to pass got: %v", err)
		}
	})

	t.Run("Aggregate just some keys", func(t *testing.T) {
		t.Cleanup(func() {
			js.DeleteStream(context.TODO(), "KV_ORDERS_NEW")
		})

		newOrders, err := CreateAggregate(context.TODO(), js, &AggregateConfig{
			Bucket:   "ORDERS_NEW",
			Writable: true,
			Origins: []*AggregateOrigin{
				{
					Name: "KV_ORDERS",
					Keys: []string{"new.>"},
				},
			},
		})
		if err != nil {
			t.Fatalf("origin creation failed: %v", err)
		}

		if newOrders.Bucket() != "ORDERS_NEW" {
			t.Fatalf("Bound to wrong bucket %v", newOrders.Bucket())
		}

		_, err = orders.Put(context.TODO(), "new.123", []byte("new"))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
		_, err = orders.Put(context.TODO(), "shipped.123", []byte("new"))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}

		// silly sleep to let replication happen, probably a flapper
		time.Sleep(50 * time.Millisecond)

		keys, err := newOrders.Keys(context.TODO())
		if err != nil {
			t.Fatalf("keys failed: %v", err)
		}

		if len(keys) != 1 {
			t.Fatalf("Invalid keys: %v", keys)
		}
		if keys[0] != "new.123" {
			t.Fatalf("Invalid keys: %v", keys)
		}
	})

	t.Run("Aggregate from non bucket", func(t *testing.T) {
		_, err := js.CreateStream(context.TODO(), jetstream.StreamConfig{
			Name: "MIRROR",
			Mirror: &jetstream.StreamSource{
				Name: "KV_ORDERS",
			},
		})
		if err != nil {
			t.Fatalf("mirror create failed: %v", err)
		}

		newOrders, err := CreateAggregate(context.TODO(), js, &AggregateConfig{
			Bucket:   "ORDERS_NEW",
			Writable: true,
			Origins: []*AggregateOrigin{
				{
					Name:   "MIRROR",
					Bucket: "ORDERS",
				},
			},
		})
		if err != nil {
			t.Fatalf("origin creation failed: %v", err)
		}

		if newOrders.Bucket() != "ORDERS_NEW" {
			t.Fatalf("Bound to wrong bucket %v", newOrders.Bucket())
		}

		// silly sleep to let replication happen, probably a flapper
		time.Sleep(50 * time.Millisecond)

		entry, err := newOrders.Get(context.TODO(), "HELLO")
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		if !bytes.Equal(entry.Value(), []byte("world")) {
			t.Fatalf("Invalid value %q", entry.Value())
		}

		_, err = newOrders.Put(context.TODO(), "X", nil)
		if err != nil {
			t.Fatalf("Expected write to pass got: %v", err)
		}
	})
}

func startJSServer(t *testing.T) (*natsd.Server, *nats.Conn, jetstream.JetStream) {
	t.Helper()

	d, err := os.MkdirTemp("", "jstest")
	if err != nil {
		t.Fatalf("temp dir could not be made: %s", err)
	}

	opts := &natsd.Options{
		JetStream: true,
		StoreDir:  d,
		Port:      -1,
		Host:      "localhost",
		LogFile:   "/dev/stdout",
		Trace:     true,
	}

	s, err := natsd.NewServer(opts)
	if err != nil {
		t.Fatal("server start failed: ", err)
	}

	go s.Start()
	if !s.ReadyForConnections(10 * time.Second) {
		t.Error("nats server did not start")
	}

	nc, err := nats.Connect(s.ClientURL(), nats.UseOldRequestStyle())
	if err != nil {
		t.Fatalf("client start failed: %s", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("manager creation failed: %s", err)
	}

	return s, nc, js
}
