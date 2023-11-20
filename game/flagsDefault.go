//go:build !for_wasm

package game

import "runtime"

const SAVING_ENABLED = true

var POOL_SIZE int = runtime.NumCPU() * 2
