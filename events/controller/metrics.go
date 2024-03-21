package controller

import (
	"strconv"
	"time"

	"github.com/metal-toolbox/rivets/condition"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	metricsNATSErrors              *prometheus.CounterVec
	metricsEventCounter            *prometheus.CounterVec
	metricsConditionRunTimeSummary *prometheus.SummaryVec
)

func init() {
	metricsNATSErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_errors",
			Help: "A count of errors while trying to use NATS.",
		},
		[]string{"operation"},
	)

	metricsEventCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_received",
			Help: "A counter metric to measure the total count of events received",
		},
		[]string{"valid", "response"}, // valid is true/false, response is ack/nack
	)

	metricsConditionRunTimeSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "condition_duration_seconds",
			Help: "A summary metric to measure the total time spent in completing each condition",
		},
		[]string{"condition", "state"},
	)

}

// spanEvent adds a span event along with the given attributes.
//
// event here is arbitrary and can be in the form of strings like - publishCondition, updateCondition etc
func spanEvent(span trace.Span, condition *condition.Condition, controllerID, event string) {
	span.AddEvent(event, trace.WithAttributes(
		attribute.String("controllerID", controllerID),
		attribute.String("conditionID", condition.ID.String()),
		attribute.String("conditionKind", string(condition.Kind)),
	))
}

func metricsNATSError(op string) {
	metricsNATSErrors.WithLabelValues(op).Inc()
}

func metricsEventsCounter(valid bool, response string) {
	metricsEventCounter.With(
		prometheus.Labels{
			"valid":    strconv.FormatBool(valid),
			"response": response,
		}).Inc()
}
func registerConditionRuntimeMetric(startTS time.Time, state string) {
	metricsConditionRunTimeSummary.With(
		prometheus.Labels{
			"condition": string(condition.FirmwareInstall),
			"state":     state,
		},
	).Observe(time.Since(startTS).Seconds())
}
