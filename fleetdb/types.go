package fleetdb

import (
	"github.com/bmc-toolbox/common"
	"github.com/google/uuid"
)

// FirmwareVersionedAttribute holds component firmware information.
type FirmwareVersionedAttribute struct {
	Firmware *common.Firmware `json:"firmware,omitempty"`
	UUID     *uuid.UUID       `json:"uuid,omitempty"` // UUID references firmware UUID identified in serverservice based on component/device attributes.
	Vendor   string           `json:"vendor,omitempty"`
}

// StatusVersionedAttribute holds component status information.
type StatusVersionedAttribute struct {
	Status        *common.Status `json:"status,omitempty"`
	NICPortStatus *NICPortStatus `json:"nic_port_status,omitempty"`
	SmartStatus   string         `json:"smart_status,omitempty"`
}

// NICPortStatus holds the NIC port status which includes the health status and link status information.
type NICPortStatus struct {
	*common.Status
	ID                   string `json:"id,omitempty"`
	MacAddress           string `json:"macaddress,omitempty"`
	ActiveLinkTechnology string `json:"active_link_technology,omitempty"`
	LinkStatus           string `json:"link_status,omitempty"`
	MTUSize              int    `json:"mtu_size,omitempty"`
	AutoSpeedNegotiation bool   `json:"autospeednegotiation,omitempty"`
}

type MetadataAttribute struct {
	Data map[string]string `json:"metadata,omitempty"`
}

type ServerVendorAttribute struct {
	Model  string `json:"model"`
	Vendor string `json:"vendor"`
	Serial string `json:"serial"`
}

type BMCAttribute struct {
	Address string `json:"address"`
}

// ComponentAtributes are generic component attributes
type ComponentCommonAttributes struct {
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
