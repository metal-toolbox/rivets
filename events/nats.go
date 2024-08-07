//nolint:wsl // useless
package events

import (
	"context"
	"log"
	"reflect"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var (
	// ErrNatsConfig is returned when the conf
	ErrNatsConfig = errors.New("error in NATs Jetstream configuration")

	// ErrNatsConn is returned when an error occurs in connecting to NATS.
	ErrNatsConn = errors.New("error opening nats connection")

	// ErrNatsJetstream is returned when an error occurs in setting up the NATS Jetstream context.
	ErrNatsJetstream = errors.New("error in NATS Jetstream")

	// ErrNatsJetstreamAddStream os returned when an attempt to add a NATS Jetstream fails.
	ErrNatsJetstreamAddStream = errors.New("error adding stream to NATS Jetstream")

	// ErrNatsJetstreamAddConsumer is returned when theres an error adding a consumer to the NATS Jetstream.
	ErrNatsJetstreamAddConsumer = errors.New("error adding consumer on NATS Jetstream")

	// ErrNatsJetstreamUpdateConsumer is returned when theres an error updating a consumer configuration on the NATS Jetstream.
	ErrNatsJetstreamUpdateConsumer = errors.New("error updating consumer configuration on NATS Jetstream")

	// ErrNatsMsgPull is returned when theres and error pulling a message from a NATS Jetstream.
	ErrNatsMsgPull = errors.New("error fetching message from NATS Jetstream")

	// ErrSubscription is returned when an error in the consumer subscription occurs.
	ErrSubscription = errors.New("error subscribing to stream")
)

const (
	consumerMaxDeliver    = 5
	consumerAckPolicy     = nats.AckExplicitPolicy
	conditionJetstreamTTL = 3 * time.Hour
)

// NatsJetstream wraps the NATs JetStream connector to implement the Stream interface.
type NatsJetstream struct {
	jsctx         nats.JetStreamContext
	conn          *nats.Conn
	parameters    *NatsOptions
	subscriptions []*nats.Subscription
	subscriberCh  MsgCh
}

// Add some conversions for functions/APIs that expect NATS primitive types. This allows consumers of
// NatsJetsream to convert easily to the types they need, without exporting the members or coercing
// and direct clients/holders of NatsJetstream to do this conversion.
// AsNatsConnection exposes the otherwise private NATS connection pointer
func AsNatsConnection(n *NatsJetstream) *nats.Conn {
	return n.conn
}

// AsNatsJetstreamContext exposes the otherwise private NATS JetStreamContext
func AsNatsJetStreamContext(n *NatsJetstream) nats.JetStreamContext {
	return n.jsctx
}

// NewNatsBroker validates the given stream broker parameters and returns a stream broker implementation.
func NewNatsBroker(params StreamParameters) (*NatsJetstream, error) {
	parameters, valid := params.(NatsOptions)
	if !valid {
		return nil, errors.Wrap(
			ErrNatsConfig,
			"expected parameters of type NatsOptions{}, got: "+reflect.TypeOf(parameters).String(),
		)
	}

	if err := parameters.validate(); err != nil {
		return nil, err
	}

	return &NatsJetstream{parameters: &parameters}, nil
}

// NewJetstreamFromConn takes an already established NATS connection pointer and returns a NatsJetstream pointer
func NewJetstreamFromConn(c *nats.Conn) *NatsJetstream {
	// JetStream() only returns an error if you call it with incompatible options. It is *not*
	// a guarantee that c has JetStream enabled.
	js, _ := c.JetStream()
	return &NatsJetstream{
		conn:  c,
		jsctx: js,
	}
}

// Open connects to the NATS Jetstream.
func (n *NatsJetstream) Open() error {
	if n.conn != nil {
		return errors.Wrap(ErrNatsConn, "NATS connection is already established")
	}

	if n.parameters == nil {
		return errors.Wrap(ErrNatsConfig, "NATS config parameters not defined")
	}

	opts := []nats.Option{
		nats.Name(n.parameters.AppName),
		nats.Timeout(n.parameters.ConnectTimeout),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(reconnectWait),
		nats.ReconnectJitter(reconnectJitter, reconnectJitter),
	}

	if n.parameters.StreamUser != "" {
		opts = append(opts, nats.UserInfo(n.parameters.StreamUser, n.parameters.StreamPass))
	} else {
		opts = append(opts, nats.UserCredentials(n.parameters.CredsFile))
	}

	conn, err := nats.Connect(n.parameters.URL, opts...)
	if err != nil {
		return errors.Wrap(ErrNatsConn, err.Error())
	}

	n.conn = conn

	// setup the channel for subscribers to read messages from.
	n.subscriberCh = make(MsgCh)

	// setup Jetstream and consumer
	return n.setup()
}

func (n *NatsJetstream) setup() error {
	js, err := n.conn.JetStream()
	if err != nil {
		return errors.Wrap(ErrNatsJetstream, err.Error())
	}

	n.jsctx = js

	if n.parameters.Stream != nil {
		if err := n.addStream(); err != nil {
			return err
		}
	}

	if n.parameters.Consumer != nil {
		if err := n.addConsumer(); err != nil {
			return err
		}
	}

	return nil
}

func (n *NatsJetstream) addStream() error {
	if n.jsctx == nil {
		return errors.Wrap(ErrNatsJetstreamAddStream, "Jetstream context is not setup")
	}

	var retention nats.RetentionPolicy

	switch n.parameters.Stream.Retention {
	case "workQueue":
		retention = nats.WorkQueuePolicy
	case "limits":
		retention = nats.LimitsPolicy
	case "interest":
		retention = nats.InterestPolicy
	default:
		return errors.Wrap(ErrNatsConfig, "unknown retention policy defined: "+n.parameters.Stream.Retention)
	}

	cfg := &nats.StreamConfig{
		Name:        n.parameters.Stream.Name,
		Subjects:    n.parameters.Stream.Subjects,
		Retention:   retention,
		AllowRollup: true, // https://docs.nats.io/nats-concepts/jetstream/streams#allowrollup
		MaxAge:      conditionJetstreamTTL,
	}

	if n.parameters.Stream.DuplicateWindow != 0 {
		cfg.Duplicates = n.parameters.Stream.DuplicateWindow
	}

	// check stream isn't already present
	for name := range n.jsctx.StreamNames() {
		if name == n.parameters.Stream.Name {
			_, err := n.jsctx.UpdateStream(cfg)
			return err
		}
	}

	_, err := n.jsctx.AddStream(cfg)
	if err != nil {
		return errors.Wrap(ErrNatsJetstreamAddStream, err.Error())
	}

	return nil
}

// AddConsumer adds a consumer for a stream
//
// Consumers are view into a NATs Jetstream
// multiple applications may bind to a consumer.
func (n *NatsJetstream) addConsumer() error {
	if n.jsctx == nil {
		return errors.Wrap(ErrNatsJetstreamAddConsumer, "Jetstream context is not setup")
	}

	// https://pkg.go.dev/github.com/nats-io/nats.go#ConsumerConfig
	cfg := &nats.ConsumerConfig{
		Durable:       n.parameters.Consumer.Name,
		MaxDeliver:    consumerMaxDeliver,
		AckPolicy:     consumerAckPolicy,
		AckWait:       n.parameters.Consumer.AckWait,
		MaxAckPending: n.parameters.Consumer.MaxAckPending,
		DeliverPolicy: nats.DeliverAllPolicy,
		DeliverGroup:  n.parameters.Consumer.QueueGroup,
		FilterSubject: n.parameters.Consumer.FilterSubject,
	}

	// add consumer if its not already present
	consumerInfo, err := n.jsctx.ConsumerInfo(n.parameters.Stream.Name, n.parameters.Consumer.Name)
	if err != nil {
		if errors.Is(err, nats.ErrConsumerNotFound) {
			if _, errAdd := n.jsctx.AddConsumer(n.parameters.Stream.Name, cfg); errAdd != nil {
				return errors.Wrap(errAdd, ErrNatsJetstreamAddConsumer.Error())
			}

			return nil
		}

		return errors.Wrap(err, ErrNatsJetstreamAddConsumer.Error()+" consumer.Name="+n.parameters.Consumer.Name)
	}

	// update consumer if its present
	if !n.consumerConfigIsEqual(consumerInfo) {
		if _, err := n.jsctx.UpdateConsumer(n.parameters.Stream.Name, cfg); err != nil {
			return errors.Wrap(err, ErrNatsJetstreamUpdateConsumer.Error())
		}
	}

	return nil
}

func (n *NatsJetstream) consumerConfigIsEqual(consumerInfo *nats.ConsumerInfo) bool {
	switch {
	case consumerInfo.Config.MaxDeliver != consumerMaxDeliver:
		return false
	case consumerInfo.Config.AckPolicy != consumerAckPolicy:
		return false
	case consumerInfo.Config.DeliverPolicy != consumerDeliverPolicy:
		return false
	case consumerInfo.Name != n.parameters.Consumer.Name:
		return false
	case consumerInfo.Config.Durable != n.parameters.Consumer.Name:
		return false
	case consumerInfo.Config.MaxAckPending != n.parameters.Consumer.MaxAckPending:
		return false
	case consumerInfo.Config.AckWait != n.parameters.Consumer.AckWait:
		return false
	case consumerInfo.Config.DeliverGroup != n.parameters.Consumer.QueueGroup:
		return false
	case consumerInfo.Config.FilterSubject != n.parameters.Consumer.FilterSubject:
		return false
	default:
		return true
	}
}

// Publish publishes an event onto the NATS Jetstream.
// The caller is responsible for message addressing and data serialization.
func (n *NatsJetstream) Publish(ctx context.Context, subjectSuffix string, data []byte) error {
	return n._publish(ctx, subjectSuffix, data, false)
}

// PublishOverwrite publishes an event and will overwrite any existing message with that subject in the queue
func (n *NatsJetstream) PublishOverwrite(ctx context.Context, subjectSuffix string, data []byte) error {
	return n._publish(ctx, subjectSuffix, data, true)
}

// rollupSubject when set to true will cause any previous messages with the same subject to be overwritten by this new msg.
// NOTE: The subject passed here will be prepended with the configured PublisherSubjectPrefix.
func (n *NatsJetstream) _publish(ctx context.Context, subjectSuffix string, data []byte, rollupSubject bool) error {
	if n.jsctx == nil {
		return errors.Wrap(ErrNatsJetstreamAddConsumer, "Jetstream context is not setup")
	}

	// retry publishing for a while
	options := []nats.PubOpt{
		nats.RetryAttempts(-1),
	}

	fullSubject := n.parameters.PublisherSubjectPrefix + "." + subjectSuffix

	msg := nats.NewMsg(fullSubject)
	msg.Data = data

	// inject otel trace context
	injectOtelTraceContext(ctx, msg)

	// https://docs.nats.io/nats-concepts/jetstream/streams#allowrollup
	if rollupSubject {
		msg.Header.Add("Nats-Rollup", "sub")
	}
	_, err := n.jsctx.PublishMsg(msg, options...)
	return err
}

func injectOtelTraceContext(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(msg.Header))
}

