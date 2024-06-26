package controller

import (
	"context"
	"encoding/json"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/metal-toolbox/rivets/condition"
	"github.com/metal-toolbox/rivets/events"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	pkgNameNatsController = "events/natscontroller"
	// Default connection timeout
	connectionTimeout = 1 * time.Minute
	// Default timeout for the condition handler, after which its canceled
	handlerTimeout = 180 * time.Minute
	// Default pull event interval
	pullEventInterval = 1 * time.Second
	// Default timeout for pulling a new event from the JS
	pullEventTimeout = 5 * time.Second
	// Default concurrency
	concurrency = 2
	// controller check in interval
	checkinInterval = 30 * time.Second
	// periodically publish an updated status
	statusInterval = checkinInterval
	// condition status considered stale after this period
	StatusStaleThreshold = condition.StaleThreshold
	//  controller considered dead after this period
	LivenessStaleThreshold = condition.StaleThreshold
	// default number of KV replicas for created NATS buckets
	kvReplicationFactor = 3
)

var (
	// This error when returned by the callback indicates it needs to be retried
	ErrRetryHandler = errors.New("retry callback")
)

type NatsController struct {
	stream            events.Stream
	syncWG            *sync.WaitGroup
	logger            *logrus.Logger
	natsConfig        events.NatsOptions
	hostname          string
	facilityCode      string
	conditionKind     condition.Kind
	pullEventInterval time.Duration
	pullEventTimeout  time.Duration
	handlerTimeout    time.Duration
	connectionTimeout time.Duration
	checkinInterval   time.Duration
	// Factory method returns a condition event handler
	// set by the caller when calling ListenEvents()
	conditionHandlerFactory ConditionHandlerFactory
	// Queryor provides methods to query the Condition State, StatusValue's
	conditionStatusQueryor ConditionStatusQueryor
	liveness               LivenessCheckin
	concurrency            int
	dispatched             int32
}

// Option sets parameters on the NatsController
type Option func(*NatsController)

func NewNatsController(
	appName,
	facilityCode,
	subjectSuffix,
	natsURL,
	credsFile string,
	conditionKind condition.Kind,
	options ...Option) *NatsController {
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	hostname, _ := os.Hostname()

	queueCfg := queueConfig(appName, facilityCode, subjectSuffix, natsURL, credsFile)
	nwp := &NatsController{
		hostname:          hostname,
		facilityCode:      facilityCode,
		conditionKind:     conditionKind,
		stream:            nil,
		logger:            logger,
		syncWG:            &sync.WaitGroup{},
		connectionTimeout: connectionTimeout,
		handlerTimeout:    handlerTimeout,
		pullEventInterval: pullEventInterval,
		pullEventTimeout:  pullEventTimeout,
		checkinInterval:   checkinInterval,
		concurrency:       concurrency,
		natsConfig:        queueCfg,
	}

	for _, opt := range options {
		opt(nwp)
	}

	return nwp
}

func WithLogger(logger *logrus.Logger) Option {
	return func(n *NatsController) {
		n.logger = logger
	}
}

func WithConcurrency(c int) Option {
	return func(n *NatsController) {
		n.concurrency = c
	}
}

// Set the number of replicates to keep for the
//
// !! In a non-clustered NATS environment, set this value to 0.
func WithKVReplicas(c int) Option {
	return func(n *NatsController) {
		n.natsConfig.KVReplicationFactor = c
	}
}

func WithHandlerTimeout(t time.Duration) Option {
	return func(n *NatsController) {
		n.handlerTimeout = t
	}
}

func WithConnectionTimeout(t time.Duration) Option {
	return func(n *NatsController) {
		n.natsConfig.ConnectTimeout = t
	}
}

func (n *NatsController) ID() string {
	return n.liveness.ControllerID().String()
}

func (n *NatsController) FacilityCode() string {
	return n.facilityCode
}

