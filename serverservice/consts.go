package serverservice

const (
	// Serverservice attribute namespace for device vendor, model, serial attributes.
	ServerAttributeNSVendor = "sh.hollow.alloy.server_vendor_attributes"

	// Serverservice attribute namespace for the BMC address.
	ServerAttributeNSBmcAddress = "sh.hollow.bmc_info"

	// Serverservice attribute namespace for firmware set labels.
	FirmwareAttributeNSFirmwareSetLabels = "sh.hollow.firmware_set.labels"

	// Serverservice attribute namespace for inband firmware information.
	FirmwareVersionInbandNS = "sh.hollow.alloy.inband.firmware"

	// Serverservice attribute namespace for outofband firmware information.
	FirmwareVersionOutofbandNS = "sh.hollow.alloy.outofband.firmware"

	// Serverservice attribute for installed firmware.
	FirmwareVersionNSInstalledAttribute = "firmware.installed"

	// Serverservice attribute namespace for inband component status information.
	StatusInbandNS = "sh.hollow.alloy.inband.status"

	// Serverservice attribute namespace for outofband component status information.
	StatusOutofbandNS = "sh.hollow.alloy.outofband.status"

	// Serverservice attribute for component health status.
	StatusNSHealthAttribute = "status.health"
)
