package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadDotenvIfPresent: 지정된 경로들(기본값 .env)에 파일이 존재하면 환경 변수로 로드한다.
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
