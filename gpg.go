package main

import (
	"bytes"
	"crypto"
	"fmt"
	"strings"

	"golang.org/x/crypto/openpgp"        //nolint:staticcheck
	"golang.org/x/crypto/openpgp/armor"  //nolint:staticcheck
	"golang.org/x/crypto/openpgp/packet" //nolint:staticcheck
)

// generateGPGKey creates a new RSA-4096 OpenPGP key and returns the key ID,
// ASCII-armored public key, and ASCII-armored private key.
func generateGPGKey() (keyID, pubArmor, privArmor string, err error) {
	cfg := &packet.Config{
		DefaultHash:   crypto.SHA256,
		DefaultCipher: packet.CipherAES256,
		RSABits:       4096,
	}
	entity, err := openpgp.NewEntity("tofu-provider registry", "", "", cfg)
	if err != nil {
		return "", "", "", fmt.Errorf("generate GPG key: %w", err)
	}

	keyID = fmt.Sprintf("%016X", entity.PrimaryKey.KeyId)

	var pubBuf bytes.Buffer
	w, err := armor.Encode(&pubBuf, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		return "", "", "", err
	}
	if err := entity.Serialize(w); err != nil {
		return "", "", "", err
	}
	w.Close()
	pubArmor = pubBuf.String()

	var privBuf bytes.Buffer
	w2, err := armor.Encode(&privBuf, "PGP PRIVATE KEY BLOCK", nil)
	if err != nil {
		return "", "", "", err
	}
	if err := entity.SerializePrivate(w2, nil); err != nil {
		return "", "", "", err
	}
	w2.Close()
	privArmor = privBuf.String()

	return keyID, pubArmor, privArmor, nil
}

// signDetached produces a binary detached OpenPGP signature over data using
// the ASCII-armored private key stored in privArmor.
func signDetached(privArmor string, data []byte) ([]byte, error) {
	block, err := armor.Decode(strings.NewReader(privArmor))
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	entities, err := openpgp.ReadKeyRing(block.Body)
	if err != nil {
		return nil, fmt.Errorf("read key ring: %w", err)
	}
	var sig bytes.Buffer
	if err := openpgp.DetachSign(&sig, entities[0], bytes.NewReader(data), nil); err != nil {
		return nil, fmt.Errorf("detach sign: %w", err)
	}
	return sig.Bytes(), nil
}
