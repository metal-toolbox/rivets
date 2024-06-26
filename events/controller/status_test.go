package controller

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	srvtest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	opts.StoreDir = t.TempDir()
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
	s.Shutdown()
	s.WaitForShutdown()
}

func TestNewNatsConditionStatusPublisher(t *testing.T) {
	srv := startJetStreamServer(t)
	defer shutdownJetStream(t, srv)
	natsConn, _ := jetStreamContext(t, srv) // nc is closed on evJS.Close(), js needs no cleanup
	evJS := events.NewJetstreamFromConn(natsConn)
	defer evJS.Close()

	cond := &condition.Condition{
		Kind: condition.FirmwareInstall,
		ID:   uuid.New(),
	}

	const facilityCode = "fac13"
	controllerID := registry.GetID("kvtest")

	controller := &NatsController{
		stream:        evJS,
		facilityCode:  facilityCode,
		conditionKind: cond.Kind,
		logger:        logrus.New(),
	}

	// test happy case
	publisher, err := NewNatsConditionStatusPublisher(
		"foo",
		cond.ID.String(),
		facilityCode,
		cond.Kind,
		controllerID,
		1,
		evJS,
		controller.logger,
	)
	require.Nil(t, err)
	require.NotNil(t, publisher, "publisher constructor")

	// assert to type to validate attributes
	natsCondStatusPublisher, ok := publisher.(*NatsConditionStatusPublisher)
	assert.True(t, ok)
	assert.NotNil(t, natsCondStatusPublisher)

	assert.Equal(t, controller.facilityCode, natsCondStatusPublisher.facilityCode)
	assert.Equal(t, cond.ID.String(), natsCondStatusPublisher.conditionID)
	assert.Equal(t, controller.logger, natsCondStatusPublisher.log)
	assert.Equal(t, controller.conditionKind, cond.Kind)
	assert.Equal(t, controllerID.String(), natsCondStatusPublisher.controllerID)
	assert.Equal(t, controller.logger, natsCondStatusPublisher.log)
	assert.Equal(t, uint64(0), natsCondStatusPublisher.lastRev)

	// Test re-initialized publisher will set lastRev to KV revision and subsequently publishes work
	serverID := uuid.New()
	require.NotPanics(t,
		func() {
			errP := publisher.Publish(
				context.Background(),
				serverID.String(),
				condition.Pending,
				[]byte(`{"pending...": "true"}`),
				false,
			)
			require.NoError(t, errP)
		},
		"publish 1",
	)
	require.Equal(t, uint64(1), natsCondStatusPublisher.lastRev)

	publisher2, err := NewNatsConditionStatusPublisher(
		"foo",
		cond.ID.String(),
		facilityCode,
		cond.Kind,
		controllerID,
		1,
		evJS,
		controller.logger,
	)
	natsConStatusPublisher, ok := publisher2.(*NatsConditionStatusPublisher)
	assert.True(t, ok)
	assert.NotNil(t, natsConStatusPublisher)

	require.Nil(t, err)
	require.NotNil(t, publisher2, "publisher constructor")
	require.Equal(t, uint64(1), natsConStatusPublisher.lastRev)

	require.NotPanics(t,
		func() {
			errP := publisher2.Publish(
				context.Background(),
				serverID.String(),
				condition.Active,
				[]byte(`{"some work...": "true"}`),
				false,
			)
			require.NoError(t, errP)
		},
		"publish 2",
	)
	require.Equal(t, uint64(2), natsConStatusPublisher.lastRev)
}

