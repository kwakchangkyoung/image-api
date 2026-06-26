package util

import (
	"crypto/md5"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func GenerateUUIDHex() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func GenerateFilename(ext string) string {
	return GenerateUUIDHex() + "." + ext
}

func SafePath(base string, subdir string, filename string) (string, bool) {
	if strings.Contains(subdir, "..") || strings.Contains(filename, "..") {
		return "", false
	}

	if strings.ContainsAny(subdir, `/\\`) || strings.ContainsAny(filename, `/\\`) {
		return "", false
	}

	baseClean := filepath.Clean(base)
	fullPath := filepath.Join(baseClean, subdir, filename)

	relPath, err := filepath.Rel(baseClean, fullPath)
	if err != nil {
		return "", false
	}

	if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		return "", false
	}

	return fullPath, true
}

func MD5Hex(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}
