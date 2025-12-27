package mq

import _ "embed"

//go:embed lua/process_with_idempotency.lua
var processWithIdempotencyLua string

//go:embed lua/complete_processing.lua
var completeProcessingLua string
