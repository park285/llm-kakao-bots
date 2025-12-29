package session

import (
	"fmt"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// 싱글톤 encoder/decoder - goroutine-safe 재사용
var (
	zstdEncoder *zstd.Encoder
	zstdDecoder *zstd.Decoder
	initOnce    sync.Once
	errInit     error
)

// initZstd: zstd encoder/decoder 초기화
func initZstd() error {
	initOnce.Do(func() {
		var err error
		// SpeedDefault: 압축률/속도 균형 (Level 3)
		zstdEncoder, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
		if err != nil {
			errInit = fmt.Errorf("create zstd encoder: %w", err)
			return
		}
		zstdDecoder, err = zstd.NewReader(nil)
		if err != nil {
			errInit = fmt.Errorf("create zstd decoder: %w", err)
		}
	})
	return errInit
}

// compressZstd: 데이터를 Zstd로 압축
func compressZstd(src []byte) ([]byte, error) {
	if err := initZstd(); err != nil {
		return nil, err
	}
	// pre-allocate destination buffer (압축 후 크기는 보통 원본보다 작음)
	dst := make([]byte, 0, len(src))
	return zstdEncoder.EncodeAll(src, dst), nil
}

// decompressZstd: Zstd 압축 해제
func decompressZstd(src []byte) ([]byte, error) {
	if err := initZstd(); err != nil {
		return nil, err
	}
	decoded, err := zstdDecoder.DecodeAll(src, nil)
	if err != nil {
		return nil, fmt.Errorf("zstd decompress: %w", err)
	}
	return decoded, nil
}
