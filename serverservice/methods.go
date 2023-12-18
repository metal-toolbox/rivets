package serverservice

import (
	"encoding/json"
	"sort"

	rt "github.com/metal-toolbox/rivets/types"
	"github.com/pkg/errors"
	ss "go.hollow.sh/serverservice/pkg/api/v1"
)

func UnpackVersionedAttribute(attr *ss.VersionedAttributes, dst any) error {
	derr := errors.New("error deserializing versioned attribute")
	err := json.Unmarshal(attr.Data, dst)
	if err != nil {
		return errors.Wrap(derr, err.Error())
	}

	return nil
}

func UnpackAttribute(attr *ss.Attributes, dst any) error {
	derr := errors.New("error deserializing attribute")
	err := json.Unmarshal(attr.Data, dst)
	if err != nil {
		return errors.Wrap(derr, err.Error())
	}

	return nil
}

// ConvertComponents converts Serverservice components to the rivets component type
// nolint:gocyclo // component conversion is cyclomatic
func ConvertComponents(cs []ss.ServerComponent) []*rt.Component {
	if cs == nil {
		return nil
	}

	set := make([]*rt.Component, 0, len(cs))
	for idx := range cs {
		c := cs[idx]
		component := &rt.Component{
			ID:        c.UUID.String(),
			Name:      c.ComponentTypeSlug,
			Vendor:    c.Vendor,
			Model:     c.Model,
			Serial:    c.Serial,
			UpdatedAt: c.UpdatedAt,
		}

		for _, vattr := range c.VersionedAttributes {
			vattr := vattr
			// TODO: set the most current data from either the inband/outofband NS
			switch vattr.Namespace {
			case FirmwareVersionInbandNS, FirmwareVersionOutofbandNS:
				fw := &FirmwareVersionedAttribute{}
				if err := UnpackVersionedAttribute(&vattr, fw); err == nil {
					component.Firmware = fw.Firmware
				}

			case StatusOutofbandNS, StatusInbandNS:
				st := &StatusVersionedAttribute{}
				if err := UnpackVersionedAttribute(&vattr, st); err == nil {
					component.Status = st.Status
				}
			}
		}

		for _, attr := range c.Attributes {
			attr := attr
			switch attr.Namespace {
			case ComponentAttributeInbandNS, ComponentAttributeOutofbandNS:
				a := &rt.ComponentAttributes{}
				if err := UnpackAttribute(&attr, a); err == nil {
					component.Attributes = a
				}
			}
		}

		set = append(set, component)
	}

	sort.Slice(set, func(i, j int) bool {
		return set[i].Name < set[j].Name
	})

	return set
}

// ConvertServer converts the serverservice server to the rivets Server type
func ConvertServer(s *ss.Server) *rt.Server {
	if s == nil {
		return nil
	}

	server := &rt.Server{
		ID:        s.UUID.String(),
		Facility:  s.FacilityCode,
		Name:      s.Name,
		UpdatedAt: s.UpdatedAt,
	}

	// TODO: set the most current data from either the inband/outofband NS
	for _, attr := range s.Attributes {
		attr := attr

		switch attr.Namespace {
		case ServerAttributeNSBmcAddress:
			bmcAttr := BMCAttribute{}
			if err := UnpackAttribute(&attr, &bmcAttr); err == nil {
				server.BMCAddress = bmcAttr.Address
			}

		case ServerAttributeNSVendor:
			svmAttr := ServerVendorAttribute{}
			if err := UnpackAttribute(&attr, &svmAttr); err == nil {
				server.Vendor = svmAttr.Vendor
				server.Model = svmAttr.Model
				server.Serial = svmAttr.Serial
			}
		case BiosConfigInbandNS, BiosConfigOutofbandNS:
			bioscfg := map[string]string{}
			if err := UnpackAttribute(&attr, &bioscfg); err == nil {
				server.BIOSCfg = bioscfg
			}
		}
	}

	return server
}
