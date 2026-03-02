package wireguardkeys

import (
	"encoding/base64"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func GenerateKeyPair() (privateKeyStr string, publicKeyStr string, err error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}
	publicKey := base64.StdEncoding.EncodeToString([]byte(privateKey.PublicKey().String() + "\n"))
	return privateKey.String(), publicKey, nil
}
