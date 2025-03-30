// main.go
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2" // Pour parser le YAML (cf. https://github.com/go-yaml/yaml) :contentReference[oaicite:0]{index=0}
)

// Artifact représente un artefact de release pour un package.
type Artifact struct {
	OS   string `yaml:"os"`   // Système d'exploitation, ex: "linux", "darwin"
	Arch string `yaml:"arch"` // Architecture, ex: "amd64", "arm64"
	File string `yaml:"file"` // Nom du fichier de l'artefact
}

// Package représente un package tel que défini dans le manifest.
type Package struct {
	Name      string     `yaml:"name"`      // Nom du package
	Version   string     `yaml:"version"`   // Version (ex: "v2.4.1" ou "latest")
	BaseURL   string     `yaml:"base_url"`  // Partie commune de l'URL des artefacts
	Artifacts []Artifact `yaml:"artifacts"` // Liste des artefacts disponibles
}

// Manifest représente la structure du fichier sources.yaml.
type Manifest struct {
	Packages []Package `yaml:"packages"`
}

// LoadManifest lit et parse le fichier manifest depuis le chemin spécifié.
func LoadManifest(path string) (*Manifest, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err = yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func DownloadFile(url string, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// InstallPackage télécharge l'artefact correspondant à l'OS et à l'architecture actuels.
func InstallPackage(pkg Package, forceBuild bool) error {
	if forceBuild {
		fmt.Println("Option --build spécifiée, mais ignorée (les artefacts prébuild sont utilisés).")
	}

	currentOS := runtime.GOOS     // ex: "linux", "darwin"
	currentArch := runtime.GOARCH // ex: "amd64", "arm64"
	var selected *Artifact

	for _, art := range pkg.Artifacts {
		if strings.EqualFold(art.OS, currentOS) && strings.EqualFold(art.Arch, currentArch) {
			selected = &art
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("aucun artefact trouvé pour OS=%s, Arch=%s", currentOS, currentArch)
	}

	// Concaténer la base_url et le nom de fichier pour obtenir l'URL complète.
	fullURL := pkg.BaseURL + selected.File

	fmt.Printf("Téléchargement du package %s (version %s) pour %s/%s...\n", pkg.Name, pkg.Version, currentOS, currentArch)
	fmt.Printf("Téléchargement depuis : %s\n", fullURL)

	DownloadFile(fullURL, pkg.Name)
	//cmd := exec.Command("wget", fullURL)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	//if err := cmd.Run(); err != nil {
	//return fmt.Errorf("erreur lors du téléchargement : %v", err)
	//}
	fmt.Printf("Le package %s a été téléchargé avec succès.\n", pkg.Name)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: goblin <command> [args]")
		fmt.Println("Commandes disponibles: install <package> [--build]")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "install":
		installCmd := flag.NewFlagSet("install", flag.ExitOnError)
		forceBuild := installCmd.Bool("build", false, "Force la compilation (build) des packages (option ignorée pour les artefacts prébuild)")
		installCmd.Parse(os.Args[2:])
		args := installCmd.Args()
		if len(args) < 1 {
			fmt.Println("Usage: goblin install <package> [--build]")
			os.Exit(1)
		}
		packageName := args[0]

		manifest, err := LoadManifest("sources.yaml")
		if err != nil {
			log.Fatalf("Erreur lors du chargement du manifest : %v", err)
		}

		found := false
		for _, pkg := range manifest.Packages {
			if strings.EqualFold(pkg.Name, packageName) {
				found = true
				if err := InstallPackage(pkg, *forceBuild); err != nil {
					log.Printf("Erreur lors de l'installation du package %s : %v", pkg.Name, err)
				}
				break
			}
		}
		if !found {
			fmt.Printf("Aucun package correspondant au nom '%s' n'a été trouvé dans le manifest.\n", packageName)
			os.Exit(1)
		}
	default:
		fmt.Printf("Commande inconnue : %s\n", command)
		os.Exit(1)
	}
}