// Connect to NATS Jetstream and register as a controller.
func (n *NatsController) Connect(ctx context.Context) error {
	ctx, span := otel.Tracer(pkgNameNatsController).Start(
		ctx,
		"Connect",
	)
	defer span.End()

	startTS := time.Now()

	errInit := errors.New("nats broker init error")
	stream, err := events.NewNatsBroker(n.natsConfig)
	if err != nil {
		return errors.Wrap(errInit, err.Error())
	}

	errOpen := errors.New("event stream connection error")
	if err := stream.Open(); err != nil {
		return errors.Wrap(errOpen, err.Error())
	}

	n.stream = stream

	// returned channel ignored, since this is a Pull based subscription.
	errStreamSub := errors.New("event stream subscription error")
	if _, err := n.stream.Subscribe(ctx); err != nil {
		return errors.Wrap(errStreamSub, err.Error())
	}

	// initialize NATS liveness checkin
	n.liveness = NewNatsLiveness(n.natsConfig, n.stream, n.logger, n.hostname, checkinInterval)
	n.liveness.StartLivenessCheckin(ctx)

	n.logger.WithFields(
		logrus.Fields{
			"hostname":      n.hostname,
			"facility":      n.facilityCode,
			"replica-count": n.natsConfig.KVReplicationFactor,
			"concurrency":   n.concurrency,
			"connect-time":  time.Since(startTS).String(),
		},
	).Info("connected to event stream as controller")
	registerNATSConnectTimeMetric(startTS)

	return nil
}

// TaskHandler is passed in by the caller to be invoked when the expected event is received
//
// Note: the TaskHandler interface is to be deprecated by the TaskHandler interface (as seen in controller_http.go)
// where the Condition is converted to a condition.Task object before being passed into
// the Handler.
// The TaskHandler interface is to replace the TaskHandler interface
type TaskHandler interface {
	HandleTask(ctx context.Context, task *condition.Task[any, any], statusPublisher Publisher) error
}

type ConditionHandlerFactory func() TaskHandler

// Handle events accepts a callback function to run when an event is fetched from the NATS JS.
//
// - The caller is expected to enclose any work and error handling for the work within the callback function.
// - When the callback function returns no error, the event is marked as completed.
// - When the callback function returns an ErrRetryHandler error, the corresponding event is placed back on the queue to be retried.
func (n *NatsController) ListenEvents(ctx context.Context, chf ConditionHandlerFactory) error {
	ctx, span := otel.Tracer(pkgNameNatsController).Start(
		ctx,
		"ListenEvents",
	)
	defer span.End()

	errListenEvents := errors.New("listen events error")
	if chf == nil {
		return errors.Wrap(errListenEvents, "expected valid ConditionHandlerFactor, got nil")
	}

	if n.stream == nil {
		return errors.Wrap(errListenEvents, "connection not initialized, invoke Connect() before calling this method")
	}

	n.conditionHandlerFactory = chf

	pullTicker := time.NewTicker(n.pullEventInterval)
	defer pullTicker.Stop()

Loop:
	for {
		select {
		case <-pullTicker.C:
			if n.concurrencyLimit() {
				continue
			}

			if err := n.processEvents(ctx); err != nil {
				return errors.Wrap(errListenEvents, err.Error())
			}
		case <-ctx.Done():
			if n.dispatched > 0 {
				continue
			}

			break Loop
		}
	}

	return nil
}

// process event into a condition
func (n *NatsController) processEvents(ctx context.Context) error {
	pullCtx, cancel := context.WithTimeout(ctx, n.pullEventTimeout)
	defer cancel()

	errProcessEvent := errors.New("process event error")
	msgs, err := n.stream.PullMsg(pullCtx, 1)
	switch {
	case err == nil:
	case errors.Is(err, nats.ErrTimeout):
		n.logger.WithFields(
			logrus.Fields{"info": err.Error()},
		).Trace("no new events")
		return nil

	default:
		n.logger.WithFields(
			logrus.Fields{"err": err.Error()},
		).Warn("retrieving new messages")

		metricsNATSError("pull-msg")
		return errors.Wrap(errProcessEvent, err.Error())
	}

	for _, msg := range msgs {
		if n.concurrencyLimit() {
			return nil
		}

		// event status setter to keep the JS updated on our progress
		eventAcknowleger := n.newNatsEventStatusAcknowleger(msg)

		if ctx.Err() != nil {
			eventAcknowleger.nak()

			return errors.Wrap(errProcessEvent, ctx.Err().Error())
		}

		// setup the handle to query condition status
		conditionStatusQueryor, err := n.NewNatsConditionStatusQueryor()
		if err != nil {
			eventAcknowleger.nak()

			return errors.Wrap(errProcessEvent, err.Error())
		}

		// spawn msg process handler
		n.syncWG.Add(1)
		go n.processConditionFromEvent(ctx, msg, eventAcknowleger, conditionStatusQueryor)
	}

	return nil
}

