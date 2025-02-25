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

	// ResetBiosConfig will reset the BIOS to default settings.
	ResetConfig BiosControlAction = "reset_config"

	// SetBiosConfig will set a new BIOS config
	SetConfig BiosControlAction = "set_config"
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

	// The URL for the bios configuration settings file.
	// Needed for BiosControlAction.SetConfig
	//
	// Required: false
	BiosConfigURL string `json:"bios_config_url,omitempty"`
}

func (p *BiosControlTaskParameters) Unmarshal(r json.RawMessage) error {
	return json.Unmarshal(r, p)
}

func (p *BiosControlTaskParameters) Marshal() (json.RawMessage, error) {
	return json.Marshal(p)
}

func (p *BiosControlTaskParameters) MustJSON() []byte {
	byt, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return byt
}

func NewBiosControlTaskParameters(assetID uuid.UUID, action BiosControlAction, configURL string) *BiosControlTaskParameters {
	return &BiosControlTaskParameters{
		AssetID:       assetID,
		Action:        action,
		BiosConfigURL: configURL,
	}
}

func NewBiosControlParametersFromCondition(condition *Condition) (*BiosControlTaskParameters, error) {
	b := &BiosControlTaskParameters{}
	err := json.Unmarshal(condition.Parameters, b)

	return b, err
}
