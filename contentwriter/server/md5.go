package server

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// calculateMD5 calculates the MD5 checksum of a given file.
func calculateMD5(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	// Create a new MD5 hash
	hash := md5.New()

	// Copy file data into the hash
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// Get the MD5 sum and encode it as a hexadecimal string
	md5sum := hex.EncodeToString(hash.Sum(nil))
	return md5sum, nil
}
