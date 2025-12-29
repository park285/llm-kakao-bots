package session

import (
	"bytes"
	"strings"
	"testing"
)

// TestCompressDecompressRoundTrip: 압축/해제 라운드트립 검증
func TestCompressDecompressRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "짧은 텍스트",
			data: `{"role":"user","content":"안녕하세요"}`,
		},
		{
			name: "긴 대화 히스토리",
			data: `{"role":"model","content":"` + strings.Repeat("이것은 테스트 메시지입니다. ", 100) + `"}`,
		},
		{
			name: "빈 문자열",
			data: "",
		},
		{
			name: "특수 문자 포함",
			data: `{"role":"user","content":"<xml>테스트</xml>\n\t\"escape\""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := []byte(tt.data)

			// 압축
			compressed, err := compressZstd(original)
			if err != nil {
				t.Fatalf("compressZstd failed: %v", err)
			}

			// 해제
			decompressed, err := decompressZstd(compressed)
			if err != nil {
				t.Fatalf("decompressZstd failed: %v", err)
			}

			// 검증
			if !bytes.Equal(original, decompressed) {
				t.Errorf("round-trip mismatch:\noriginal: %q\ndecompressed: %q", original, decompressed)
			}
		})
	}
}

// TestCompressionRatio: 압축률 검증 (자연어 텍스트는 50% 이상 절감 기대)
func TestCompressionRatio(t *testing.T) {
	// 실제 히스토리 엔트리와 유사한 반복 텍스트
	original := []byte(`{"role":"model","content":"` + strings.Repeat("사용자의 질문에 대한 상세한 답변입니다. 이 게임에서는 예/아니오로만 대답할 수 있습니다. ", 50) + `"}`)

	compressed, err := compressZstd(original)
	if err != nil {
		t.Fatalf("compressZstd failed: %v", err)
	}

	ratio := float64(len(compressed)) / float64(len(original)) * 100
	t.Logf("원본 크기: %d bytes", len(original))
	t.Logf("압축 크기: %d bytes", len(compressed))
	t.Logf("압축률: %.1f%% (절감: %.1f%%)", ratio, 100-ratio)

	// 자연어 텍스트는 최소 30% 이상 절감 기대
	if ratio > 70 {
		t.Errorf("압축률이 기대치보다 낮음: %.1f%% (70%% 이하 기대)", ratio)
	}
}

// TestDecompressInvalidData: 잘못된 데이터 해제 시 에러 반환 검증
func TestDecompressInvalidData(t *testing.T) {
	invalidData := []byte("this is not zstd compressed data")

	_, err := decompressZstd(invalidData)
	if err == nil {
		t.Error("decompressZstd should return error for invalid data")
	}
}

// BenchmarkCompressZstd: 압축 성능 벤치마크
func BenchmarkCompressZstd(b *testing.B) {
	data := []byte(`{"role":"model","content":"` + strings.Repeat("벤치마크 테스트 데이터입니다. ", 20) + `"}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = compressZstd(data)
	}
}

// BenchmarkDecompressZstd: 해제 성능 벤치마크
func BenchmarkDecompressZstd(b *testing.B) {
	data := []byte(`{"role":"model","content":"` + strings.Repeat("벤치마크 테스트 데이터입니다. ", 20) + `"}`)
	compressed, _ := compressZstd(data)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = decompressZstd(compressed)
	}
}
