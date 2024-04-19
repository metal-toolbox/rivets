package fleetdb

const (
	// FleetDB attribute namespace for device vendor, model, serial attributes.
	ServerVendorAttributeNS = "sh.hollow.alloy.server_vendor_attributes"

	// FleetDB attribute namespace for the BMC address.
	ServerAttributeNSBmcAddress = "sh.hollow.bmc_info"

	// FleetDB attribute namespace for firmware set labels.
	FirmwareSetAttributeNS = "sh.hollow.firmware_set.labels"

	// FleetDB attribute namespace for inband firmware information.
	FirmwareVersionInbandNS = "sh.hollow.alloy.inband.firmware"

	// FleetDB attribute namespace for outofband firmware information.
	FirmwareVersionOutofbandNS = "sh.hollow.alloy.outofband.firmware"

	// FleetDB attribute namespace for outofband bios configuration.
	BiosConfigOutofbandNS = "sh.hollow.alloy.outofband.bios_configuration"

	// FleetDB attribute namespace for inband bios configuration.
	BiosConfigInbandNS = "sh.hollow.alloy.inband.bios_configuration"

	// FleetDB attribute namespace for inband component attributes/metadata.
	ComponentAttributeInbandNS = "sh.hollow.alloy.inband.metadata"

	// FleetDB attribute namespace for outofband component attributes/metadata.
	ComponentAttributeOutofbandNS = "sh.hollow.alloy.outofband.metadata"

	// FleetDB attribute for installed firmware.
	FirmwareVersionNSInstalledAttribute = "firmware.installed"

	// FleetDB attribute namespace for inband component status information.
	StatusInbandNS = "sh.hollow.alloy.inband.status"

	// FleetDB attribute namespace for outofband component status information.
	StatusOutofbandNS = "sh.hollow.alloy.outofband.status"

	// FleetDB attribute for component health status.
	StatusNSHealthAttribute = "status.health"

	// Serserverservice attribute for errors that occurred when connecting/collecting inventory from the bmc are stored here.
	ServerNSBMCErrorsAttribute = "sh.hollow.alloy.outofband.server_bmc_errors"
)