func (n *NatsController) processConditionFromEvent(ctx context.Context, msg events.Message, eventAcknowleger eventStatusAcknowleger, conditionStatusQueryor ConditionStatusQueryor) {
	defer n.syncWG.Done()
	ctx, _ = otel.Tracer(pkgNameNatsController).Start(
		ctx,
		"processConditionFromEvent",
	)

	atomic.AddInt32(&n.dispatched, 1)
	defer atomic.AddInt32(&n.dispatched, -1)

	cond, err := conditionFromEvent(msg)
	if err != nil {
		n.logger.WithError(err).WithField(
			"subject", msg.Subject()).Warn("unable to retrieve condition from message")

		metricsEventsCounter(false, "ack")
		eventAcknowleger.complete()

		return
	}

	// extract parent trace context from the event if any.
	ctx = msg.ExtractOtelTraceContext(ctx)
	n.conditionStatusQueryor = conditionStatusQueryor
	n.processCondition(ctx, cond, eventAcknowleger)
}

func conditionFromEvent(e events.Message) (*condition.Condition, error) {
	errConditionDeserialize := errors.New("unable to deserialize condition")
	data := e.Data()
	if data == nil {
		return nil, errors.Wrap(errConditionDeserialize, "data field empty")
	}

	cond := &condition.Condition{}
	if err := json.Unmarshal(data, cond); err != nil {
		return nil, errors.Wrap(errConditionDeserialize, err.Error())
	}

	return cond, nil
}

// process condition
func (n *NatsController) processCondition(
	ctx context.Context,
	cond *condition.Condition,
	eventAcknowleger eventStatusAcknowleger, // the NATS JS event status ack interface
) {
	ctx, span := otel.Tracer(pkgNameNatsController).Start(
		ctx,
		"processCondition",
	)
	defer span.End()

	// measure runtime
	startTS := time.Now()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "conditionKind",
			Value: attribute.StringValue(cond.ID.String()),
		},
	)

	// check and see if the condition is or has-been handled by another controller
	currentState := n.conditionStatusQueryor.ConditionState(cond.ID.String())
	switch currentState {
	case inProgress:
		n.logger.WithField("conditionID", cond.ID.String()).Info("condition is already in progress")
		eventAcknowleger.inProgress()
		spanEvent(span, cond, n.ID(), "ackInProgress")

		return

	case complete:
		n.logger.WithField("conditionID", cond.ID.String()).Info("condition is complete")
		eventAcknowleger.complete()
		spanEvent(span, cond, n.ID(), "ackComplete")

		return

	case orphaned:
		n.logger.WithField("conditionID", cond.ID.String()).Warn("restarting this condition")
		eventAcknowleger.inProgress()
		spanEvent(span, cond, n.ID(), "restarting condition")

	// we need to restart this event
	case notStarted:
		n.logger.WithField("conditionID", cond.ID.String()).Info("starting new condition")
		eventAcknowleger.inProgress()
		spanEvent(span, cond, n.ID(), "start new condition")

	// break out here, this is a new event
	case indeterminate:
		n.logger.WithField("conditionID", cond.ID.String()).Warn("unable to determine state of this condition")
		// send it back to NATS to try again
		eventAcknowleger.nak()
		spanEvent(span, cond, n.ID(), "sent nack, indeterminate state")

		return
	}

	handlerCtx, cancel := context.WithTimeout(ctx, n.handlerTimeout)
	defer cancel()

	errHandler := n.runConditionHandlerWithMonitor(handlerCtx, cond, eventAcknowleger, statusInterval)
	if errHandler != nil {
		registerConditionRuntimeMetric(startTS, string(condition.Failed))

		// handler indicates this must be retried
		if errors.Is(errHandler, ErrRetryHandler) {
			n.logger.WithError(errHandler).Warn("condition handler returned retry error")

			// TODO: write this condition back onto the Jetstream so its retried
		}

		// TODO: consider publishing a failed status value here
		// other errors are logged
		n.logger.WithError(errHandler).Error("condition handler returned error")
		spanEvent(
			span,
			cond,
			n.ID(),
			"condition handler returned error: "+errHandler.Error(),
		)
	}

	registerConditionRuntimeMetric(startTS, string(condition.Succeeded))

	spanEvent(
		span,
		cond,
		n.ID(),
		"condition complete",
	)
}

