package condition

import "github.com/google/uuid"

type (
	VirtualMediaType        string
	VirtualMediaMountMethod string
)

const (
	// VirtualMediaMount condition kind uploads, mounts a virtual media disk image/iso.
	VirtualMediaMount Kind = "virtualMediaMount"

	MediaTypeFloppy VirtualMediaType = "floppy"
	MediaTypeISO    VirtualMediaType = "iso"

	// MountMethodUpload identifies the means of mounting the virtual media by uploading it onto the BMC.
	MountMethodUpload VirtualMediaMountMethod = "upload"

	// MountMethodUpload identifies the means of mounting the virtual media by linking the BMC to the given URL.
	MountMethodURL VirtualMediaMountMethod = "url"
)

// VirtualMediaTaskParameters are the parameters set for a VirtualMedia condition.
//
// nolint:govet // prefer readability over fieldalignment for this case
type VirtualMediaTaskParameters struct {
	// MountMethod specifies the means of obtaining the image to be mounted
	//
	// Required: true
	MountMethod VirtualMediaMountMethod `json:"mount_method"`

	// ImageURL is a URL accessible to the Disko controller and the BMC to download the image.
	//
	// Required: true
	ImageURL string `json:"image_uri"`

	// MediaType indicates the kind of media being uploaded/mounted
	//
	// Required: true
	MediaType VirtualMediaType `json:"media_type"`

	// Identifier for the Asset in the Asset store.
	//
	// Required: true
	AssetID uuid.UUID `json:"asset_id"`
}

func NewVirtualMediaTaskParameters(assetID uuid.UUID, imageURL string, mediaType VirtualMediaType, mountMethod VirtualMediaMountMethod) *VirtualMediaTaskParameters {
	return &VirtualMediaTaskParameters{
		AssetID:     assetID,
		ImageURL:    imageURL,
		MediaType:   mediaType,
		MountMethod: mountMethod,
	}
}
