package controller

import (
	"context"

	"github.com/google/uuid"
	co "github.com/metal-toolbox/conditionorc/pkg/api/v1/client"
	"github.com/metal-toolbox/conditionorc/pkg/api/v1/types"
	"github.com/metal-toolbox/rivets/condition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// HttpTaskRepository implements the ConditionTaskRepository to Publish and Query Task information from NATS over http.
type HttpTaskRepository struct {
	logger        *logrus.Logger
	coClient      *co.Client
	conditionKind condition.Kind
	conditionID   uuid.UUID
	serverID      uuid.UUID
}

func NewHttpTaskRepository(
	serverID,
	conditionID uuid.UUID,
	conditionKind condition.Kind,
	coClient *co.Client,
	logger *logrus.Logger,
) ConditionTaskRepository {
	return &HttpTaskRepository{
		coClient:      coClient,
		conditionID:   conditionID,
		conditionKind: conditionKind,
		serverID:      serverID,
		logger:        logger,
	}
}

func (h *HttpTaskRepository) Publish(ctx context.Context, task *condition.Task[any, any], tsUpdateOnly bool) error {
	_, span := otel.Tracer(pkgNameNatsController).Start(
		ctx,
		"controller.task_http.Publish",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	resp, err := h.coClient.ConditionTaskPublish(ctx, h.conditionKind, h.serverID, h.conditionID, task, tsUpdateOnly)
	if err != nil {
		h.logger.WithError(err).Error("Task update error")
		return errors.Wrap(errTaskPublish, err.Error())
	}

	h.logger.WithFields(
		logrus.Fields{
			"status": resp.StatusCode,
		},
	).Trace("Task update published successfully")

	return nil
}

func (h *HttpTaskRepository) Query(ctx context.Context) (*condition.Task[any, any], error) {
	_, span := otel.Tracer(pkgNameNatsController).Start(
		ctx,
		"controller.task_http.Query",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	resp, err := h.coClient.ConditionTaskQuery(ctx, h.conditionKind, h.serverID)
	if err != nil {
		h.logger.WithError(err).Error("Task query error")
		return nil, errors.Wrap(errTaskQuery, err.Error())
	}

	h.logger.WithFields(
		logrus.Fields{
			"status": resp.StatusCode,
		},
	).Trace("Task queried successfully")

	return h.taskFromResponse(resp)
}

func (h *HttpTaskRepository) taskFromResponse(resp *types.ServerResponse) (*condition.Task[any, any], error) {
	errEmptyRecords := errors.New("empty records object in response")
	errNoTask := errors.New("no Task object in response")

	if resp.Records == nil {
		return nil, errEmptyRecords
	}

	if len(resp.Records.Tasks) > 0 {
		return nil, errNoTask
	}

	return resp.Records.Tasks[0], nil
}