func (n *NatsController) runConditionHandlerWithMonitor(ctx context.Context, cond *condition.Condition, eventStatusSet eventStatusAcknowleger, statusInterval time.Duration) (err error) {
	ctx, span := otel.Tracer(pkgNameNatsController).Start(
		ctx,
		"runConditionHandlerWithMonitor",
	)
	defer span.End()

	// msg nak helper
	nakFail := func(info string, failErr error) error {
		n.logger.WithError(failErr).WithFields(logrus.Fields{
			"condition.id": cond.ID.String(),
		}).Warn(info)

		// send this message back on the bus to be redelivered, or atleast attempt to.
		eventStatusSet.nak()
		metricsEventsCounter(true, "nack")

		return err
	}

	publisher, err := NewNatsPublisher(
		n.natsConfig.AppName,
		cond.ID.String(),
		cond.Target.String(),
		n.facilityCode,
		n.conditionKind,
		n.liveness.ControllerID(),
		n.natsConfig.KVReplicationFactor,
		n.stream.(*events.NatsJetstream),
		n.logger,
	)
	if err != nil {
		return nakFail("failed to initialize publisher", err)
	}

	// create first record of condition in controller KV
	//
	// failure to publish the first status KV record is fatal
	task := condition.NewTaskFromCondition(cond)
	task.Status = condition.NewTaskStatusRecord("In process by controller: " + n.hostname)

	if err = publisher.Publish(ctx, task, false); err != nil {
		info := "error publishing first KV status record, condition aborted"
		return nakFail(info, err)
	}

	publish := func(state condition.State, status string) {

		// append to existing status record, unless it was overwritten by the controller somehow
		task.Status.Append(status)
		task.State = state

		// publish failed state, status
		if err := publisher.Publish(
			ctx,
			task,
			false,
		); err != nil {
			n.logger.WithError(err).Error("failed to publish final status")
		}
	}

	// mark message as complete as the status KV record is in place
	eventStatusSet.complete()

	// doneCh indicates the handler run completed
	doneCh := make(chan bool)

	// monitor updates TS on status until the task handler returns.
	monitor := func() {
		ticker := time.NewTicker(statusInterval)
		defer ticker.Stop()

		// periodically update the LastUpdate TS in status KV,
		/// which keeps the Orchestrator from reconciling this condition.
	Loop:
		for {
			select {
			case <-ticker.C:
				// only timestamp update
				publisher.Publish(ctx, task, true)
			case <-doneCh:
				break Loop
			}
		}
	}

	go monitor()
	defer close(doneCh)

	// panic handler
	defer func() {
		if rec := recover(); rec != nil {
			// overwrite returned err - declared in func signature
			err = errors.New("Panic occurred while running Condition handler")
			n.logger.Printf("!!panic %s: %s", rec, debug.Stack())
			n.logger.Error(err)

			publish(condition.Failed, "Fatal error running Condition, check logs for details")
		}
	}() // nolint:errcheck // nope

	if err := n.conditionHandlerFactory().HandleTask(ctx, task, publisher); err != nil {
		publish(condition.Failed, "failed with error: "+err.Error())
		return err
	}

	// TODO:
	// If the handler has returned and not updated the Task.State, StatusValue.State
	// into a final state, then set those fields to failed.
	n.logger.Info("Controller completed task")

	return nil
}

func (n *NatsController) concurrencyLimit() bool {
	return n.dispatched >= int32(n.concurrency)
}
