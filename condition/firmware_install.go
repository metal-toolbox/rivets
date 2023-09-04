package condition

import "github.com/google/uuid"

const (
	FirmwareInstall Kind = "firmwareInstall"
)

// FirmwareTaskParameters are the parameters set for a firmwareInstall condition
//
// nolint:govet // fieldalignment struct is easier to read in the current format
type FirmwareInstallTaskParameters struct {
	// Inventory identifier for the asset to install firmware on.
	AssetID uuid.UUID `json:"assetID"`

	// Reset device BMC before firmware install
	ResetBMCBeforeInstall bool `json:"resetBMCBeforeInstall,omitempty"`

	// Force install given firmware regardless of current firmware version.
	ForceInstall bool `json:"forceInstall,omitempty"`

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
	FirmwareSetID uuid.UUID `json:"firmwareSetID,omitempty"`
}

// Firmware holds attributes for a firmware object
type Firmware struct {
	ID        string   `yaml:"id"`
	Vendor    string   `yaml:"vendor"`
	FileName  string   `yaml:"filename"`
	Version   string   `yaml:"version"`
	URL       string   `yaml:"URL"`
	Component string   `yaml:"component"`
	Checksum  string   `yaml:"checksum"`
	Models    []string `yaml:"models"`
}
