//nolint:wsl // it's useless
package registry

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrBadFormat = errors.New("bad worker id format")
)

type ControllerID interface {
	fmt.Stringer
	updateVersion(uint64)
	version() uint64
}

type workerUUID struct {
	appName   string
	uuid      uuid.UUID
	kvVersion uint64
}

func (id *workerUUID) String() string {
	return id.appName + "/" + id.uuid.String()
}

func ControllerIDFromString(s string) (ControllerID, error) {
	name, uuidStr, found := strings.Cut(s, "/")
	if !found {
		return nil, fmt.Errorf("%w: missing delimiter", ErrBadFormat)
	}
	id, err := uuid.Parse(uuidStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadFormat, err.Error())
	}
	return &workerUUID{
		appName: name,
		uuid:    id,
	}, nil
}

func (id *workerUUID) updateVersion(rev uint64) {
	id.kvVersion = rev
}

func (id *workerUUID) version() uint64 {
	return id.kvVersion
}

func GetID(app string) ControllerID {
	return &workerUUID{
		appName: app,
		uuid:    uuid.New(),
	}
}

func GetIDWithUUID(app string, id uuid.UUID) ControllerID {
	return &workerUUID{
		appName: app,
		uuid:    id,
	}
}

type activityRecord struct {
	LastActive time.Time `json:"last_active"`
}
