package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadDotenvIfPresent 는 동작을 수행한다.
func LoadDotenvIfPresent(paths ...string) error {
	if len(paths) == 0 {
		paths = []string{".env"}
	}

	for _, path := range paths {
		_, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("stat dotenv file failed path=%s: %w", path, err)
		}

		if err := godotenv.Load(path); err != nil {
			return fmt.Errorf("load dotenv file failed path=%s: %w", path, err)
		}
	}

	return nil
}
