module github.com/Lexer747/acci-ping

go 1.23.0

toolchain go1.23.2

require (
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394
	golang.org/x/net v0.37.0
	golang.org/x/term v0.30.0
)

// Test dependencies
require (
	github.com/google/go-cmp v0.7.0
	gotest.tools/v3 v3.5.2
)

require golang.org/x/sys v0.31.0 // indirect