// Subscribe to all configured SubscribeSubjects
func (n *NatsJetstream) Subscribe(ctx context.Context) (MsgCh, error) {
	if n.jsctx == nil {
		return nil, errors.Wrap(ErrNatsJetstreamAddConsumer, "Jetstream context is not setup")
	}

	// Subscribe as a pull based subscriber
	if n.parameters.Consumer != nil && n.parameters.Consumer.Pull {
		if err := n.subscribeAsPull(ctx); err != nil {
			return nil, err
		}
	}

	// regular Async subscription
	for _, subject := range n.parameters.SubscribeSubjects {
		subscription, err := n.jsctx.Subscribe(subject, n.subscriptionCallback, nats.Durable(n.parameters.AppName))
		if err != nil {
			return nil, errors.Wrap(ErrSubscription, err.Error()+": "+subject)
		}

		n.subscriptions = append(n.subscriptions, subscription)
	}

	return n.subscriberCh, nil
}

// subscribeAsPull sets up the pull subscription
func (n *NatsJetstream) subscribeAsPull(_ context.Context) error {
	if n.jsctx == nil {
		return errors.Wrap(ErrNatsJetstreamAddConsumer, "Jetstream context is not setup")
	}

	for _, subject := range n.parameters.Consumer.SubscribeSubjects {
		subscription, err := n.jsctx.PullSubscribe(subject, n.parameters.Consumer.Name,
			nats.BindStream(n.parameters.Stream.Name))
		if err != nil {
			log.Printf("PullSubscribe with subject=%s, durable=%s, stream=%s => %v", subject, n.parameters.AppName,
				n.parameters.Stream.Name, err)
			return errors.Wrap(ErrSubscription, err.Error()+": "+subject)
		}

		n.subscriptions = append(n.subscriptions, subscription)
	}

	return nil
}

