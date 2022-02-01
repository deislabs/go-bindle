package keyring

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/deislabs/go-bindle/types"
	"github.com/pelletier/go-toml"
)

// GenerateSignatureKey generates a keypair for signing Bindle invoices
// The return types are the public key (wrapped in a SignatureKey), the private key, and any error
func GenerateSignatureKey(author, role string) (*types.SignatureKey, []byte, error) {
	if exists, val := types.ValidRoles[role]; !exists || !val {
		return nil, nil, types.ErrInvalidRole
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	pubString := base64.StdEncoding.EncodeToString(pub)

	labelSig := ed25519.Sign(priv, []byte(author))
	sigString := base64.StdEncoding.EncodeToString(labelSig)

	sigKey := &types.SignatureKey{
		Label:          author,
		Roles:          []string{role},
		Key:            pubString,
		LabelSignature: sigString,
	}

	return sigKey, priv, nil
}

// Localkeyring returns the keyring stored on your local machine
func LocalKeyring() (*types.Keyring, error) {
	filepath := keyringFilepath()

	keyringBytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	keyring := &types.Keyring{}
	if err := toml.Unmarshal(keyringBytes, keyring); err != nil {
		return nil, err
	}

	return keyring, nil
}

// AddLocalKey adds a new key to your local keyring file
func AddLocalKey(key *types.SignatureKey) error {
	keyring, err := LocalKeyring()
	if err != nil {
		// nothing to be done, create a new one

		keyring = &types.Keyring{
			Version: "1.0.0",
			Key:     []types.SignatureKey{},
		}
	}

	keyring.Key = append(keyring.Key, *key)

	keyringBytes, err := toml.Marshal(keyring)
	if err != nil {
		return err
	}

	// overwrite the file if it exists
	if err := os.WriteFile(keyringFilepath(), keyringBytes, 0600); err != nil {
		return err
	}

	return nil
}

// WritePrivKey writes a private key (encoded to base64) to the provided filepath
func WritePrivKey(privKey []byte, filepath string) error {
	keyString := base64.StdEncoding.EncodeToString(privKey)

	if err := os.WriteFile(filepath, []byte(keyString), 0600); err != nil {
		return err
	}

	return nil
}

// ReadPrivKey reads a private key from a file and returns its raw bytes
func ReadPrivKey(filepath string) ([]byte, error) {
	keyBytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(string(keyBytes))
}

func keyringFilepath() string {
	base := filepath.Join("$HOME", ".bindle")

	if home, err := os.UserHomeDir(); err == nil {
		base = filepath.Join(home, ".bindle")
	}

	if config, err := os.UserConfigDir(); err == nil {
		base = filepath.Join(config, "bindle")
	}

	return filepath.Join(base, "keyring.toml")
}
