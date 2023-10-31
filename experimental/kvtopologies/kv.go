package kvtopologies

import (
	"errors"
	"regexp"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	KeyValueMaxHistory = 64
	AllKeys            = ">"
	kvBucketNameTmpl   = "KV_%s"
	kvBucketNamePre    = "KV_"
	kvSubjectsTmpl     = "$KV.%s.>"
	kvSubjectsPreTmpl  = "$KV.%s."
	kvKeySubjectTmpl   = "$KV.%s.%s"
)

var (
	validBucketRe = regexp.MustCompile(`\A[a-zA-Z0-9_-]+\z`)
	validKeyRe    = regexp.MustCompile(`\A[-/_=\.a-zA-Z0-9]+\z`)

	ErrInvalidBucketName           = errors.New("invalid bucket name")
	ErrKeyValueConfigRequired      = errors.New("nats: config required")
	ErrHistoryToLarge              = errors.New("nats: history limited to a max of 64")
	ErrUnsupportedStorage          = errors.New("nats: unsupported storage type")
	ErrOriginsRequired             = errors.New("nats: origin configuration is required")
	ErrOriginNameRequired          = errors.New("nats: origin name is required")
	ErrAggregateRequiresBucketName = errors.New("nats: creating an aggregate requires origin bucket name")
	ErrMirrorNameIsKV              = errors.New("nats: mirror names may not start with KV_")
)

// External allows you to qualify access to a stream source in another account when backed by JetStream
type External struct {
	ApiPrefix     string
	DeliverPrefix string
}

// Placement describes bucket placement requirements for a bucket when backed by JetStream
type Placement struct {
	Cluster string
	Tags    []string
}

type StorageType int

const (
	FileStorage StorageType = iota
	MemoryStorage
)

type RePublish struct {
	Source      string
	Destination string
	HeadersOnly bool
}

type genericTopology interface {
	history() int64
	replicas() int
	maxBytes() int64
	ttl() time.Duration
	storage() StorageType
	placement() *Placement
}

func configureCommonSettings(from genericTopology, scfg *jetstream.StreamConfig) error {
	if from.history() > KeyValueMaxHistory {
		return ErrHistoryToLarge
	}

	scfg.MaxMsgsPerSubject = from.history()
	if scfg.MaxMsgsPerSubject == 0 {
		scfg.MaxMsgsPerSubject = 1
	}

	scfg.Replicas = from.replicas()
	if scfg.Replicas == 0 {
		scfg.Replicas = 1
	}

	scfg.MaxBytes = from.maxBytes()
	if scfg.MaxBytes == 0 {
		scfg.MaxBytes = -1
	}

	scfg.Duplicates = 2 * time.Minute
	if from.ttl() > 0 && from.ttl() < scfg.Duplicates {
		scfg.Duplicates = from.ttl()
	}

	switch from.storage() {
	case MemoryStorage:
		scfg.Storage = jetstream.FileStorage
	case FileStorage:
		scfg.Storage = jetstream.FileStorage
	default:
		return ErrUnsupportedStorage
	}

	placement := from.placement()
	if placement != nil {
		scfg.Placement = &jetstream.Placement{
			Cluster: placement.Cluster,
			Tags:    placement.Tags,
		}
	}

	scfg.Discard = jetstream.DiscardNew
	scfg.MaxMsgs = -1
	scfg.MaxConsumers = -1

	return nil
}
