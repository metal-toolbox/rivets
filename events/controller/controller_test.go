package controller

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/goleak"

	"github.com/metal-toolbox/rivets/condition"
	"github.com/metal-toolbox/rivets/events"
	"github.com/metal-toolbox/rivets/events/registry"
)

func TestNewNatsControllerWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		option   Option
		validate func(*testing.T, *NatsController)
	}{
		{
			name:   "WithLogger",
			option: WithLogger(logrus.New()),
			validate: func(t *testing.T, nc *NatsController) {
				assert.NotNil(t, nc.logger, "logger should not be nil")
			},
		},
		{
			name:   "WithConcurrency",
			option: WithConcurrency(10),
			validate: func(t *testing.T, nc *NatsController) {
				assert.Equal(t, 10, nc.concurrency, "concurrency should match the expected value")
			},
		},
		{
			name:   "WithKVReplicas",
			option: WithKVReplicas(0),
			validate: func(t *testing.T, nc *NatsController) {
				assert.Equal(t, 0, nc.natsConfig.KVReplicationFactor, "kvReplicas should match the expected value")
			},
		},
		{
			name:   "WithHandlerTimeout",
			option: WithHandlerTimeout(15 * time.Minute),
			validate: func(t *testing.T, nc *NatsController) {
				assert.Equal(t, 15*time.Minute, nc.handlerTimeout, "handlerTimeout should match the expected value")
			},
		},
		{
			name:   "WithConnectionTimeout",
			option: WithConnectionTimeout(45 * time.Second),
			validate: func(t *testing.T, nc *NatsController) {
				assert.Equal(t, 45*time.Second, nc.natsConfig.ConnectTimeout, "connectionTimeout should match the expected value")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nc := &NatsController{}
			// set option
			tc.option(nc)
			// verify
			tc.validate(t, nc)
		})
	}
}

func TestNatsControllerDefaultParameters(t *testing.T) {
	c := NewNatsController(
		"app",
		"facility",
		"firmwareInstall",
		"nats://localhost:4222",
		"/etc/nats/creds.file",
		condition.FirmwareInstall,
	)

	expectedHostname, _ := os.Hostname()

	assert.Equal(t, expectedHostname, c.hostname, "hostname should match the expected default value")
	assert.Equal(t, "facility", c.facilityCode, "facilityCode should match the expected default value")
	assert.Equal(t, condition.FirmwareInstall, c.conditionKind, "conditionKind should match the expected default value")
	assert.Equal(t, connectionTimeout, c.connectionTimeout, "connectionTimeout should match the expected default value")
	assert.Equal(t, handlerTimeout, c.handlerTimeout, "handlerTimeout should match the expected default value")
	assert.Equal(t, pullEventInterval, c.pullEventInterval, "pullEventInterval should match the expected default value")
	assert.Equal(t, pullEventTimeout, c.pullEventTimeout, "pullEventTimeout should match the expected default value")
	assert.Equal(t, concurrency, c.concurrency, "concurrency should match the expected default value")
	assert.Equal(t, kvReplicationFactor, c.natsConfig.KVReplicationFactor, "kv replicas should match the expected default value")
}

