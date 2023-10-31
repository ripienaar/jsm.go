package kvtopologies

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type MirrorConfig struct {
	Name        string
	Description string
	Replicas    int
	History     uint8
	TTL         time.Duration
	MaxBytes    int64
	Storage     StorageType
	Placement   *Placement

	OriginBucket string
	Keys         []string // optional filter defaults to >
	External     *External
}

func (c MirrorConfig) history() int64        { return int64(c.History) }
func (c MirrorConfig) replicas() int         { return c.Replicas }
func (c MirrorConfig) maxBytes() int64       { return c.MaxBytes }
func (c MirrorConfig) ttl() time.Duration    { return c.TTL }
func (c MirrorConfig) storage() StorageType  { return c.Storage }
func (c MirrorConfig) placement() *Placement { return c.Placement }

func CreateMirror(ctx context.Context, js jetstream.JetStream, cfg MirrorConfig) error {
	if strings.HasPrefix(cfg.Name, kvBucketNamePre) {
		return ErrMirrorNameIsKV
	}

	scfg := jetstream.StreamConfig{
		Name:         cfg.Name,
		Description:  cfg.Description,
		MaxAge:       cfg.TTL,
		MirrorDirect: true,
	}

	err := configureCommonSettings(cfg, &scfg)
	if err != nil {
		return err
	}

	name := cfg.OriginBucket
	bucket := ""
	if strings.HasPrefix(name, kvBucketNamePre) {
		bucket = cfg.OriginBucket[len(kvBucketNamePre):]
	} else {
		bucket = cfg.OriginBucket
		name = fmt.Sprintf(kvBucketNameTmpl, name)
	}

	scfg.Mirror = &jetstream.StreamSource{
		Name: name,
	}

	if cfg.External != nil {
		scfg.Mirror.External = &jetstream.ExternalStream{
			APIPrefix:     cfg.External.ApiPrefix,
			DeliverPrefix: cfg.External.DeliverPrefix,
		}
	}

	for _, key := range cfg.Keys {
		transform := jetstream.SubjectTransformConfig{
			Destination: fmt.Sprintf(kvKeySubjectTmpl, bucket, key),
			Source:      fmt.Sprintf(kvKeySubjectTmpl, bucket, key),
		}
		scfg.Mirror.SubjectTransforms = append(scfg.Mirror.SubjectTransforms, transform)
	}

	_, err = js.CreateStream(ctx, scfg)
	return err
}
