package main

// Version is set at build time via ldflags:
//
//	-ldflags "-X main.Version=1.2.0"
//
// Falls back to "dev" when not set (local builds).
var Version = "dev"
