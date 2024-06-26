package kv

import (
	"time"

	"go.hollow.sh/toolbox/events/pkg/kv"
)

func LivenessCheckinKVOpts(ttl time.Duration, replicas int) []kv.Option {
	opts := []kv.Option{
		kv.WithTTL(ttl),
		kv.WithReplicas(replicas),
	}

	return opts
}
