package pending

import _ "embed" // Lua 스크립트 임베드용

//go:embed lua/pending_enqueue.lua
var enqueueLua string

//go:embed lua/pending_dequeue.lua
var dequeueLua string