func TestNatsConditionStatusPublisher_Publish(t *testing.T) {
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
	controllerID := registry.GetID("kvtest")

	controller := &NatsController{
		stream:        evJS,
		facilityCode:  facilityCode,
		conditionKind: cond.Kind,
		logger:        logrus.New(),
	}

	publisher, err := NewNatsConditionStatusPublisher(
		cond.ID.String(),
		cond.ID.String(),
		facilityCode,
		cond.Kind,
		controllerID,
		1,
		evJS,
		controller.logger,
	)
	require.Nil(t, err)
	natsCondStatusPublisher, ok := publisher.(*NatsConditionStatusPublisher)
	assert.True(t, ok)
	assert.NotNil(t, natsCondStatusPublisher)

	kv, err := jsCtx.KeyValue(string(cond.Kind))
	require.NoError(t, err, "kv read handle")

	serverID := uuid.New()

	// publish pending status
	require.NotPanics(t,
		func() {
			errP := publisher.Publish(
				context.Background(),
				serverID.String(),
				condition.Pending,
				[]byte(`{"pending...": "true"}`),
				false,
			)
			require.NoError(t, errP)
		},
		"publish pending",
	)
	require.NotEqual(t, 0, natsCondStatusPublisher.lastRev, "last rev - 1")

	entry, err := kv.Get(facilityCode + "." + cond.ID.String())
	require.Nil(t, err)
	require.Equal(t, entry.Revision(), natsCondStatusPublisher.lastRev, "last rev - 2")

	sv := &condition.StatusValue{}
	err = json.Unmarshal(entry.Value(), sv)
	require.NoError(t, err, "unmarshal")

	require.Equal(t, condition.StatusValueVersion, sv.MsgVersion, "version check")
	require.Equal(t, serverID.String(), sv.Target, "sv Target")
	require.Contains(t, string(sv.Status), condition.Pending, "sv Status")

	// publish active status
	require.NotPanics(t,
		func() {
			errP := publisher.Publish(
				context.Background(),
				serverID.String(),
				condition.Active,
				[]byte(`{"active...": "true"}`),
				false,
			)
			require.NoError(t, errP)

		},
		"publish active",
	)

	entry, err = kv.Get(facilityCode + "." + cond.ID.String())
	require.Nil(t, err)
	require.Equal(t, entry.Revision(), natsCondStatusPublisher.lastRev, "last rev - 3")
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
			// nolint:revive // function param names I'd like to keep around, k thx revive
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
				if _, err := kv.Put("testFacility."+conditionID, data); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "Condition indeterminate (unreadable status)",
			setup: func(conditionID string, kv nats.KeyValue) {
				// Put unreadable data
				if _, err := kv.Put("testFacility."+conditionID, []byte("not json")); err != nil {
					t.Fatal(err)
				}
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
				if _, err := kv.Put("testFacility."+conditionID, data); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tt := range tests {
		tt.conditionID = uuid.New().String()
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.conditionID, kvStore)

			entry, err := kvStore.Get("testFacility." + tt.conditionID)
			if errors.Is(err, nats.ErrKeyNotFound) && tt.name == "Condition not started" {
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

func TestStatusValueUpdate(t *testing.T) {
	tests := []struct {
		name                string
		curSV               *condition.StatusValue
		newSV               *condition.StatusValue
		expectedSV          *condition.StatusValue
		expectedErrContains string
	}{
		{
			name: "Successful status update with different states",
			curSV: &condition.StatusValue{
				WorkerID: "worker1",
				Target:   "target1",
				TraceID:  "trace1",
				SpanID:   "span1",
				State:    string(condition.Pending),
				Status:   json.RawMessage(`{"msg":"status1"}`),
			},
			newSV: &condition.StatusValue{
				State:  string(condition.Active),
				Status: json.RawMessage(`{"msg":"status2"}`),
			},
			expectedSV: &condition.StatusValue{
				WorkerID:  "worker1",
				Target:    "target1",
				TraceID:   "trace1",
				SpanID:    "span1",
				State:     string(condition.Active),
				Status:    json.RawMessage(`{"msg":"status2"}`),
				UpdatedAt: time.Now(),
			},
		},
		{
			name: "Error returned when update on a finalized condition",
			curSV: &condition.StatusValue{
				WorkerID: "worker1",
				Target:   "target1",
				TraceID:  "trace1",
				SpanID:   "span1",
				State:    string(condition.Succeeded),
				Status:   json.RawMessage(`{"msg":"status1"}`),
			},
			newSV: &condition.StatusValue{
				State:  string(condition.Active),
				Status: json.RawMessage(`{"msg":"status2"}`),
			},
			expectedSV:          nil,
			expectedErrContains: "invalid update, condition state already finalized",
		},
		{
			name: "Error returned for invalid current Status JSON",
			curSV: &condition.StatusValue{
				WorkerID: "worker1",
				Target:   "target1",
				TraceID:  "trace1",
				SpanID:   "span1",
				State:    string(condition.Active),
				Status:   json.RawMessage(`{"msg":"status1}`),
			},
			newSV: &condition.StatusValue{
				State:  string(condition.Active),
				Status: json.RawMessage(`{"msg":"status2"}`),
			},
			expectedSV:          nil,
			expectedErrContains: "current StatusValue unmarshal error",
		},
		{
			name: "Error returned for invalid new Status JSON",
			curSV: &condition.StatusValue{
				WorkerID: "worker1",
				Target:   "target1",
				TraceID:  "trace1",
				SpanID:   "span1",
				State:    string(condition.Active),
				Status:   json.RawMessage(`{"msg":"status1"}`),
			},
			newSV: &condition.StatusValue{
				State:  string(condition.Active),
				Status: json.RawMessage(`{"msg":"status2}`),
			},
			expectedSV:          nil,
			expectedErrContains: "new StatusValue unmarshal error",
		},
		{
			name: "Empty status update does not overwrite current",
			curSV: &condition.StatusValue{
				WorkerID: "worker1",
				Target:   "target1",
				TraceID:  "trace1",
				SpanID:   "span1",
				State:    string(condition.Active),
				Status:   json.RawMessage(`{"msg":"hello"}`),
			},
			newSV: &condition.StatusValue{
				State:  string(condition.Active),
				Status: json.RawMessage(`{}`),
			},
			expectedSV: &condition.StatusValue{
				WorkerID:  "worker1",
				Target:    "target1",
				TraceID:   "trace1",
				SpanID:    "span1",
				State:     string(condition.Active),
				Status:    json.RawMessage(`{"msg":"hello"}`),
				UpdatedAt: time.Now(),
			},
		},
		{
			name: "Empty new State does not overwrite current",
			curSV: &condition.StatusValue{
				WorkerID: "worker1",
				Target:   "target1",
				TraceID:  "trace1",
				SpanID:   "span1",
				State:    string(condition.Active),
				Status:   json.RawMessage(`{"msg":"hello"}`),
			},
			newSV: &condition.StatusValue{
				Status: json.RawMessage(`{"msg": "hello 2"}`),
			},
			expectedSV: &condition.StatusValue{
				WorkerID:  "worker1",
				Target:    "target1",
				TraceID:   "trace1",
				SpanID:    "span1",
				State:     string(condition.Active),
				Status:    json.RawMessage(`{"msg":"hello 2"}`),
				UpdatedAt: time.Now(),
			},
		},
		{
			name: "No update when status is equal",
			curSV: &condition.StatusValue{
				WorkerID: "worker1",
				Target:   "target1",
				TraceID:  "trace1",
				SpanID:   "span1",
				State:    string(condition.Active),
				Status:   json.RawMessage(`{"msg":"same"}`),
			},
			newSV: &condition.StatusValue{
				State:  string(condition.Active),
				Status: json.RawMessage(`{"msg":"same"}`),
			},
			expectedSV: &condition.StatusValue{
				WorkerID:  "worker1",
				Target:    "target1",
				TraceID:   "trace1",
				SpanID:    "span1",
				State:     string(condition.Active),
				Status:    json.RawMessage(`{"msg":"same"}`),
				UpdatedAt: time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSV, err := statusValueUpdate(tt.curSV, tt.newSV)

			if tt.expectedErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSV.WorkerID, gotSV.WorkerID)
				assert.Equal(t, tt.expectedSV.Target, gotSV.Target)
				assert.Equal(t, tt.expectedSV.TraceID, gotSV.TraceID)
				assert.Equal(t, tt.expectedSV.SpanID, gotSV.SpanID)
				assert.Equal(t, tt.expectedSV.State, gotSV.State)
				assert.JSONEq(t, string(tt.expectedSV.Status), string(gotSV.Status))
				assert.WithinDuration(t, tt.expectedSV.UpdatedAt, gotSV.UpdatedAt, time.Second)
			}
		})
	}
}
