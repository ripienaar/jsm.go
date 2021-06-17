// Copyright 2021 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

type Server struct {
	Info          server.ServerInfo  `json:"info"`
	Variables     server.Varz        `json:"variables"`
	JetStream     server.JSInfo      `json:"jetstream"`
	Cluster       server.ClusterInfo `json:"cluster"`
	Gateways      server.Gatewayz    `json:"gateways"`
	Routes        server.Routez      `json:"routes"`
	Connections   server.Connz       `json:"connections"`
	Accounts      server.Accountz    `json:"accounts"`
	Subscriptions server.Subsz       `json:"subscriptions"` // TODO
	Configuration []byte             `json:"configuration"` // TODO: not supported by the server now
}

func (s *Server) String() string {
	return fmt.Sprintf("%s (%s)", s.Info.Name, s.Info.ID)
}

func (r *Run) fetchServers() error {
	r.Infof("Gathering cluster state")

	err := r.fetchServerInfo()
	if err != nil {
		return err
	}

	err = r.fetchJSZ()
	if err != nil {
		return err
	}

	err = r.fetchGateways()
	if err != nil {
		return err
	}

	err = r.fetchRoutes()
	if err != nil {
		return err
	}

	err = r.fetchConns()
	if err != nil {
		return err
	}

	err = r.fetchAccounts()
	if err != nil {
		return err
	}

	r.Infof("Fetched %d servers", len(r.Servers))

	return nil
}

// TODO: make generic versions of this stuff and use them in the CLI too, remove the duplicate implementations. in reality these would need to recurse and fetch the full set of data, which below does not support

func (r *Run) fetchAccounts() error {
	r.Infof("Fetching Account info")

	type reqresp struct {
		Server server.ServerInfo `json:"server"`
		Data   server.Accountz   `json:"data"`
	}

	gzr, err := r.doReq("ACCOUNTZ", len(r.Servers), nil)
	if err != nil {
		return err
	}

	sgz := map[string]reqresp{}
	for _, res := range gzr {
		resp := reqresp{}
		err = json.Unmarshal(res, &resp)
		if err != nil {
			return err
		}

		sgz[resp.Server.ID] = resp
	}

	for _, srv := range r.Servers {
		jsz, ok := sgz[srv.Info.ID]
		if !ok {
			return fmt.Errorf("no Account data from %s", srv)
		}

		srv.Accounts = jsz.Data
	}

	return nil
}

func (r *Run) fetchConns() error {
	r.Infof("Fetching Connection info")

	type reqresp struct {
		Server server.ServerInfo `json:"server"`
		Data   server.Connz      `json:"data"`
	}

	gzr, err := r.doReq("CONNZ", len(r.Servers), server.ConnzEventOptions{
		ConnzOptions: server.ConnzOptions{
			Username:            true,
			Subscriptions:       true,
			SubscriptionsDetail: true,
		},
	})
	if err != nil {
		return err
	}

	sgz := map[string]reqresp{}
	for _, res := range gzr {
		resp := reqresp{}
		err = json.Unmarshal(res, &resp)
		if err != nil {
			return err
		}

		sgz[resp.Server.ID] = resp
	}

	for _, srv := range r.Servers {
		jsz, ok := sgz[srv.Info.ID]
		if !ok {
			return fmt.Errorf("no Connection data from %s", srv)
		}

		srv.Connections = jsz.Data
	}

	return nil
}

func (r *Run) fetchRoutes() error {
	r.Infof("Fetching Route info")

	type reqresp struct {
		Server server.ServerInfo `json:"server"`
		Data   server.Routez     `json:"data"`
	}

	gzr, err := r.doReq("ROUTEZ", len(r.Servers), nil)
	if err != nil {
		return err
	}

	sgz := map[string]reqresp{}
	for _, res := range gzr {
		resp := reqresp{}
		err = json.Unmarshal(res, &resp)
		if err != nil {
			return err
		}

		sgz[resp.Server.ID] = resp
	}

	for _, srv := range r.Servers {
		jsz, ok := sgz[srv.Info.ID]
		if !ok {
			return fmt.Errorf("no Route data from %s", srv)
		}

		srv.Routes = jsz.Data
	}

	return nil
}

