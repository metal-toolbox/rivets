package condition

import (
	"encoding/json"

	"github.com/google/uuid"
)

const (
	FirmwareInstall Kind = "firmwareInstall"
)

// FirmwareTaskParameters are the parameters set for a firmwareInstall condition
//
// nolint:govet // fieldalignment struct is easier to read in the current format
type FirmwareInstallTaskParameters struct {
	// Inventory identifier for the asset to install firmware on.
	AssetID uuid.UUID `json:"asset_id"`

	// Reset device BMC before firmware install
	ResetBMCBeforeInstall bool `json:"reset_bmc_before_install,omitempty"`

	// Force install given firmware regardless of current firmware version.
	ForceInstall bool `json:"force_install,omitempty"`

	// When defined, flasher will not perform any disruptive actions on the asset,
	// it will download the firmware to be installed and determine if the firmware is applicable for the device.
	//
	// No firmware installs will be attempted and if the device is powered off, it will not be powered on.
	DryRun bool `json:"dry_run,omitempty"`

	// When true, flasher will expect the host to be powered off before proceeding,
	// if the host is not already powered off - the install task will be failed.
	RequireHostPoweredOff bool `json:"require_host_powered_off,omitempty"`

	// Task priority is the task priority between 0 and 3
	// where 0 is the default and 3 is the max.
	//
	// Tasks are picked from the `queued` state based on the priority.
	//
	// When there are multiple tasks with the same priority,
	// the task CreatedAt attribute is considered.
	Priority int `json:"priority,omitempty"`

	// Firmwares is the list of firmwares to be installed.
	Firmwares []Firmware `json:"firmwares,omitempty"`

	// FirmwareSetID specifies the firmware set to be applied.
	FirmwareSetID uuid.UUID `json:"firmware_set_id,omitempty"`
}

func (p *FirmwareInstallTaskParameters) MustJSON() []byte {
	byt, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return byt
}

// Firmware holds attributes for a firmware object
type Firmware struct {
	ID        string   `yaml:"id" json:"id"`
	Vendor    string   `yaml:"vendor" json:"vendor"`
	FileName  string   `yaml:"filename" json:"filename"`
	Version   string   `yaml:"version" json:"version"`
	URL       string   `yaml:"URL" json:"URL"`
	Component string   `yaml:"component" json:"component"`
	Checksum  string   `yaml:"checksum" json:"checksum"`
	Models    []string `yaml:"models" json:"models"`
}
