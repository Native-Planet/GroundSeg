package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	plainPath := filepath.Join(dir, "backup.bin")
	plaintext := []byte("hello encrypted backup payload")
	if err := os.WriteFile(plainPath, plaintext, 0o600); err != nil {
		t.Fatalf("write plaintext file: %v", err)
	}

	ciphertext, err := EncryptFile(plainPath, "private-key")
	if err != nil {
		t.Fatalf("EncryptFile returned error: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("expected ciphertext bytes")
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("expected ciphertext to differ from plaintext")
	}

	decrypted, err := DecryptFile(ciphertext, "private-key")
	if err != nil {
		t.Fatalf("DecryptFile returned error: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted payload mismatch: got %q want %q", string(decrypted), string(plaintext))
	}
}

func TestEncryptFileMissingPath(t *testing.T) {
	t.Parallel()

	if _, err := EncryptFile(filepath.Join(t.TempDir(), "missing.bin"), "private-key"); err == nil {
		t.Fatal("expected EncryptFile to fail for missing input path")
	}
}

func TestDecryptFileRejectsShortCiphertext(t *testing.T) {
	t.Parallel()

	if _, err := DecryptFile([]byte("tiny"), "private-key"); err == nil {
		t.Fatal("expected short ciphertext to fail decryption")
	}
}
