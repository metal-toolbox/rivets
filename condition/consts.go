package condition

import "fmt"

const (
	// condition status considered stale after this period
	StatusStaleThreshold = StaleThreshold

	// controller considered dead after this period
	LivenessStaleThreshold = StaleThreshold
)

// Returns the stream subject with which the condition is to be published.
func StreamSubject(facilityCode string, conditionKind Kind) string {
	return fmt.Sprintf("%s.servers.%s", facilityCode, conditionKind)
}

// KV Key for the Condition Status Values
func StatusValueKVKey(facilityCode, conditionID string) string {
	return fmt.Sprintf("%s.%s", facilityCode, conditionID)
}
