package install

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alexandre1a/goblin/internal/models/consts"
	"github.com/alexandre1a/goblin/internal/models/types"
	"github.com/alexandre1a/goblin/internal/utils/common"
)

func InstallPackage(pkg types.Package) error {
	// Todo: parse yaml file
	var selected types.Artifact
	for _, art := range pkg.Artifacts {
		if strings.EqualFold(art.OS, consts.CurrentOS) && strings.EqualFold(art.Arch, consts.CurrentArch) {
			selected = &art
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("No Artifact found for this system")
	}

	// Create the full URL
	fullURL := pkg.BaseURL + selected.File
	binPath := filepath.Join(consts.HomeDir, consts.GoblinBaseDir, consts.BinDir)
	fmt.Printf("Started downloading package %s (version %s) for %s/%s...\n", pkg.Name, pkg.Version, consts.CurrentOS, consts.CurrentArch)
	fmt.Printf("Downloading from : %s\n", fullURL)

	if common.DownloadFile(fullURL, binPath, pkg.Name) != nil {
		return fmt.Errorf("Error while downloading")
	}

	fmt.Printf("The package %s has been downloaded with sucess ! \n", pkg.Name)
	// Todo: Use lock file
	return nil
}
