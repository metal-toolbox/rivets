package condition

import "github.com/google/uuid"

type (
	VirtualMediaType string
)

const (
	// VirtualMediaMount condition kind uploads, mounts a virtual media disk image/iso.
	VirtualMediaMount Kind = "virtualMediaMount"

	MediaTypeFloppy VirtualMediaType = "floppy"
	MediaTypeISO    VirtualMediaType = "iso"
)

// VirtualMediaTaskParameters are the parameters set for a VirtualMedia condition.
//
// nolint:govet // prefer readability over fieldalignment for this case
type VirtualMediaTaskParameters struct {
	// UploadImage when set will attempt to upload the image onto the BMC.
	// Required: false
	UploadImage bool `json:"upload_image"`

	// ImageURL is a URL accessible to the Disko controller and the BMC to download the image.
	// Required: true
	ImageURL string `json:"image_uri"`

	// MediaType indicates the kind of media being uploaded/mounted
	// Required: true
	MediaType VirtualMediaType `json:"media_type"`

	// Identifier for the Asset in the Asset store.
	// Reqruired: true
	AssetID uuid.UUID `json:"asset_id"`
}

func NewVirtualMediaTaskParameters(assetID uuid.UUID, imageURL string, mediaType VirtualMediaType, uploadImage bool) *VirtualMediaTaskParameters {
	return &VirtualMediaTaskParameters{
		AssetID:     assetID,
		ImageURL:    imageURL,
		MediaType:   mediaType,
		UploadImage: uploadImage,
	}
}
