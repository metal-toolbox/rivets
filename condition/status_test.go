package condition

//nolint:all // test file

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/metal-toolbox/flasher/types"
	"github.com/nats-io/nats-server/v2/server"
	srvtest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.hollow.sh/toolbox/events"
	"go.hollow.sh/toolbox/events/pkg/kv"
	"go.hollow.sh/toolbox/events/registry"
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

func TestCheckConditionInProgress(t *testing.T) {
	srv := startJetStreamServer(t)
	defer shutdownJetStream(t, srv)
	nc, js := jetStreamContext(t, srv)
	evJS := events.NewJetstreamFromConn(nc)
	defer evJS.Close()

	testKvBucket := "testBucket"
	facilityCode := "test1"
	conditionID := uuid.New()
	workerID := "workerA"

	workerRegistryID := registry.GetID(workerID)

	key := fmt.Sprintf("%s.%s", facilityCode, conditionID.String())

	// set up the fake status KV
	cfg := &nats.KeyValueConfig{
		Bucket: string(testKvBucket),
	}

	kvWriteHandle, err := js.CreateKeyValue(cfg)
	require.NoError(t, err, "creating KV")

	// nolint:govet // these tests get no fieldalignment because
	tests := []struct {
		name                 string
		expectState          TaskState
		expectErrorContains  string
		statusValue          func() []byte
		deregisterController bool
	}{
		{
			"empty KV",
			NotStarted,
			"",
			nil,
			false,
		},
		{
			"bad status value",
			Indeterminate,
			"unable to construct a sane status for condition",
			func() []byte { return []byte(`non-status-value`) },
			false,
		},
		{
			"finished status value",
			Complete,
			"",
			func() []byte {
				sv := &types.StatusValue{State: string(Failed)}
				return sv.MustBytes()
			},
			false,
		},
		{
			"bogus worker ID",
			Indeterminate,
			"bad worker ID",
			func() []byte {
				sv := &StatusValue{
					State:    string(Pending),
					WorkerID: "some junk id",
				}

				return sv.MustBytes()
			},
			false,
		},
		{
			"register test worker and lookup value",
			InProgress,
			"",
			func() []byte {
				// init registry and register controller
				err = registry.InitializeRegistryWithOptions(evJS, kv.WithReplicas(1))
				require.NoError(t, err, "initialize controller registry")

				err = registry.RegisterController(workerRegistryID)
				require.NoError(t, err, "register test controller")

				sv := &StatusValue{
					State:    string(Pending),
					WorkerID: workerRegistryID.String(),
				}
				return sv.MustBytes()
			},
			false,
		},
		{
			"no live workers - condition Orphaned",
			Orphaned,
			"",
			func() []byte {
				// deregister controller
				err = registry.DeregisterController(workerRegistryID)
				require.NoError(t, err, "deregister controller")

				sv := &StatusValue{
					State:    string(Pending),
					WorkerID: workerRegistryID.String(),
				}
				return sv.MustBytes()
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.statusValue != nil {
				_, err = kvWriteHandle.Put(key, tt.statusValue())
				if err != nil {
					t.Fatal(err)
				}
			}

			gotState, err := CheckConditionInProgress(conditionID.String(), facilityCode, testKvBucket, js)
			if tt.expectErrorContains != "" {
				assert.ErrorContains(t, err, tt.expectErrorContains)
				return
			}

			assert.Nil(t, err)

			// make sure that the Delete fired to clear the KV and make things clean for a new worker
			if tt.deregisterController {
				_, err = kvWriteHandle.Get(conditionID.String())
				require.ErrorIs(t, err, nats.ErrKeyNotFound)
			}

			require.Equal(t, tt.expectState, gotState, tt.name)
		})
	}
}
