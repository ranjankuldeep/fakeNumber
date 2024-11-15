package lib

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

// GenerateTronAddress generates a new TRON address and private key
func GenerateTronAddress() (map[string]string, error) {
	// Generate a new private key using the secp256k1 curve
	privateKey, err := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("error generating private key: %v", err)
	}

	// Convert the private key to a hexadecimal string
	privateKeyBytes := privateKey.D.Bytes()
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// Derive the TRON address from the public key
	publicKey := privateKey.PublicKey
	tronAddress := address.PubkeyToAddress(publicKey)

	// Return the private key and TRON address
	return map[string]string{
		"privateKey": privateKeyHex,
		"address":    tronAddress.String(),
	}, nil
}
