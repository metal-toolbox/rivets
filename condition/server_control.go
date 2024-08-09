package condition

import (
	"encoding/json"

	"github.com/google/uuid"
)

type (
	ServerControlAction string
)

const (
	// ServerControl identifies the Condition kind to power on/off/cycle servers and set the next boot device.
	ServerControl Kind = "serverControl"

	// SetPowerState sets the server power state
	//
	// Accepted ActionParameter value, one of:
	// - on
	// - off
	// - cycle
	// - reset
	// - soft
	SetPowerState ServerControlAction = "set_power_state"

	// GetPowerState retrieves the current power state on the server.
	GetPowerState ServerControlAction = "get_power_state"

	// SetNextBootDevice sets the next boot device
	//
	// Required: false
	//
	// Accepted ActionParameter value, one of:
	// - bios
	// - cdrom
	// - diag
	// - floppy
	// - disk
	// - none
	// - pxe
	// - remote_drive
	// - sd_card
	// - usb
	// - utilities
	SetNextBootDevice ServerControlAction = "set_next_boot_device"

	// PowerCycleBMC power cycles the BMC
	PowerCycleBMC ServerControlAction = "power_cycle_bmc"

	// Run a basic firmware test
	ValidateFirmware ServerControlAction = "validate_firmware"
)

// ServerControlTaskParameters are the parameters that are passed for the ServerControl condition.
// nolint:govet // prefer readability over fieldalignment for this case
type ServerControlTaskParameters struct {
	// Identifier for the Asset in the Asset store.
	//
	// Required: true
	AssetID uuid.UUID `json:"asset_id"`

	// The server control action to be performed
	//
	// Required: true
	Action ServerControlAction `json:"action"`

	// Action parameter to the ServerControlAction
	//
	// Required for SetPowerState, SetNextBootDevice actions
	ActionParameter string `json:"action_parameter"`

	// Persist next boot device
	// For use with SetNextBootDevice action.
	// Required: false
	SetNextBootDevicePersistent bool `json:"set_next_boot_device_persistent"`

	// Set next boot device to be UEFI
	// For use with SetNextBootDevice action.
	// Required: false
	SetNextBootDeviceEFI bool `json:"set_next_boot_device_efi"`

	// The timeout in seconds for a ValidateFirmware action
	// Required for ValidateFirmware.
	ValidateFirmwareTimeout uint `json:"validate_firmware_timeout"`
}

func (p *ServerControlTaskParameters) Unmarshal(r json.RawMessage) error {
	return json.Unmarshal(r, p)
}

func (p *ServerControlTaskParameters) Marshal() (json.RawMessage, error) {
	return json.Marshal(p)
}

func NewServerControlTaskParameters(assetID uuid.UUID, action ServerControlAction, controlParam string, bootDevicePersistent, efiBoot bool) *ServerControlTaskParameters {
	return &ServerControlTaskParameters{
		AssetID:                     assetID,
		Action:                      action,
		ActionParameter:             controlParam,
		SetNextBootDevicePersistent: bootDevicePersistent,
		SetNextBootDeviceEFI:        efiBoot,
	}
}
