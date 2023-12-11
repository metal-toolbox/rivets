package condition

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/metal-toolbox/rivets/events/registry"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

type ErrQueryStatus struct {
	err       error
	kvBucket  string
	lookupKey string
	workerID  string
}

func (e *ErrQueryStatus) Error() string {
	s := fmt.Sprintf("Condition status lookup error, bucket=%s", e.kvBucket)

	if e.lookupKey != "" {
		s += fmt.Sprintf(" lookupKey=%s", e.lookupKey)
	}

	if e.workerID != "" {
		s += fmt.Sprintf(" workerID=%s", e.workerID)
	}

	if e.err != nil {
		s += fmt.Sprintf(" err=%s", e.err.Error())
	}

	return s
}

func newErrQueryStatus(err error, kvBucket, lookupKey, workerID string) error {
	return &ErrQueryStatus{
		err:       err,
		kvBucket:  kvBucket,
		lookupKey: lookupKey,
		workerID:  workerID,
	}
}

const (
	StatusValueVersion int32 = 1
)

// StatusValue is the canonical structure for reporting status of an ongoing task
type StatusValue struct {
	UpdatedAt  time.Time       `json:"updated"`
	WorkerID   string          `json:"worker"`
	Target     string          `json:"target"`
	TraceID    string          `json:"traceID"`
	SpanID     string          `json:"spanID"`
	State      string          `json:"state"`
	Status     json.RawMessage `json:"status"`
	MsgVersion int32           `json:"msgVersion"`
	// WorkSpec json.RawMessage `json:"spec"` XXX: for re-publish use-cases
}

// MustBytes sets the version field of the StatusValue so any callers don't have
// to deal with it. It will panic if we cannot serialize to JSON for some reason.
func (v *StatusValue) MustBytes() []byte {
	v.MsgVersion = StatusValueVersion
	byt, err := json.Marshal(v)
	if err != nil {
		panic("unable to serialize status value: " + err.Error())
	}
	return byt
}

func UnmarshalStatusValue(b []byte) (*StatusValue, error) {
	sv := &StatusValue{}
	if err := json.Unmarshal(b, &sv); err != nil {
		return nil, err
	}

	return sv, nil
}

// TaskState holds the value for the task state in the KV.
//
// note: this differs from the condition.State, can these be merged?
type TaskState int

const (
	NotStarted    TaskState = iota
	InProgress              // another worker has started it, is still around and updated recently
	Complete                // task is done
	Orphaned                // the worker that started this task doesn't exist anymore
	Indeterminate           // we got an error in the process of making the check
)

// CheckConditionInProgress returns the status of the task from the KV store
//
//nolint:gocyclo // status checks are cyclomatic
func CheckConditionInProgress(conditionID, facilityCode, kvBucket string, js nats.JetStreamContext) (TaskState, error) {
	handle, err := js.KeyValue(kvBucket)
	if err != nil {
		errKV := errors.Wrap(err, "bind to status KV bucket for condition lookup failed")
		return Indeterminate, newErrQueryStatus(errKV, kvBucket, "", "")
	}

	lookupKey := fmt.Sprintf("%s.%s", facilityCode, conditionID)
	entry, err := handle.Get(lookupKey)
	switch err {
	case nats.ErrKeyNotFound:
		// This should be by far the most common path through this code.
		return NotStarted, nil

	case nil:
		break // we'll handle this outside the switch

	default:
		return Indeterminate, newErrQueryStatus(err, kvBucket, lookupKey, "")
	}

	// we have an status entry for this condition, is is complete?
	sv := StatusValue{}
	if errJSON := json.Unmarshal(entry.Value(), &sv); errJSON != nil {
		errJSON = errors.Wrap(errJSON, "unable to construct a sane status for condition")
		return Indeterminate, newErrQueryStatus(errJSON, kvBucket, lookupKey, "")
	}

	if StateIsComplete(State(sv.State)) {
		return Complete, nil
	}

	// is the worker handling this condition alive?
	controllerID, err := registry.ControllerIDFromString(sv.WorkerID)
	if err != nil {
		errWorker := errors.Wrap(err, "bad worker ID")
		return Indeterminate, newErrQueryStatus(errWorker, kvBucket, lookupKey, sv.WorkerID)
	}

	_, err = registry.LastContact(controllerID)
	switch err {
	case nats.ErrKeyNotFound:
		// the data for this worker aged-out, it's no longer active
		// XXX: the most conservative thing to do here is to return
		// indeterminate but most times this will indicate that the
		// worker crashed/restarted and this task should be restarted.

		// We're going to restart this condition when we return from
		// this function. Use the KV handle we have to delete the
		// existing task key.
		if err = handle.Delete(lookupKey); err != nil {
			errWorker := errors.Wrap(err, "unable to delete existing condition status")

			return Indeterminate, newErrQueryStatus(errWorker, kvBucket, lookupKey, sv.WorkerID)
		}

		return Orphaned, nil

	case nil:
		return InProgress, nil

	default:
		errWorker := errors.Wrap(err, "error looking up worker last contact")
		return Indeterminate, newErrQueryStatus(errWorker, kvBucket, lookupKey, sv.WorkerID)
	}
}
