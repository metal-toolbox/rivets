package types

import (
	"time"

	"github.com/bmc-toolbox/common"
)

type Component struct {
	UpdatedAt  time.Time            `json:"updated,omitempty"`
	Firmware   *common.Firmware     `json:"firmware,omitempty"`
	Status     *common.Status       `json:"status,omitempty"`
	Attributes *ComponentAttributes `json:"attributes,omitempty"`
	Name       string               `json:"name,omitempty"`
	Vendor     string               `json:"vendor,omitempty"`
	Model      string               `json:"model,omitempty"`
	Serial     string               `json:"serial,omitempty"`
}

type Server struct {
	UpdatedAt  time.Time         `json:"updated,omitempty"`
	BIOSCfg    map[string]string `json:"bios_cfg,omitempty"`
	ID         string            `json:"id,omitempty"`
	Facility   string            `json:"facility,omitempty"`
	Name       string            `json:"name,omitempty"`
	BMCAddress string            `json:"bmc_address,omitempty"`
	Vendor     string            `json:"vendor,omitempty"`
	Model      string            `json:"model,omitempty"`
	Serial     string            `json:"serial,omitempty"`
	Status     string            `json:"status,omitempty"`
	Components []*Component      `json:"components,omitempty"`
}

// ComponentAtributes are generic component attributes
type ComponentAttributes struct {
	Capabilities                 []*common.Capability `json:"capabilities,omitempty"`
	Metadata                     map[string]string    `json:"metadata,omitempty"`
	ID                           string               `json:"id,omitempty"`
	ChassisType                  string               `json:"chassis_type,omitempty"`
	Description                  string               `json:"description,omitempty"`
	ProductName                  string               `json:"product_name,omitempty"`
	InterfaceType                string               `json:"interface_type,omitempty"`
	Slot                         string               `json:"slot,omitempty"`
	Architecture                 string               `json:"architecture,omitempty"`
	MacAddress                   string               `json:"macaddress,omitempty"`
	SupportedControllerProtocols string               `json:"supported_controller_protocol,omitempty"`
	SupportedDeviceProtocols     string               `json:"supported_device_protocol,omitempty"`
	SupportedRAIDTypes           string               `json:"supported_raid_types,omitempty"`
	PhysicalID                   string               `json:"physid,omitempty"`
	FormFactor                   string               `json:"form_factor,omitempty"`
	PartNumber                   string               `json:"part_number,omitempty"`
	OemID                        string               `json:"oem_id,omitempty"`
	DriveType                    string               `json:"drive_type,omitempty"`
	StorageController            string               `json:"storage_controller,omitempty"`
	BusInfo                      string               `json:"bus_info,omitempty"`
	WWN                          string               `json:"wwn,omitempty"`
	Protocol                     string               `json:"protocol,omitempty"`
	SmartStatus                  string               `json:"smart_status,omitempty"`
	SmartErrors                  []string             `json:"smart_errors,omitempty"`
	PowerCapacityWatts           int64                `json:"power_capacity_watts,omitempty"`
	SizeBytes                    int64                `json:"size_bytes,omitempty"`
	CapacityBytes                int64                `json:"capacity_bytes,omitempty" diff:"immutable"`
	ClockSpeedHz                 int64                `json:"clock_speed_hz,omitempty"`
	Cores                        int                  `json:"cores,omitempty"`
	Threads                      int                  `json:"threads,omitempty"`
	SpeedBits                    int64                `json:"speed_bits,omitempty"`
	SpeedGbps                    int64                `json:"speed_gbps,omitempty"`
	BlockSizeBytes               int64                `json:"block_size_bytes,omitempty"`
	CapableSpeedGbps             int64                `json:"capable_speed_gbps,omitempty"`
	NegotiatedSpeedGbps          int64                `json:"negotiated_speed_gbps,omitempty"`
	Oem                          bool                 `json:"oem,omitempty"`
}
