package jsm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func (m *Manager) GetSubsz(expected int, opts *server.SubszEventOptions) (map[string]*server.Subsz, error) {
	if opts == nil {
		opts = &server.SubszEventOptions{
			SubszOptions: server.SubszOptions{
				Subscriptions: true,
			},
			EventFilterOptions: server.EventFilterOptions{},
		}
	}

	responses, err := m.doZReq("$SYS.REQ.SERVER.PING.SUBSZ", expected, opts, func() interface{} { return &server.Subsz{} })
	if err != nil {
		return nil, err
	}

	r := map[string]*server.Subsz{}
	for _, resp := range responses {
		r[resp.Server.ID] = resp.Data.(*server.Subsz)
	}

	if expected > 0 && len(r) != expected {
		return r, fmt.Errorf("received %d / %d responses", len(r), expected)
	}

	return r, nil
}

func (m *Manager) GetVarz(expected int, opts *server.VarzEventOptions) (map[string]*server.Varz, error) {
	responses, err := m.doZReq("$SYS.REQ.SERVER.PING.VARZ", expected, opts, func() interface{} { return &server.Varz{} })
	if err != nil {
		return nil, err
	}

	r := map[string]*server.Varz{}
	for _, resp := range responses {
		r[resp.Server.ID] = resp.Data.(*server.Varz)
	}

	if expected > 0 && len(r) != expected {
		return r, fmt.Errorf("received %d / %d responses", len(r), expected)
	}

	return r, nil
}

type zResponse struct {
	Server server.ServerInfo `json:"server"`
	Data   interface{}       `json:"data"`
}

// TODO: paging
func (m *Manager) doZReq(subj string, wait int, req interface{}, respf func() interface{}) (map[string]*zResponse, error) {
	jreq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	responses := map[string]*zResponse{}
	var mu sync.Mutex
	ctr := 0

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	errs := make(chan error)

	sub, err := m.nc.Subscribe(nats.NewInbox(), func(m *nats.Msg) {
		resp := &zResponse{
			Data: respf(),
		}

		if m.Header.Get("Status") == "503" {
			errs <- fmt.Errorf("ensure a system account is used")
		}

		json.Unmarshal(m.Data, resp)

		mu.Lock()
		defer mu.Unlock()

		ctr++
		responses[resp.Server.ID] = resp

		if wait > 0 && ctr == wait {
			cancel()
		}
	})
	if err != nil {
		return nil, err
	}

	err = m.nc.PublishRequest(subj, sub.Subject, jreq)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
	case err := <-errs:
		if err != nil {
			return nil, err
		}
	}

	return responses, nil
}
