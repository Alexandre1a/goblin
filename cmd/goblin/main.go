// Complete Rewrite
package main

import (
	"os"

	"github.com/alexandre1a/goblin/internal/utils/checks"
)

// All consts
const (
	GoblinBaseDir  = ".goblin"
	BinDir         = "bin"
	ManifestName   = "sources.yaml"
	LockFilePath   = "goblin.lock"
	RawManifestURL = "https://raw.githubusercontent.com/Alexandre1a/goblin-remote/refs/heads/main/sources.yaml" // Hardcoded manifest path, change it as you like
)

// Global variables
var homeDir, err = os.UserHomeDir()

func Init() {
	if checks.CheckConnectivity() != nil {
		isConnected := true
	} else {
		isConnected := false
	}
}
