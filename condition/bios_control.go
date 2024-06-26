package condition

import (
	"encoding/json"

	"github.com/google/uuid"
)

type (
	BiosControlAction string
)

const (
	// BiosControl identifies the Condition kind to configure the BIOS.
	BiosControl Kind = "biosControl"

	// ResetSettings will reset the BIOS to default settings.
	ResetSettings BiosControlAction = "reset_settings"
)

// BiosControlTaskParameters are the parameters that are passed for the BiosControl condition.
type BiosControlTaskParameters struct {
	// Identifier for the Asset in the Asset store.
	//
	// Required: true
	AssetID uuid.UUID `json:"asset_id"`

	// The bios control action to be performed
	//
	// Required: true
	Action BiosControlAction `json:"action"`
}

func NewBiosControlTaskParameters(assetID uuid.UUID, action BiosControlAction) *BiosControlTaskParameters {
	return &BiosControlTaskParameters{
		AssetID: assetID,
		Action:  action,
	}
}

func NewBiosControlParametersFromCondition(condition *Condition) (*BiosControlTaskParameters, error) {
	b := &BiosControlTaskParameters{}
	err := json.Unmarshal(condition.Parameters, b)

	return b, err
}