func (r *Run) fetchGateways() error {
	r.Infof("Fetching Gateway info")

	type reqresp struct {
		Server server.ServerInfo `json:"server"`
		Data   server.Gatewayz   `json:"data"`
	}

	gzr, err := r.doReq("GATEWAYZ", len(r.Servers), server.GatewayzEventOptions{
		GatewayzOptions: server.GatewayzOptions{Accounts: true},
	})
	if err != nil {
		return err
	}

	sgz := map[string]reqresp{}
	for _, res := range gzr {
		resp := reqresp{}
		err = json.Unmarshal(res, &resp)
		if err != nil {
			return err
		}

		sgz[resp.Server.ID] = resp
	}

	for _, srv := range r.Servers {
		if srv.Info.Cluster == "" {
			r.Infof("Not fetching Gateway info from server with no cluster name %s", srv)
			continue
		}

		jsz, ok := sgz[srv.Info.ID]
		if !ok {
			return fmt.Errorf("no Gateway data from %s", srv)
		}

		srv.Gateways = jsz.Data
	}

	return nil
}

func (r *Run) fetchJSZ() error {
	r.Infof("Fetching JetStream info")

	type reqresp struct {
		Data   server.JSInfo     `json:"data"`
		Server server.ServerInfo `json:"server"`
	}

	jsr, err := r.doReq("JSZ", len(r.Servers), server.JszEventOptions{
		JSzOptions: server.JSzOptions{Account: "", Accounts: true, Streams: true, Consumer: true, Config: true},
	})
	if err != nil {
		return err
	}

	sjz := map[string]reqresp{}
	for _, res := range jsr {
		resp := reqresp{}
		err = json.Unmarshal(res, &resp)
		if err != nil {
			return err
		}

		sjz[resp.Server.ID] = resp
	}

	for _, srv := range r.Servers {
		if !srv.Info.JetStream {
			r.Infof("Not fetching JetStream info from disabled server %s", srv)
			continue
		}

		jsz, ok := sjz[srv.Info.ID]
		if !ok {
			return fmt.Errorf("no JetStream data from %s", srv)
		}

		srv.JetStream = jsz.Data
		if srv.JetStream.Meta.Leader == srv.Info.Name {
			srv.Cluster = *srv.JetStream.Meta
		}
	}

	return nil
}

func (r *Run) fetchServerInfo() error {
	r.Infof("Fetching general Server Info")

	type reqresp struct {
		Data   server.ServerInfo `json:"data"`
		Server json.RawMessage   `json:"server"`
	}

	infos, err := r.doReq("PING", 0, nil)
	if err != nil {
		return err
	}

	for _, sbytes := range infos {
		resp := reqresp{}
		err := json.Unmarshal(sbytes, &resp)
		if err != nil {
			return err
		}

		srv := &Server{}
		err = json.Unmarshal(resp.Server, &srv.Info)
		if err != nil {
			return err
		}

		r.Servers = append(r.Servers, srv)
	}

	r.Infof("Discovered %d servers", len(r.Servers))

	return nil
}

func (r *Run) doReq(kind string, wait int, req interface{}) ([][]byte, error) {
	jreq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	subj := fmt.Sprintf("$SYS.REQ.SERVER.PING.%s", kind)
	if kind == "PING" {
		subj = "$SYS.REQ.SERVER.PING"
	}

	var resp [][]byte
	var mu sync.Mutex
	ctr := 0

	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	sub, err := r.nc.Subscribe(nats.NewInbox(), func(m *nats.Msg) {
		mu.Lock()
		defer mu.Unlock()

		var b bytes.Buffer
		json.Indent(&b, m.Data, "", "   ")

		resp = append(resp, b.Bytes())
		ctr++

		if wait > 0 && ctr == wait {
			cancel()
		}
	})
	if err != nil {
		return nil, err
	}

	err = r.nc.PublishRequest(subj, sub.Subject, jreq)
	if err != nil {
		return nil, err
	}

	<-ctx.Done()

	return resp, nil
}