// XXX: the ergonomics here are weird, because we're handling potentially multiple subscriptions
// in a single call, and an error on any single retrieve just aborts the group operation.

// PullMsg pulls up to the batch count of messages from each pull-based subscription to
// subjects on the stream.
func (n *NatsJetstream) PullMsg(_ context.Context, batch int) ([]Message, error) {
	if n.jsctx == nil {
		return nil, errors.Wrap(ErrNatsJetstreamAddConsumer, "Jetstream context is not setup")
	}

	var hasPullSubscription bool
	var msgs []Message

	for _, subscription := range n.subscriptions {
		if subscription.Type() != nats.PullSubscription {
			continue
		}

		hasPullSubscription = true

		subMsgs, err := subscription.Fetch(batch)
		if err != nil {
			return nil, errors.Wrap(err, ErrNatsMsgPull.Error())
		}
		msgs = append(msgs, msgIfFromNats(subMsgs...)...)
	}

	if !hasPullSubscription {
		return nil, errors.Wrap(ErrNatsMsgPull, "no pull subscriptions to fetch from")
	}

	return msgs, nil
}

func (n *NatsJetstream) subscriptionCallback(msg *nats.Msg) {
	select {
	case <-time.After(subscriptionCallbackTimeout):
		_ = msg.NakWithDelay(nakDelay)
	case n.subscriberCh <- &natsMsg{msg: msg}:
	}
}

// Close drains any subscriptions and closes the NATS Jetstream connection.
func (n *NatsJetstream) Close() error {
	var errs error

	for _, subscription := range n.subscriptions {
		if err := subscription.Drain(); err != nil {
			errs = multierror.Append(err, err)
		}
	}

	if n.conn != nil {
		n.conn.Close()
	}

	return errs
}
