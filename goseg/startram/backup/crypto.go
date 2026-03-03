package backup

import "groundseg/startram/backup/crypto"

// EncryptFile returns AES-GCM encrypted bytes for the given plaintext file.
func EncryptFile(filename string, keyString string) ([]byte, error) {
	return crypto.EncryptFile(filename, keyString)
}

// DecryptFile decrypts AES-GCM encrypted bytes using the same key format as EncryptFile.
func DecryptFile(file []byte, keyString string) ([]byte, error) {
	return crypto.DecryptFile(file, keyString)
}
