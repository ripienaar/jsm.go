package kvtopologies

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// AggregateConfig configures an aggregate
type AggregateConfig struct {
	Bucket       string
	Writable     bool
	Description  string
	Replicas     int
	MaxValueSize int32
	History      uint8
	TTL          time.Duration
	MaxBytes     int64
	Storage      StorageType // a new kv specific storage struct, for now identical to normal one
	Placement    *Placement  // a new kv specific placement struct, for now identical to normal one
	RePublish    *RePublish  // a new kv specific replacement struct, for now identical to normal one
	Origins      []*AggregateOrigin
}

func (c AggregateConfig) history() int64        { return int64(c.History) }
func (c AggregateConfig) replicas() int         { return c.Replicas }
func (c AggregateConfig) maxBytes() int64       { return c.MaxBytes }
func (c AggregateConfig) ttl() time.Duration    { return c.TTL }
func (c AggregateConfig) storage() StorageType  { return c.Storage }
func (c AggregateConfig) placement() *Placement { return c.Placement }

type AggregateOrigin struct {
	Name     string   // note this is Stream and not Bucket since the origin may be a mirror which may not be a bucket
	Bucket   string   // when Name is a non bucket like a mirror we need to know the bucket name to construct mappings with
	Keys     []string // optional filter defaults to >
	External *External
}

// CreateAggregate creates an aggregate from a non KV stream - like a mirror - this requires the bucket name to map the subjects accordingly
func CreateAggregate(ctx context.Context, js jetstream.JetStream, cfg *AggregateConfig) (jetstream.KeyValue, error) {
	if !validBucketRe.MatchString(cfg.Bucket) {
		return nil, ErrInvalidBucketName
	}

	if cfg == nil {
		return nil, ErrKeyValueConfigRequired
	}

	if !validBucketRe.MatchString(cfg.Bucket) {
		return nil, ErrInvalidBucketName
	}

	if len(cfg.Origins) == 0 {
		return nil, ErrOriginsRequired
	}

	scfg := jetstream.StreamConfig{
		Name:        fmt.Sprintf(kvBucketNameTmpl, cfg.Bucket),
		Description: cfg.Description,
		MaxAge:      cfg.TTL,
		DenyDelete:  true,
		AllowDirect: true,
	}

	err := configureCommonSettings(cfg, &scfg)
	if err != nil {
		return nil, err
	}

	if cfg.Writable {
		scfg.AllowRollup = true
		scfg.DenyDelete = false
		scfg.Subjects = []string{fmt.Sprintf(kvSubjectsTmpl, cfg.Bucket)}
	}

	for _, o := range cfg.Origins {
		if o.Name == "" {
			return nil, ErrOriginNameRequired
		}

		name := o.Name
		bucket := o.Bucket

		if strings.HasPrefix(o.Name, kvBucketNamePre) {
			bucket = o.Name[len(kvBucketNamePre):]
		}

		if bucket == "" {
			return nil, ErrAggregateRequiresBucketName
		}

		source := jetstream.StreamSource{
			Name: name,
		}

		if o.External != nil {
			source.External = &jetstream.ExternalStream{
				APIPrefix:     o.External.ApiPrefix,
				DeliverPrefix: o.External.DeliverPrefix,
			}
		}

		keys := make([]string, len(o.Keys))
		copy(keys, o.Keys)
		if len(keys) == 0 {
			keys = append(keys, ">")
		}

		for _, key := range keys {
			transform := jetstream.SubjectTransformConfig{
				Destination: fmt.Sprintf(kvKeySubjectTmpl, cfg.Bucket, key),
				Source:      fmt.Sprintf(kvKeySubjectTmpl, bucket, key),
			}
			source.SubjectTransforms = append(source.SubjectTransforms, transform)
		}

		scfg.Sources = append(scfg.Sources, &source)
	}

	_, err = js.CreateStream(ctx, scfg)
	if err != nil {
		return nil, err
	}

	return js.KeyValue(ctx, cfg.Bucket)
}
