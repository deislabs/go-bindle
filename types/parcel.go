package types

import (
	"crypto/sha256"
	"encoding/hex"
)

// NewParcel creates a new Parcel
func NewParcel(name, mediaType string, data []byte) Parcel {
	sha := sha256.New()
	sha.Write(data)

	fileSHA := hex.EncodeToString(sha.Sum(nil))

	label := Label{
		SHA256:    fileSHA,
		MediaType: mediaType,
		Name:      name,
		Size:      uint64(len(data)),
	}

	parcel := Parcel{
		Label: label,
	}

	return parcel
}
