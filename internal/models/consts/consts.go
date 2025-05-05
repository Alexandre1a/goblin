package consts

import (
	"os"
	"runtime"
)

// All consts
const (
	GoblinBaseDir  = ".goblin"
	BinDir         = "bin"
	ManifestName   = "sources.yaml"
	LockFilePath   = "goblin.lock"
	RawManifestURL = "https://raw.githubusercontent.com/Alexandre1a/goblin-remote/refs/heads/main/sources.yaml" // Hardcoded manifest path, change it as you like
	CurrentOS      = runtime.GOOS
	CurrentArch    = runtime.GOARCH
)

var HomeDir, err = os.UserHomeDir()
