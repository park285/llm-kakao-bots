// Package valkeyx 는 Redis/Valkey 클라이언트 공통 유틸리티를 제공한다.
// 키 생성, 연결, nil 체크 등의 헬퍼 함수들을 포함한다.
package valkeyx

import (
	"fmt"
	"strings"
)

// 키 생성 헬퍼 함수들

// BuildKey 는 prefix와 id를 결합하여 키를 생성한다.
// 형식: {prefix}:{id}
func BuildKey(prefix, id string) string {
	return fmt.Sprintf("%s:%s", prefix, strings.TrimSpace(id))
}

// BuildKey2 는 prefix와 두 개의 id를 결합하여 키를 생성한다.
// 형식: {prefix}:{id1}:{id2}
func BuildKey2(prefix, id1, id2 string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, strings.TrimSpace(id1), strings.TrimSpace(id2))
}

// BuildKey3 는 prefix와 세 개의 id를 결합하여 키를 생성한다.
// 형식: {prefix}:{id1}:{id2}:{id3}
func BuildKey3(prefix, id1, id2, id3 string) string {
	return fmt.Sprintf("%s:%s:%s:%s", prefix, strings.TrimSpace(id1), strings.TrimSpace(id2), strings.TrimSpace(id3))
}

// BuildKeySuffix 는 prefix, id, suffix를 결합하여 키를 생성한다.
// 형식: {prefix}:{id}:{suffix}
func BuildKeySuffix(prefix, id, suffix string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, strings.TrimSpace(id), suffix)
}
