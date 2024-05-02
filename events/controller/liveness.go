package controller

import (
	"context"
	"sync"
	"time"

	"github.com/metal-toolbox/rivets/events"
	"github.com/metal-toolbox/rivets/events/pkg/kv"
	"github.com/metal-toolbox/rivets/events/registry"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

var (
	once sync.Once

	checkinLivenessTTL = 3 * time.Minute
)

func (n *NatsController) checkinKVOpts() []kv.Option {
	opts := []kv.Option{
		kv.WithTTL(checkinLivenessTTL),
		kv.WithReplicas(n.natsConfig.KVReplicationFactor),
	}

	return opts
}

// This starts a go-routine to peridically check in with the NATS kv
func (n *NatsController) startLivenessCheckin(ctx context.Context) {
	once.Do(func() {
		n.controllerID = registry.GetID(n.hostname)

		if err := registry.InitializeRegistryWithOptions(n.stream.(*events.NatsJetstream), n.checkinKVOpts()...); err != nil {
			metricsNATSError("initialize liveness registry")
			n.logger.WithError(err).Error("unable to initialize active controller registry")
			return
		}

		go n.checkinRoutine(ctx)
	})
}

func (n *NatsController) checkinRoutine(ctx context.Context) {
	if err := registry.RegisterController(n.controllerID); err != nil {
		n.logger.WithError(err).Warn("unable to do initial controller liveness registration")
	}

	tick := time.NewTicker(n.checkinInterval)
	defer tick.Stop()

	var stop bool
	for !stop {
		select {
		case <-tick.C:
			err := registry.ControllerCheckin(n.controllerID)
			if err != nil {
				n.logger.WithError(err).
					WithField("id", n.controllerID.String()).
					Warn("controller check-in failed")
				metricsNATSError("liveness check-in")
				if err = refreshControllerToken(n.controllerID); err != nil {
					n.logger.WithError(err).
						WithField("id", n.controllerID.String()).
						Fatal("unable to refresh controller liveness token")
				}
			}
		case <-ctx.Done():
			n.logger.Info("liveness check-in stopping on done context")
			stop = true
		}
	}
}

// try to de-register/re-register this id.
func refreshControllerToken(id registry.ControllerID) error {
	err := registry.DeregisterController(id)
	if err != nil && !errors.Is(err, nats.ErrKeyNotFound) {
		metricsNATSError("liveness refresh: de-register")
		return err
	}
	err = registry.RegisterController(id)
	if err != nil {
		metricsNATSError("liveness refresh: register")
		return err
	}
	return nil
}
