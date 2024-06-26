package kv

import (
	"fmt"
)

// KV Key for the Condition Status Values
func StatusValueKVKey(facilityCode, conditionID string) string {
	return fmt.Sprintf("%s.%s", facilityCode, conditionID)
}
