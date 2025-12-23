package assets

import _ "embed" // 에셋 임베드용

// GameMessagesYAML 는 20문답 게임 메시지 YAML이다.
//
//go:embed messages/game-messages.yml
var GameMessagesYAML string

// LockAcquireReadLua 는 읽기 락 획득 Lua 스크립트다.
//
//go:embed lua/lock_acquire_read.lua
var LockAcquireReadLua string

// LockAcquireWriteLua 는 쓰기 락 획득 Lua 스크립트다.
//
//go:embed lua/lock_acquire_write.lua
var LockAcquireWriteLua string

// LockReleaseLua 는 락 해제 Lua 스크립트다.
//
//go:embed lua/lock_release.lua
var LockReleaseLua string

// LockReleaseReadLua 는 읽기 락 해제 Lua 스크립트다.
//
//go:embed lua/lock_release_read.lua
var LockReleaseReadLua string

// LockRenewReadLua 는 읽기 락 갱신 Lua 스크립트다.
//
//go:embed lua/lock_renew_read.lua
var LockRenewReadLua string

// LockRenewWriteLua 는 쓰기 락 갱신 Lua 스크립트다.
//
//go:embed lua/lock_renew_write.lua
var LockRenewWriteLua string
