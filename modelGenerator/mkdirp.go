package modelGenerator

import (
	"io/fs"
	"os"
)

// MkdirP makes a directory like bash mkdir -p
func MkdirP(path string, perm fs.FileMode) error {
	// Check if the directory already exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// If not, create the directory and all parent directories
		err := os.MkdirAll(path, perm)
		if err != nil {
			return err
		}
	}
	return nil
}
