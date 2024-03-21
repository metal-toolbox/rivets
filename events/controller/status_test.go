package controller

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	srvtest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/metal-toolbox/rivets/condition"
	"github.com/metal-toolbox/rivets/events"
	"github.com/metal-toolbox/rivets/events/registry"
)

func startJetStreamServer(t *testing.T) *server.Server {
	t.Helper()
	opts := srvtest.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	return srvtest.RunServer(&opts)
}

func jetStreamContext(t *testing.T, s *server.Server) (*nats.Conn, nats.JetStreamContext) {
	t.Helper()
	nc, err := nats.Connect(s.ClientURL())
	if err != nil {
		t.Fatalf("connect => %v", err)
	}
	js, err := nc.JetStream(nats.MaxWait(10 * time.Second))
	if err != nil {
		t.Fatalf("JetStream => %v", err)
	}
	return nc, js
}

func shutdownJetStream(t *testing.T, s *server.Server) {
	t.Helper()
	var sd string
	if config := s.JetStreamConfig(); config != nil {
		sd = config.StoreDir
	}
	s.Shutdown()
	if sd != "" {
		if err := os.RemoveAll(sd); err != nil {
			t.Fatalf("Unable to remove storage %q: %v", sd, err)
		}
	}
	s.WaitForShutdown()
}

func TestPublish(t *testing.T) {
	srv := startJetStreamServer(t)
	defer shutdownJetStream(t, srv)
	natsConn, jsCtx := jetStreamContext(t, srv) // nc is closed on evJS.Close(), js needs no cleanup
	evJS := events.NewJetstreamFromConn(natsConn)
	defer evJS.Close()

	cond := &condition.Condition{
		Kind: condition.FirmwareInstall,
		ID:   uuid.New(),
	}

	const facilityCode = "fac13"

	controller := &NatsController{
		stream:        evJS,
		facilityCode:  facilityCode,
		controllerID:  registry.GetID("kvtest"),
		conditionKind: cond.Kind,
		logger:        logrus.New(),
	}

	publisher, err := controller.NewNatsConditionStatusPublisher(cond.ID.String())
	require.Nil(t, err)
	require.NotNil(t, publisher, "publisher constructor")

	kv, err := jsCtx.KeyValue(string(cond.Kind))
	require.NoError(t, err, "kv read handle")

	serverID := uuid.New()

	// publish pending status
	require.NotPanics(t,
		func() {
			publisher.Publish(context.Background(), serverID.String(), condition.Pending, []byte(`{"pending...": "true"}`))
		},
		"publish pending",
	)
	require.NotEqual(t, 0, publisher.lastRev, "last rev - 1")

	entry, err := kv.Get(facilityCode + "." + cond.ID.String())
	require.Nil(t, err)
	require.Equal(t, entry.Revision(), publisher.lastRev, "last rev - 2")

	sv := &condition.StatusValue{}
	err = json.Unmarshal(entry.Value(), sv)
	require.NoError(t, err, "unmarshal")

	require.Equal(t, cond.Version, sv.MsgVersion, "version check")
	require.Equal(t, serverID.String(), sv.Target, "sv Target")
	require.Contains(t, string(sv.Status), condition.Pending, "sv Status")

	// publish active status
	require.NotPanics(t,
		func() {
			publisher.Publish(context.Background(), serverID.String(), condition.Active, []byte(`{"active...": "true"}`))
		},
		"publish active",
	)

	entry, err = kv.Get(facilityCode + "." + cond.ID.String())
	require.Nil(t, err)
	require.Equal(t, entry.Revision(), publisher.lastRev, "last rev - 3")
}

func TestConditionState(t *testing.T) {
	srv := startJetStreamServer(t)
	defer shutdownJetStream(t, srv)
	natsConn, jsCtx := jetStreamContext(t, srv) // nc is closed on evJS.Close(), js needs no cleanup
	evJS := events.NewJetstreamFromConn(natsConn)
	defer evJS.Close()

	kvName := "testKV"
	_, err := jsCtx.CreateKeyValue(&nats.KeyValueConfig{Bucket: kvName})
	require.NoError(t, err)

	kvStore, err := jsCtx.KeyValue(kvName)
	require.NoError(t, err)

	tests := []struct {
		name        string
		setup       func(string, nats.KeyValue)
		conditionID string
		expected    *condition.StatusValue
	}{
		{
			name: "Condition not started",
			setup: func(conditionID string, kv nats.KeyValue) {
				// No setup needed as no KV entry will exist
			},
		},
		{
			name: "Condition complete",
			setup: func(conditionID string, kv nats.KeyValue) {
				statusValue := &condition.StatusValue{
					State: string(condition.Succeeded),
					// Populate other required fields
				}
				data, _ := json.Marshal(statusValue)
				kv.Put("testFacility."+conditionID, data)
			},
		},
		{
			name: "Condition indeterminate (unreadable status)",
			setup: func(conditionID string, kv nats.KeyValue) {
				// Put unreadable data
				kv.Put("testFacility."+conditionID, []byte("not json"))
			},
		},
		{
			name: "Condition orphaned (missing worker data)",
			setup: func(conditionID string, kv nats.KeyValue) {
				statusValue := &condition.StatusValue{
					State:    string(condition.Pending),
					WorkerID: "missingWorker",
					// Populate other required fields
				}
				data, _ := json.Marshal(statusValue)
				kv.Put("testFacility."+conditionID, data)
			},
		},
	}

	for _, tt := range tests {
		tt.conditionID = uuid.New().String()
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.conditionID, kvStore)

			entry, err := kvStore.Get("testFacility." + tt.conditionID)
			if err == nats.ErrKeyNotFound && tt.name == "Condition not started" {
				// Expected path for not started condition
				return
			}

			require.NoError(t, err, "Expect no error fetching entry")
			var sv condition.StatusValue

			err = json.Unmarshal(entry.Value(), &sv)
			if tt.name == "Condition indeterminate (unreadable status)" {
				require.Error(t, err, "Expect error on unmarshal for indeterminate condition")
			} else {
				require.NoError(t, err, "Expect successful unmarshal for condition status")
			}
		})
	}
}
