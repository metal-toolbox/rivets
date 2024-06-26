package controller

import (
	"testing"

	"github.com/google/uuid"
	"github.com/metal-toolbox/rivets/condition"
	"github.com/metal-toolbox/rivets/events"
	"github.com/metal-toolbox/rivets/events/registry"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewTaskRepository(t *testing.T) {
	srv := startJetStreamServer(t)
	defer shutdownJetStream(t, srv)
	natsConn, _ := jetStreamContext(t, srv) // nc is closed on evJS.Close(), js needs no cleanup
	evJS := events.NewJetstreamFromConn(natsConn)
	defer evJS.Close()

	condID := uuid.New()
	serverID := uuid.New()
	contID, _ := registry.ControllerIDFromString("foobar")
	logger := logrus.New()

	natsTaskRepo, err := NewNatsConditionTaskRepository(
		"foo",
		condID.String(),
		serverID.String(),
		"fac-13",
		condition.FirmwareInstall,
		contID,
		1,
		evJS,
		logger,
	)

	assert.NotNil(t, err)
	nctr, ok := natsTaskRepo.(*NatsConditionTaskRepository)
	assert.True(t, ok)

	assert.Equal(t, condition.TaskKVRepositoryBucket, nctr.bucketName)
	assert.Equal(t, "fac-13", nctr.facilityCode)
	assert.Equal(t, condID.String(), nctr.conditionID)
	assert.Equal(t, serverID.String(), nctr.serverID)
	assert.Equal(t, condition.FirmwareInstall, nctr.conditionKind)
	assert.Equal(t, contID.String(), nctr.conditionID)
	assert.Equal(t, logger, nctr.log)
	assert.Equal(t, 0, nctr.lastRev)
}