func TestProcessCondition(t *testing.T) {
	tests := []struct {
		name         string
		conditionID  uuid.UUID
		state        ConditionState
		expectMethod string
		handlerErr   error
	}{
		{
			name:         "Condition in progress",
			conditionID:  uuid.New(),
			state:        inProgress,
			expectMethod: "inProgress",
			handlerErr:   nil,
		},
		{
			name:         "Condition complete",
			conditionID:  uuid.New(),
			state:        complete,
			expectMethod: "complete",
			handlerErr:   nil,
		},
		{
			name:         "Condition orphaned",
			conditionID:  uuid.New(),
			state:        orphaned,
			expectMethod: "inProgress",
			handlerErr:   nil,
		},
		{
			name:         "Condition not started",
			conditionID:  uuid.New(),
			state:        notStarted,
			expectMethod: "inProgress",
			handlerErr:   nil,
		},
		{
			name:         "Condition state indeterminate",
			conditionID:  uuid.New(),
			state:        indeterminate,
			expectMethod: "nak",
			handlerErr:   nil,
		},
		{
			name:         "Condition handler ErrRetryHandler is a nak",
			conditionID:  uuid.New(),
			state:        notStarted,
			expectMethod: "inProgress",
			handlerErr:   errors.Wrap(ErrRetryHandler, "cosmic rays"),
		},
		{
			name:         "Condition handler other errors is an ack",
			conditionID:  uuid.New(),
			state:        notStarted,
			expectMethod: "inProgress",
			handlerErr:   errors.New("some other error occurred"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := &condition.Condition{ID: tt.conditionID}

			cStatusQueryor := NewMockConditionStatusQueryor(t)
			// expect method to be invoked for each condition status query
			cStatusQueryor.On("ConditionState", tt.conditionID.String()).Return(tt.state)

			eStatusAcknowledger := NewMockeventStatusAcknowleger(t)
			eStatusAcknowledger.On(tt.expectMethod).Return()

			handler := NewMockConditionHandler(t)
			// expect handler, completions for orphaned and notStarted conditions
			if tt.state == orphaned || tt.state == notStarted {
				if tt.handlerErr == nil {
					// no handler errors
					handler.On("Handle", mock.Anything, mock.Anything, mock.Anything).Return(nil)
					eStatusAcknowledger.On("complete").Return()
				} else {
					// handler errors
					if errors.Is(tt.handlerErr, ErrRetryHandler) {
						handler.On("Handle", mock.Anything, mock.Anything, mock.Anything).Return(tt.handlerErr)
						eStatusAcknowledger.On("nak").Return()
					} else {
						handler.On("Handle", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("some other error"))
						eStatusAcknowledger.On("complete").Return()
					}
				}
			}

			n := &NatsController{
				logger:                  logrus.New(),
				controllerID:            registry.GetID("mock"),
				conditionHandlerFactory: func() ConditionHandler { return handler },
			}

			n.processCondition(context.Background(), cond, eStatusAcknowledger, cStatusQueryor, nil)

			eStatusAcknowledger.AssertCalled(t, tt.expectMethod)
			cStatusQueryor.AssertExpectations(t)
		})
	}
}

func TestConditionFromEvent(t *testing.T) {
	conditionID := uuid.New()

	tests := []struct {
		name    string
		data    func() []byte
		want    *condition.Condition
		wantErr bool
	}{
		{
			name:    "data field empty",
			data:    func() []byte { return nil },
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unable to deserialize condition",
			data:    func() []byte { return []byte(`invalid json`) },
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid condition",
			data: func() []byte {
				condition := &condition.Condition{
					Version:    "1",
					ID:         conditionID,
					Kind:       "test-kind",
					Parameters: json.RawMessage(`{"foo":"bar"}`),
				}
				data, _ := json.Marshal(condition)
				return data
			},
			wantErr: false,
			want: func() *condition.Condition {
				condition := &condition.Condition{
					Version:    "1",
					ID:         uuid.New(),
					Kind:       "test-kind",
					Parameters: json.RawMessage(`{"foo":"bar"}`),
				}
				return condition
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMsg := events.NewMockMessage(t)
			mockMsg.On("Data").Return(tt.data())

			got, err := conditionFromEvent(mockMsg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.Version, got.Version)
				assert.Equal(t, conditionID.String(), got.ID.String())
				assert.Equal(t, tt.want.Kind, got.Kind)
				assert.Equal(t, tt.want.Parameters, got.Parameters)
			}

			mockMsg.AssertExpectations(t)
		})
	}
}

func TestRunConditionHandlerWithMonitor(t *testing.T) {
	// test for no leaked go routines
	defer goleak.VerifyNone(t)

	// test monitor calls ackInprogress
	ctx := context.Background()
	cond := &condition.Condition{Kind: condition.FirmwareInstall}
	publisher := NewMockConditionStatusPublisher(t)

	message := events.NewMockMessage(t)
	message.On("InProgress").Return(nil)

	handler := NewMockConditionHandler(t)
	handler.On("Handle", mock.Anything, cond, publisher).Times(1).
		//  sleep for 100ms
		Run(func(_ mock.Arguments) { time.Sleep(100 * time.Millisecond) }).
		Return(nil)

	n := &NatsController{
		logger: logrus.New(),
		conditionHandlerFactory: func() ConditionHandler {
			return handler
		},
	}

	gotErr := n.runConditionHandlerWithMonitor(
		ctx,
		cond,
		n.newNatsEventStatusAcknowleger(message),
		publisher,
		10*time.Millisecond, // ack every 10ms
	)

	assert.Nil(t, gotErr)

	message.AssertExpectations(t)
	handler.AssertExpectations(t)
	assert.GreaterOrEqual(t, len(message.Calls), 9, "expect multiple ackInprogress")
	assert.Equal(t, 1, len(handler.Calls), "expect handler to be called once")
}
