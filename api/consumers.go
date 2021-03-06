// Copyright 2020 The NATS Authors
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

package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

const (
	JetStreamCreateConsumerT          = "$JS.STREAM.%s.CONSUMER.%s.CREATE"
	JetStreamCreateEphemeralConsumerT = "$JS.STREAM.%s.EPHEMERAL.CONSUMER.CREATE"
	JetStreamConsumersT               = "$JS.STREAM.%s.CONSUMERS"
	JetStreamConsumerInfoT            = "$JS.STREAM.%s.CONSUMER.%s.INFO"
	JetStreamDeleteConsumerT          = "$JS.STREAM.%s.CONSUMER.%s.DELETE"
	JetStreamRequestNextT             = "$JS.STREAM.%s.CONSUMER.%s.NEXT"
	JetStreamMetricConsumerAckPre     = JetStreamMetricPrefix + ".CONSUMER_ACK"
)

type AckPolicy string

func (p AckPolicy) String() string { return strings.Title(string(p)) }

const (
	AckNone     AckPolicy = "none"
	AckAll      AckPolicy = "all"
	AckExplicit AckPolicy = "explicit"
)

func (p *AckPolicy) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case jsonString("none"):
		*p = AckNone
	case jsonString("all"):
		*p = AckAll
	case jsonString("explicit"):
		*p = AckExplicit
	default:
		return fmt.Errorf("can not unmarshal %q", data)
	}

	return nil
}

type ReplayPolicy string

func (p ReplayPolicy) String() string { return strings.Title(string(p)) }

const (
	ReplayInstant  ReplayPolicy = "instant"
	ReplayOriginal ReplayPolicy = "original"
)

func (p *ReplayPolicy) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case jsonString("instant"):
		*p = ReplayInstant
	case jsonString("original"):
		*p = ReplayOriginal
	default:
		return fmt.Errorf("can not unmarshal %q", data)
	}

	return nil
}

var (
	AckAck      = []byte(OK)
	AckNak      = []byte("-NAK")
	AckProgress = []byte("+WPI")
	AckNext     = []byte("+NXT")
)

type DeliverPolicy string

func (p DeliverPolicy) String() string { return strings.Title(string(p)) }

const (
	DeliverAll             DeliverPolicy = "all"
	DeliverLast            DeliverPolicy = "last"
	DeliverNew             DeliverPolicy = "new"
	DeliverByStartSequence DeliverPolicy = "by_start_sequence"
	DeliverByStartTime     DeliverPolicy = "by_start_time"
)

func (p *DeliverPolicy) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case jsonString("all"), jsonString("undefined"):
		*p = DeliverAll
	case jsonString("last"):
		*p = DeliverLast
	case jsonString("new"):
		*p = DeliverNew
	case jsonString("by_start_sequence"):
		*p = DeliverByStartSequence
	case jsonString("by_start_time"):
		*p = DeliverByStartTime
	}

	return nil
}

// ConsumerConfig is the configuration for a JetStream consumes
//
// NATS Schema Type io.nats.jetstream.api.v1.consumer_configuration
type ConsumerConfig struct {
	Durable         string        `json:"durable_name,omitempty"`
	DeliverSubject  string        `json:"deliver_subject,omitempty"`
	DeliverPolicy   DeliverPolicy `json:"deliver_policy"`
	OptStartSeq     uint64        `json:"opt_start_seq,omitempty"`
	OptStartTime    *time.Time    `json:"opt_start_time,omitempty"`
	AckPolicy       AckPolicy     `json:"ack_policy"`
	AckWait         time.Duration `json:"ack_wait,omitempty"`
	MaxDeliver      int           `json:"max_deliver,omitempty"`
	FilterSubject   string        `json:"filter_subject,omitempty"`
	ReplayPolicy    ReplayPolicy  `json:"replay_policy"`
	SampleFrequency string        `json:"sample_freq,omitempty"`
}

// SchemaID is the url to the JSON Schema for JetStream Consumer Configuration
func (c ConsumerConfig) SchemaID() string {
	return "https://nats.io/schemas/jetstream/api/v1/consumer_configuration.json"
}

// SchemaType is the NATS schema type like io.nats.jetstream.api.v1.stream_configuration
func (c ConsumerConfig) SchemaType() string {
	return "io.nats.jetstream.api.v1.consumer_configuration"
}

// Schema is a Draft 7 JSON Schema for the JetStream Consumer Configuration
func (c ConsumerConfig) Schema() []byte {
	return schemas[c.SchemaType()]
}

func (c ConsumerConfig) Validate() (bool, []string) {
	sl := gojsonschema.NewSchemaLoader()
	sl.AddSchema("https://nats.io/schemas/jetstream/api/v1/definitions.json", gojsonschema.NewBytesLoader(schemas["io.nats.jetstream.api.v1.definitions"]))
	root := gojsonschema.NewBytesLoader(c.Schema())

	js, err := sl.Compile(root)
	if err != nil {
		return false, []string{err.Error()}
	}

	doc := gojsonschema.NewGoLoader(c)

	result, err := js.Validate(doc)
	if err != nil {
		return false, []string{err.Error()}
	}

	if result.Valid() {
		return true, nil
	}

	errors := make([]string, len(result.Errors()))
	for i, verr := range result.Errors() {
		errors[i] = verr.String()
	}

	return false, errors
}

type CreateConsumerRequest struct {
	Stream string         `json:"stream_name"`
	Config ConsumerConfig `json:"config"`
}

type ConsumerState struct {
	Delivered   SequencePair      `json:"delivered"`
	AckFloor    SequencePair      `json:"ack_floor"`
	Pending     map[uint64]int64  `json:"pending"`
	Redelivered map[uint64]uint64 `json:"redelivered"`
}

type SequencePair struct {
	ConsumerSeq uint64 `json:"consumer_seq"`
	StreamSeq   uint64 `json:"stream_seq"`
}

type ConsumerInfo struct {
	Stream string         `json:"stream_name"`
	Name   string         `json:"name"`
	Config ConsumerConfig `json:"config"`
	State  ConsumerState  `json:"state"`
}

func jsonString(s string) string {
	return "\"" + s + "\""
}
