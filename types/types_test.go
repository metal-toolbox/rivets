package types

import (
	"testing"

	common "github.com/metal-toolbox/bmc-common"
	"github.com/stretchr/testify/assert"
)

func TestComponentsByNameModel(t *testing.T) {
	component1 := &Component{
		Name:     "cpu",
		Serial:   "123",
		Vendor:   "Intel",
		Model:    "Core i7",
		Firmware: &common.Firmware{Installed: "v1.0"},
	}

	component2 := &Component{
		Name:     "gpu",
		Serial:   "456",
		Vendor:   "NVIDIA",
		Model:    "GeForce RTX 3080",
		Firmware: &common.Firmware{Installed: "v2.0"},
	}

	component3 := &Component{
		Name:     "cpu",
		Serial:   "789",
		Vendor:   "AMD",
		Model:    "Ryzen 9",
		Firmware: &common.Firmware{Installed: "v1.5"},
	}

	biosComponent := &Component{
		Name:     "bios",
		Serial:   "111",
		Vendor:   "AMI",
		Model:    "BIOS Model",
		Firmware: &common.Firmware{Installed: "v1.2"},
	}

	bmcComponent := &Component{
		Name:     "bmc",
		Serial:   "222",
		Vendor:   "Supermicro",
		Model:    "BMC Model",
		Firmware: &common.Firmware{Installed: "v3.0"},
	}

	components := Components{component1, component2, component3, biosComponent, bmcComponent}

	testCases := []struct {
		name     string
		slug     string
		models   []string
		expected *Component
	}{
		{
			name:     "Single Match",
			slug:     "gpu",
			models:   []string{"RTX 3080"},
			expected: component2,
		},
		{
			name:     "No Match",
			slug:     "memory",
			models:   []string{"DDR4"},
			expected: nil,
		},
		{
			name:     "Multiple Matches",
			slug:     "cpu",
			models:   []string{"Ryzen 9"},
			expected: component3,
		},
		{
			name:     "Slug Match",
			slug:     "cpu",
			models:   []string{},
			expected: nil,
		},
		{
			name:     "Slug BIOS",
			slug:     "bios",
			models:   []string{},
			expected: biosComponent,
		},
		{
			name:     "Slug upper case BIOS",
			slug:     "BIOS",
			models:   []string{},
			expected: biosComponent,
		},
		{
			name:     "Slug BMC",
			slug:     "bmc",
			models:   []string{},
			expected: bmcComponent,
		},
		{
			name:     "Slug BMC with Models",
			slug:     "bmc",
			models:   []string{"BMC Model"},
			expected: bmcComponent,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := components.ByNameModel(tc.slug, tc.models)
			assert.Equal(t, tc.expected, result)
		})
	}
}
