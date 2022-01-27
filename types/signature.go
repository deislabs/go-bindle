package types

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"time"
)

// Cleartext format:
// Matt Butcher <matt.butcher@example.com>
// mybindle
// 0.1.0
// creator
// 1611960337
// ~
// e1706ab0a39ac88094b6d54a3f5cdba41fe5a901
// 098fa798779ac88094b6d54a3f5cdba41fe5a901
// 5b992e90b71d5fadab3cd3777230ef370df75f5b

// GenerateCreatorSignature generates a signature for the creator using the first 'author'
// in the Invoice, and then appends the new signature to the Invoice's 'Signature' list.
// Use keyring.GenerateSignatureKey to create a keypair
func (i *Invoice) GenerateCreatorSignature(sigKey *SignatureKey, privKey []byte) error {
	timestamp := time.Now()

	cleartext := i.generateCleartext("creator", timestamp)

	sig := ed25519.Sign(privKey, []byte(cleartext))

	pubKey, err := base64.StdEncoding.DecodeString(sigKey.Key)
	if err != nil {
		return err
	}

	signature := Signature{
		By:        i.Bindle.Authors[0],
		Signature: base64.StdEncoding.EncodeToString(sig),
		Key:       base64.StdEncoding.EncodeToString(pubKey),
		Role:      "creator",
		At:        timestamp.Unix(),
	}

	if i.Signature == nil {
		i.Signature = []Signature{}
	}

	i.Signature = append(i.Signature, signature)

	return nil
}

func (i *Invoice) generateCleartext(role string, timestamp time.Time) string {
	// metadata
	cleartextParts := []string{
		i.Bindle.Authors[0],
		i.Bindle.Name,
		i.Bindle.Version,
		role,
		"~",
	}

	// parcel SHAs
	for _, p := range i.Parcel {
		cleartextParts = append(cleartextParts, p.Label.SHA256)
	}

	return strings.Join(cleartextParts, "\n")
}
