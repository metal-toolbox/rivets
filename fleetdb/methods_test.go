package fleetdb

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	ss "github.com/metal-toolbox/fleetdb/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestConvertComponents(t *testing.T) {
	// Test case 1: Empty input
	cs := []ss.ServerComponent{}
	result := ConvertComponents(cs)
	assert.Empty(t, result, "ConvertComponents should return an empty slice for empty input")

	// Test case 2: Conversion with valid input
	cs = []ss.ServerComponent{
		{
			UUID:                uuid.New(),
			ServerUUID:          uuid.New(),
			Name:                "CPU",
			Vendor:              "Intel",
			Model:               "Xeon",
			Serial:              "ABC123",
			Attributes:          []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{},
			ComponentTypeID:     "cpu-type",
			ComponentTypeName:   "cpu",
			ComponentTypeSlug:   "cpu",
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
		{
			UUID:                uuid.New(),
			ServerUUID:          uuid.New(),
			Name:                "Memory",
			Vendor:              "Samsung",
			Model:               "DDR4",
			Serial:              "XYZ456",
			Attributes:          []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{},
			ComponentTypeID:     "memory-type",
			ComponentTypeName:   "memory",
			ComponentTypeSlug:   "memory",
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
	}

	result = ConvertComponents(cs)

	assert.Len(t, result, 2, "ConvertComponents should return a slice with two elements")

	// Sort the result for consistent comparison
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	// Check the converted components
	assert.Equal(t, cs[0].UUID.String(), result[0].ID)
	assert.Equal(t, cs[0].ComponentTypeSlug, result[0].Name)
	assert.Equal(t, cs[0].Vendor, result[0].Vendor)
	assert.Equal(t, cs[0].Model, result[0].Model)
	assert.Equal(t, cs[0].Serial, result[0].Serial)

	assert.Equal(t, cs[1].UUID.String(), result[1].ID)
	assert.Equal(t, cs[1].ComponentTypeSlug, result[1].Name)
	assert.Equal(t, cs[1].Vendor, result[1].Vendor)
	assert.Equal(t, cs[1].Model, result[1].Model)
	assert.Equal(t, cs[1].Serial, result[1].Serial)

	// Test case 3: Conversion with versioned attributes
	cs = []ss.ServerComponent{
		{
			UUID:       uuid.New(),
			ServerUUID: uuid.New(),
			Name:       "BIOS",
			Vendor:     "Dell",
			Model:      "Version X",
			Serial:     "DEF789",
			Attributes: []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{
				{
					Namespace: FirmwareVersionInbandNS,
					Data:      json.RawMessage(`{"firmware": {"installed": "1.2.3"}}`),
				},
				{
					Namespace: StatusInbandNS,
					Data:      json.RawMessage(`{"status": {"state": "ok"}}`),
				},
			},
			ComponentTypeID:   "bios-type",
			ComponentTypeName: "BIOS",
			ComponentTypeSlug: "BIOS",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}

	result = ConvertComponents(cs)

	assert.Len(t, result, 1, "ConvertComponents should return a slice with one element")

	// Check the converted component with versioned attributes
	assert.Equal(t, cs[0].UUID.String(), result[0].ID)
	assert.Equal(t, cs[0].Name, result[0].Name)
	assert.Equal(t, cs[0].Vendor, result[0].Vendor)
	assert.Equal(t, cs[0].Model, result[0].Model)
	assert.Equal(t, cs[0].Serial, result[0].Serial)

	assert.NotNil(t, result[0].Firmware)
	assert.Equal(t, "1.2.3", result[0].Firmware.Installed)

	assert.NotNil(t, result[0].Status)
	assert.Equal(t, "ok", result[0].Status.State)
}
