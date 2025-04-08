package types

import "time"

// Create all types
type Artifact struct {
	OS   string `yaml:"os"`   // The operation system, ex: "linux", "windows"
	Arch string `yaml:"arch"` // The architecture of the package, ex: "arm64" or "amd64"
	File string `yaml:"file"` // The name of the file
}

type Package struct {
	Name      string     `yaml:"name"`      //Name of the package in the manifest
	Version   string     `yaml:"version"`   // Version (ex: "v2.4.1" ou "latest")
	BaseURL   string     `yaml:"base_url"`  // The common part in the artifact URL
	Artifacts []Artifact `yaml:"artifacts"` // List all available artifacts
}

type Manifest struct {
	Packages []Package `yaml:"packages"` // Represent the structure of the manifest
}

type LockFile struct {
	Packages []InstalledPackage `json:"packages"` // List installed packages
}

type InstalledPackage struct {
	Name         string    `json:"name"`          // Package name
	Version      string    `json:"version"`       // Installed version
	ResolvedFrom string    `json:"resolved_from"` // Version specified in the manifest
	InstallDate  time.Time `json:"install_date"`  // The date when the package was installed
	OS           string    `json:"os"`
	Arch         string    `json:"arch"`
	Path         string    `json:"path"` // Installed path
}

type UpdateResult struct {
	Name            string // Name of the package
	PerviousVersion string // The old version
	NewVersion      string // The new, installed version
	Status          string // Update status
	Message         string // Detailled message (in case of error)
}
