// The registry package builds functionality for tracking live controller processes
// in a NATS KV store. The store is configured with replication and with a
// default 3 minute TTL for keys. The rationale here is that faulting workers
// will be reaped by the system, making it easy to determine which workers are
// active.
//
//nolint:wsl // useless
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/metal-toolbox/rivets/events"
	"github.com/metal-toolbox/rivets/events/pkg/kv"
)

var (
	registry nats.KeyValue

	RegistryName  = "active-controllers"
	registryTTL   = 3 * time.Minute
	replicaCount  = 3
	kvDescription = "a list of active controllers in the system"

	ErrRegistryUninitialized         = errors.New("controller registry uninitialized")
	ErrRegistryPreviouslyInitialized = errors.New("controller registry previously initialized")
	ErrBadRegistryData               = errors.New("bad registry data")
)

func InitializeActiveControllerRegistry(njs *events.NatsJetstream) error {
	return InitializeRegistryWithOptions(njs,
		kv.WithReplicas(replicaCount),
		kv.WithDescription(kvDescription),
		kv.WithTTL(registryTTL),
	)
}

// SetHandle is a helper method which assigns the nats.KeyValue to the pkg global registry
func SetHandle(handle nats.KeyValue) error {
	if registry != nil {
		return ErrRegistryPreviouslyInitialized
	}

	registry = handle

	return nil
}

// XXX: You probably don't want the un-opinionated one, but it's here.
func InitializeRegistryWithOptions(njs *events.NatsJetstream, opts ...kv.Option) error {
	if registry != nil {
		return ErrRegistryPreviouslyInitialized
	}
	var err error
	registry, err = kv.CreateOrBindKVBucket(njs, RegistryName, opts...)
	return err
}

func proofOfLife() ([]byte, error) {
	active := &activityRecord{
		LastActive: time.Now(),
	}
	return json.Marshal(active)
}

func RegisterController(id ControllerID) error {
	if registry == nil {
		return ErrRegistryUninitialized
	}
	active, err := proofOfLife()
	if err != nil {
		return err
	}
	rev, err := registry.Create(id.String(), active)
	if err == nil {
		id.updateVersion(rev)
	}
	return err
}

func ControllerCheckin(id ControllerID) error {
	if registry == nil {
		return ErrRegistryUninitialized
	}
	active, err := proofOfLife()
	if err != nil {
		return err
	}

	// The controller current revision does not match the existing
	// TODO: im not sure how this occured but this works around the case
	// where a controller check in is refused because the revision doesn't match.
	curr, err := registry.Get(id.String())
	if err != nil {
		return err
	}

	var currRev uint64
	if curr.Revision() != id.version() {
		fmt.Printf("revisions did not match KV=%d != recorded=%d\n", curr.Revision(), id.version())
		currRev = curr.Revision()
	}

	rev, err := registry.Update(id.String(), active, currRev)
	if err == nil {
		id.updateVersion(rev)
	}
	return err
}

func DeregisterController(id ControllerID) error {
	if registry == nil {
		return ErrRegistryUninitialized
	}
	return registry.Delete(id.String())
}

func LastContact(id ControllerID) (time.Time, error) {
	var zt time.Time
	if registry == nil {
		return zt, ErrRegistryUninitialized
	}
	entry, err := registry.Get(id.String())
	if err != nil {
		return zt, err // this can either be a communication error or nats.ErrKeyNotFound
	}
	// if we have an entry the controller was alive in the last TTL period
	var ar activityRecord
	err = json.Unmarshal(entry.Value(), &ar)
	if err != nil {
		return zt, ErrBadRegistryData // consumers should *probably* treat this as a success?
	}
	return ar.LastActive, nil
}
