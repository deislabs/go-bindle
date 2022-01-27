package keyring

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/deislabs/go-bindle/types"
	"github.com/pelletier/go-toml"
)

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
		return err
	}

	keyring.Key = append(keyring.Key, *key)

	keyringBytes, err := toml.Marshal(keyring)
	if err != nil {
		return err
	}

	// overwrite the file
	if err := os.WriteFile(keyringFilepath(), keyringBytes, fs.FileMode(os.O_RDWR)); err != nil {
		return err
	}

	return nil
}

func keyringFilepath() string {
	base := ""

	xdg, exists := os.LookupEnv("XDG_CONFIG")
	if exists {
		base = xdg
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			base = "$HOME"
		} else {
			base = home
		}
	}

	return filepath.Join(base, "keyring.toml")
}
