// Package wasm implements functions that can be imported by WASM programs.
//
// When called from .wast files, implementation of functions in this
// package is ignored and WASM stdio package is used for the implementation.
// When run as straight Go code, the implementation emulates what the
// WASM stdio package provides.

package wasm
