package assets

import _ "embed" // 에셋 임베드용

// GameMessagesYAML: 터틀수프 게임 메시지 YAML입니다.
//
//go:embed messages/game-messages.yml
var GameMessagesYAML string

// LockReleaseLua: 분산 락 해제 Lua 스크립트입니다.
//
//go:embed lua/lock_release.lua
var LockReleaseLua string
