package controller

import (
	"context"
	"time"

	"github.com/google/uuid"
	co "github.com/metal-toolbox/conditionorc/pkg/api/v1/client"
	"github.com/metal-toolbox/rivets/condition"
	"github.com/metal-toolbox/rivets/events"
	"github.com/metal-toolbox/rivets/events/registry"
	"github.com/sirupsen/logrus"
)

type publisherKind string

const (
	natsPublisher publisherKind = "nats"
	httpPublisher publisherKind = "http"
)

// The Publisher interface wraps the Task and StatusValue publishers into one.
type Publisher interface {
	Publish(ctx context.Context, task *condition.Task[any, any], tsUpdate bool) error
}

type PublisherHttp struct {
	logger               *logrus.Logger
	statusValuePublisher *HttpConditionStatusPublisher
	taskRepository       *HttpTaskRepository
}

func NewHttpPublisher(serverID,
	conditionID uuid.UUID,
	conditionKind condition.Kind,
	controllerID registry.ControllerID,
	coClient *co.Client,
	logger *logrus.Logger) Publisher {

	p := &PublisherHttp{logger: logger}
	httpStatusValuePublisher := NewHttpConditionStatusPublisher(
		serverID,
		conditionID,
		conditionKind,
		controllerID,
		coClient,
		logger,
	)

	p.statusValuePublisher = httpStatusValuePublisher.(*HttpConditionStatusPublisher)

	httpTaskRepository := NewHttpTaskRepository(
		serverID,
		conditionID,
		conditionKind,
		coClient,
		logger,
	)

	p.taskRepository = httpTaskRepository.(*HttpTaskRepository)

	return p
}

func (p *PublisherHttp) Publish(ctx context.Context, task *condition.Task[any, any], tsUpdate bool) error {
	err := p.statusValuePublisher.Publish(ctx, task.Server.ID, task.State, task.Status.MustMarshal(), tsUpdate)
	if err != nil {
		p.logger.WithError(err).Error("Status Value publish error")
		return err
	}

	if tsUpdate {
		task.UpdatedAt = time.Now()
	}

	err = p.taskRepository.Publish(ctx, task, tsUpdate)
	if err != nil {
		p.logger.WithError(err).Error("Task publish error")
		return err
	}

	return nil
}

type PublisherNats struct {
	logger               *logrus.Logger
	statusValuePublisher *NatsConditionStatusPublisher
	taskRepository       *NatsConditionTaskRepository
}

func NewNatsPublisher(
	appName,
	conditionID,
	serverID,
	facilityCode string,
	conditionKind condition.Kind,
	controllerID registry.ControllerID,
	kvReplicas int,
	stream *events.NatsJetstream,
	logger *logrus.Logger,
) (Publisher, error) {

	p := &PublisherNats{logger: logger}
	svPublisher, err := NewNatsConditionStatusPublisher(
		appName,
		conditionID,
		facilityCode,
		conditionKind,
		controllerID,
		kvReplicas,
		stream,
		logger,
	)
	if err != nil {
		return nil, err
	}

	p.statusValuePublisher = svPublisher.(*NatsConditionStatusPublisher)

	taskRepository, err := NewNatsConditionTaskRepository(
		appName,
		conditionID,
		serverID,
		facilityCode,
		conditionKind,
		controllerID,
		kvReplicas,
		stream,
		logger,
	)
	if err != nil {
		return nil, err
	}

	p.taskRepository = taskRepository.(*NatsConditionTaskRepository)

	return p, nil
}

func (p *PublisherNats) Publish(ctx context.Context, task *condition.Task[any, any], tsUpdate bool) error {
	err := p.statusValuePublisher.Publish(ctx, task.Server.ID, task.State, task.Status.MustMarshal(), tsUpdate)
	if err != nil {
		p.logger.WithError(err).Error("Status Value publish error")
		return err
	}

	err = p.taskRepository.Publish(ctx, task, tsUpdate)
	if err != nil {
		p.logger.WithError(err).Error("Task publish error")
		return err
	}

	return nil
}
