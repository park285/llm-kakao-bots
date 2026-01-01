package assets

import _ "embed" // 에셋 임베드용

// GameMessagesYAML: 20문답 게임 메시지 YAML입니다.
//
//go:embed messages/game-messages.yml
var GameMessagesYAML string

// LockAcquireLua: 락 획득 Lua 스크립트입니다.
//
//go:embed lua/lock_acquire.lua
var LockAcquireLua string

// LockReleaseLua: 락 해제 Lua 스크립트입니다.
//
//go:embed lua/lock_release.lua
var LockReleaseLua string

// LockRenewWriteLua: 쓰기 락 갱신 Lua 스크립트입니다.
//
//go:embed lua/lock_renew_write.lua
var LockRenewWriteLua string

// GuessRateLimitLua: 정답 시도 Rate Limit 체크 및 설정 Lua 스크립트입니다.
//
//go:embed lua/guess_rate_limit.lua
var GuessRateLimitLua string
