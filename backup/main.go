package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	// To check network connectivity
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

// InstalledPackage représente les informations d'un package installé.
type InstalledPackage struct {
	Name         string    `json:"name"`          // Nom du package
	Version      string    `json:"version"`       // Version installée
	ResolvedFrom string    `json:"resolved_from"` // Version spécifiée dans le manifest (peut être "latest")
	InstallDate  time.Time `json:"install_date"`  // Date d'installation
	OS           string    `json:"os"`            // Système d'exploitation
	Arch         string    `json:"arch"`          // Architecture
	Path         string    `json:"path"`          // Chemin d'installation
}

// LockFile représente la structure du fichier goblin.lock.
type LockFile struct {
	Packages []InstalledPackage `json:"packages"` // Liste des packages installés
}

// UpdateResult représente le résultat d'une opération de mise à jour
type UpdateResult struct {
	Name            string // Nom du package
	PreviousVersion string // Version précédente
	NewVersion      string // Nouvelle version
	Status          string // Status de la mise à jour (success, error, skipped)
	Message         string // Message détaillé (notamment en cas d'erreur)
}

// Constantes pour le fichier de verrouillage
const (
	GoblinBaseDir   = ".goblin"
	BinDirName      = "bin"
	ManifestDirName = "manifest"
	ManifestName    = "sources.yaml"
	LockFilePath    = "goblin.lock"
	RawManifestURL  = "https://raw.githubusercontent.com/Alexandre1a/goblin-remote/refs/heads/main/sources.yml"
)

func GetGoblinDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("erreur répertoire utilisateur: %v", err)
	}
	return filepath.Join(homeDir, GoblinBaseDir), nil
}

// EnsureManifest vérifie et télécharge le manifest si nécessaire
func EnsureManifest() error {
	goblinDir, err := GetGoblinDir()
	if err != nil {
		return err
	}

	manifestDir := filepath.Join(goblinDir, ManifestDirName)
	manifestPath := filepath.Join(manifestDir, ManifestName)
	altManifestPath := filepath.Join(manifestDir, "sources.yml")

	// Vérifier si un manifest existe déjà
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		if _, err := os.Stat(altManifestPath); os.IsNotExist(err) {
			// Créer le répertoire
			if err := os.MkdirAll(manifestDir, 0755); err != nil {
				return fmt.Errorf("erreur création répertoire: %v", err)
			}

			// Télécharger le manifest
			resp, err := http.Get(RawManifestURL)
			if err != nil {
				return fmt.Errorf("erreur connexion GitHub: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("erreur HTTP %d", resp.StatusCode)
			}

			file, err := os.Create(manifestPath)
			if err != nil {
				return fmt.Errorf("erreur création fichier: %v", err)
			}
			defer file.Close()

			if _, err := io.Copy(file, resp.Body); err != nil {
				return fmt.Errorf("erreur écriture fichier: %v", err)
			}
		}
	}
	return nil
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

// LoadLockFile charge le fichier de verrouillage s'il existe.
func LoadLockFile() (*LockFile, error) {
	goblinDir, err := GetGoblinDir()
	if err != nil {
		return nil, err
	}

	lockFilePath := filepath.Join(goblinDir, LockFilePath)

	// Vérifier si le fichier existe
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		// Créer un nouveau fichier de verrouillage vide
		return &LockFile{Packages: []InstalledPackage{}}, nil
	}

	// Lire le fichier existant
	data, err := ioutil.ReadFile(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture du fichier lock : %v", err)
	}

	var lockFile LockFile
	if err = json.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing du fichier lock : %v", err)
	}

	return &lockFile, nil
}

// SaveLockFile enregistre le fichier de verrouillage.
func SaveLockFile(lockFile *LockFile) error {
	goblinDir, err := GetGoblinDir()
	if err != nil {
		return err
	}

	lockFilePath := filepath.Join(goblinDir, LockFilePath)

	data, err := json.MarshalIndent(lockFile, "", "  ")
	if err != nil {
		return fmt.Errorf("erreur lors de la sérialisation du fichier lock : %v", err)
	}

	if err = ioutil.WriteFile(lockFilePath, data, 0644); err != nil {
		return fmt.Errorf("erreur lors de l'écriture du fichier lock : %v", err)
	}

	return nil
}

// CompareVersions compare deux versions sémantiques et retourne:
// -1 si v1 < v2
//
//	0 si v1 = v2
//	1 si v1 > v2
func CompareVersions(v1, v2 string) int {
	// Si l'une des versions est "unknown", on considère qu'une mise à jour est nécessaire
	if v1 == "unknown" {
		return -1
	}
	if v2 == "unknown" {
		return 1
	}

	// Normaliser les versions en supprimant le 'v' préfixe s'il existe
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Diviser en composants [majeur, mineur, patch]
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Assurer que nous avons 3 composants pour chaque version en ajoutant des "0" si nécessaire
	for len(parts1) < 3 {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < 3 {
		parts2 = append(parts2, "0")
	}

	// Comparer chaque composant
	for i := 0; i < 3; i++ {
		// Convertir en entiers (simplification: nous ignorons les erreurs de parsing)
		var num1, num2 int
		fmt.Sscanf(parts1[i], "%d", &num1)
		fmt.Sscanf(parts2[i], "%d", &num2)

		if num1 < num2 {
			return -1
		} else if num1 > num2 {
			return 1
		}
	}

	// Si tous les composants sont égaux
	return 0
}

// ExtractVersionFromFilename essaie d'extraire la version à partir du nom de fichier
func ExtractVersionFromFilename(filename string, pkgName string) string {
	// Supprime l'extension
	basename := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Supprime le nom du package s'il est présent
	versionPart := strings.Replace(basename, pkgName+"-", "", 1)
	versionPart = strings.Replace(versionPart, pkgName, "", 1)

	// Si le résultat commence par "v", c'est probablement une version
	if strings.HasPrefix(versionPart, "v") || strings.Contains(versionPart, ".") {
		return versionPart
	}

	// Sinon, retourner "unknown"
	return "unknown"
}

// GetActualVersion obtient la version réelle d'un package à partir de l'en-tête de réponse HTTP
// ou du nom de fichier si possible.
func GetActualVersion(resp *http.Response, filename string, pkgName string, manifestVersion string) string {
	// Essayer d'abord d'obtenir la version à partir des en-têtes HTTP
	if version := resp.Header.Get("X-Version"); version != "" {
		return version
	}

	// Essayer d'extraire la version du nom de fichier
	extractedVersion := ExtractVersionFromFilename(filename, pkgName)
	if extractedVersion != "unknown" {
		return extractedVersion
	}

	// Si on ne peut pas déterminer la version réelle
	return manifestVersion
}



func UpdateManifest() error {
	if err := CheckConnectivity(); err != nil {
		return fmt.Errorf("pas de connexion Internet : %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("erreur répertoire utilisateur: %v", err)
	}

	manifestPath := filepath.Join(homeDir, ManifestDir, ManifestName)

	// Télécharger le nouveau manifest
	resp, err := http.Get(RawManifestURL)
	if err != nil {
		return fmt.Errorf("erreur de connexion à GitHub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erreur HTTP %d", resp.StatusCode)
	}

	newData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erreur de lecture: %v", err)
	}

	// Valider le format YAML
	var testManifest Manifest
	if err := yaml.Unmarshal(newData, &testManifest); err != nil {
		return fmt.Errorf("manifest invalide: %v", err)
	}

	// Écraser le fichier existant
	if err := ioutil.WriteFile(manifestPath, newData, 0644); err != nil {
		return fmt.Errorf("erreur d'écriture: %v", err)
	}

	// Supprimer l'ancienne version .yml si existante
	ymlPath := filepath.Join(homeDir, ManifestDir, "sources.yml")
	if _, err := os.Stat(ymlPath); err == nil {
		os.Remove(ymlPath)
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

	// Télécharger le fichier et obtenir la version réelle
	actualVersion, err := DownloadFile(fullURL, pkg.Name, pkg.Name, pkg.Version)
	if err != nil {
		return fmt.Errorf("erreur lors du téléchargement : %v", err)
	}

	fmt.Printf("Le package %s a été téléchargé avec succès.\n", pkg.Name)

	// Effectuer les opérations post-installation
	binPath, err := postInstall(pkg.Name)
	if err != nil {
		return fmt.Errorf("erreur lors de la post-installation : %v", err)
	}

	// Charger le fichier de verrouillage
	lockFile, err := LoadLockFile()
	if err != nil {
		return fmt.Errorf("erreur lors du chargement du fichier lock : %v", err)
	}

	// Créer une entrée pour le package installé
	installedPkg := InstalledPackage{
		Name:         pkg.Name,
		Version:      actualVersion,
		ResolvedFrom: pkg.Version,
		InstallDate:  time.Now(),
		OS:           currentOS,
		Arch:         currentArch,
		Path:         binPath,
	}

	// Vérifier si le package existe déjà dans le fichier lock
	found := false
	for i, p := range lockFile.Packages {
		if p.Name == pkg.Name {
			// Mettre à jour l'entrée existante
			lockFile.Packages[i] = installedPkg
			found = true
			break
		}
	}

	// Si le package n'existe pas encore, l'ajouter
	if !found {
		lockFile.Packages = append(lockFile.Packages, installedPkg)
	}

	// Enregistrer le fichier de verrouillage
	if err := SaveLockFile(lockFile); err != nil {
		return fmt.Errorf("erreur lors de l'enregistrement du fichier lock : %v", err)
	}

	fmt.Printf("Le package %s (version %s) a été enregistré dans le fichier lock.\n", pkg.Name, actualVersion)

	return nil
}

func UninstallPackage(pkgName string) error {
	lockFile, err := LoadLockFile()
	if err != nil {
		return fmt.Errorf("erreur lors du chargement du fichier lock : %v", err)
	}

	found := false
	for i, pkg := range lockFile.Packages {
		if strings.EqualFold(pkg.Name, pkgName) {
			// Supprimer le binaire
			if err := os.Remove(pkg.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("erreur lors de la suppression du binaire : %v", err)
			}

			// Supprimer du lock file
			lockFile.Packages = append(lockFile.Packages[:i], lockFile.Packages[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("package '%s' non installé", pkgName)
	}

	if err := SaveLockFile(lockFile); err != nil {
		return fmt.Errorf("erreur lors de la sauvegarde du lock file : %v", err)
	}

	return nil
}

// UpdatePackage met à jour un package spécifique si nécessaire
func UpdatePackage(pkgName string, manifest *Manifest, force bool) (*UpdateResult, error) {
	// Trouver le package dans le manifest
	var manifestPkg *Package
	for i, p := range manifest.Packages {
		if strings.EqualFold(p.Name, pkgName) {
			manifestPkg = &manifest.Packages[i]
			break
		}
	}

	if manifestPkg == nil {
		return &UpdateResult{
			Name:    pkgName,
			Status:  "error",
			Message: "Package non trouvé dans le manifest",
		}, fmt.Errorf("package '%s' non trouvé dans le manifest", pkgName)
	}

	// Charger le fichier de verrouillage
	lockFile, err := LoadLockFile()
	if err != nil {
		return &UpdateResult{
			Name:    pkgName,
			Status:  "error",
			Message: fmt.Sprintf("Erreur lors du chargement du fichier lock: %v", err),
		}, err
	}

	// Vérifier si le package est installé
	var installedPkg *InstalledPackage
	for i, p := range lockFile.Packages {
		if strings.EqualFold(p.Name, pkgName) {
			installedPkg = &lockFile.Packages[i]
			break
		}
	}

	if installedPkg == nil {
		// Si le package n'est pas installé, nous l'installons simplement
		fmt.Printf("Le package %s n'est pas installé. Installation...\n", pkgName)
		if err := InstallPackage(*manifestPkg, false); err != nil {
			return &UpdateResult{
				Name:    pkgName,
				Status:  "error",
				Message: fmt.Sprintf("Erreur lors de l'installation: %v", err),
			}, err
		}

		// Relire le fichier lock pour obtenir la version installée
		updatedLockFile, _ := LoadLockFile()
		for _, p := range updatedLockFile.Packages {
			if strings.EqualFold(p.Name, pkgName) {
				return &UpdateResult{
					Name:       pkgName,
					NewVersion: p.Version,
					Status:     "success",
					Message:    "Installé avec succès",
				}, nil
			}
		}

		return &UpdateResult{
			Name:    pkgName,
			Status:  "success",
			Message: "Installé avec succès",
		}, nil
	}

	// Si le package est déjà installé, vérifier si une mise à jour est nécessaire
	fmt.Printf("Package %s installé (version actuelle: %s)\n", pkgName, installedPkg.Version)

	// Si la version dans le manifest est "latest" ou différente de la version installée,
	// ou si l'option force est activée, nous procédons à la mise à jour
	if manifestPkg.Version == "latest" || force ||
		(manifestPkg.Version != installedPkg.ResolvedFrom &&
			CompareVersions(installedPkg.Version, manifestPkg.Version) < 0) {

		previousVersion := installedPkg.Version
		fmt.Printf("Mise à jour du package %s...\n", pkgName)

		// Supprimer le fichier binaire existant si présent
		if _, err := os.Stat(installedPkg.Path); err == nil {
			if err := os.Remove(installedPkg.Path); err != nil {
				return &UpdateResult{
					Name:            pkgName,
					PreviousVersion: previousVersion,
					Status:          "error",
					Message:         fmt.Sprintf("Erreur lors de la suppression de l'ancien binaire: %v", err),
				}, err
			}
		}

		// Installer la nouvelle version
		if err := InstallPackage(*manifestPkg, false); err != nil {
			return &UpdateResult{
				Name:            pkgName,
				PreviousVersion: previousVersion,
				Status:          "error",
				Message:         fmt.Sprintf("Erreur lors de la mise à jour: %v", err),
			}, err
		}

		// Relire le fichier lock pour obtenir la nouvelle version
		updatedLockFile, _ := LoadLockFile()
		for _, p := range updatedLockFile.Packages {
			if strings.EqualFold(p.Name, pkgName) {
				return &UpdateResult{
					Name:            pkgName,
					PreviousVersion: previousVersion,
					NewVersion:      p.Version,
					Status:          "success",
					Message:         "Mise à jour réussie",
				}, nil
			}
		}
	} else {
		// Aucune mise à jour nécessaire
		return &UpdateResult{
			Name:            pkgName,
			PreviousVersion: installedPkg.Version,
			NewVersion:      installedPkg.Version,
			Status:          "skipped",
			Message:         "Déjà à jour",
		}, nil
	}

	return &UpdateResult{
		Name:    pkgName,
		Status:  "unknown",
		Message: "État inconnu",
	}, nil
}

func SyncPackages(manifest *Manifest) ([]*UpdateResult, error) {
	lockFile, err := LoadLockFile()
	if err != nil {
		return nil, fmt.Errorf("erreur lors du chargement du fichier lock : %v", err)
	}

	var results []*UpdateResult

	for _, installedPkg := range lockFile.Packages {
		if _, err := os.Stat(installedPkg.Path); os.IsNotExist(err) {
			fmt.Printf("Binaire manquant pour %s, réinstallation...\n", installedPkg.Name)

			// Trouver le package dans le manifest
			var pkg *Package
			for i, mpkg := range manifest.Packages {
				if mpkg.Name == installedPkg.Name {
					pkg = &manifest.Packages[i]
					break
				}
			}

			if pkg == nil {
				results = append(results, &UpdateResult{
					Name:    installedPkg.Name,
					Status:  "error",
					Message: "Package non trouvé dans le manifest",
				})
				continue
			}

			// Réinstaller le package
			result, err := UpdatePackage(pkg.Name, manifest, true)
			if err != nil {
				results = append(results, result)
				continue
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// UpdateAllPackages met à jour tous les packages installés
func UpdateAllPackages(manifest *Manifest, force bool) ([]*UpdateResult, error) {
	// Charger le fichier de verrouillage
	lockFile, err := LoadLockFile()
	if err != nil {
		return nil, fmt.Errorf("erreur lors du chargement du fichier lock : %v", err)
	}

	if len(lockFile.Packages) == 0 {
		fmt.Println("Aucun package installé.")
		return []*UpdateResult{}, nil
	}

	results := make([]*UpdateResult, 0, len(lockFile.Packages))

	for _, installedPkg := range lockFile.Packages {
		fmt.Printf("Vérification des mises à jour pour %s...\n", installedPkg.Name)
		result, _ := UpdatePackage(installedPkg.Name, manifest, force)
		results = append(results, result)
	}

	return results, nil
}

func postInstall(pkg string) (string, error) {
	newpath := filepath.Join(".", "bin")
	err := os.MkdirAll(newpath, os.ModePerm)
	binDir := "./bin"
	basePath := pkg
	finalPath := binDir + "/" + pkg
	e := os.Rename(basePath, finalPath)
	if e != nil {
		log.Fatal(e)
	}
	err = os.Chmod(finalPath, 0700)
	if err != nil {
		log.Fatal(err)
	}
	// Renvoyer le chemin absolu
	absPath, err := filepath.Abs(finalPath)
	if err != nil {
		return finalPath, nil // En cas d'erreur, renvoyer le chemin relatif
	}
	return absPath, nil
}

// ListPackages affiche la liste des packages installés.
func ListPackages() error {
	lockFile, err := LoadLockFile()
	if err != nil {
		return fmt.Errorf("erreur lors du chargement du fichier lock : %v", err)
	}

	if len(lockFile.Packages) == 0 {
		fmt.Println("Aucun package installé.")
		return nil
	}

	fmt.Println("Packages installés :")
	fmt.Println("--------------------")
	for _, pkg := range lockFile.Packages {
		fmt.Printf("Nom: %s\n", pkg.Name)
		fmt.Printf("  Version: %s (depuis '%s')\n", pkg.Version, pkg.ResolvedFrom)
		fmt.Printf("  Date d'installation: %s\n", pkg.InstallDate.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Plateforme: %s/%s\n", pkg.OS, pkg.Arch)
		fmt.Printf("  Chemin: %s\n", pkg.Path)
		fmt.Println("--------------------")
	}

	return nil
}

func Help() {
	fmt.Println("Usage: goblin <command> [args]")
	fmt.Println("Commandes disponibles:")
	fmt.Println("  install <package> [--build]  - Installe un package")
	fmt.Println("  remove <package>             - Désinstalle un package")
	fmt.Println("  update [package] [--force]   - Met à jour un package ou tous les packages")
	fmt.Println("  sync                         - Synchronise les binaires manquants")
	fmt.Println("  list                         - Liste les packages installés")
}

func main() {
	if len(os.Args) < 2 {
		Help()
		os.Exit(1)
	}

	if err := EnsureManifest(); err != nil {
		log.Fatalf("Erreur initialisation: %v", err)
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
		homeDir, err := os.UserHomeDir()
		manifestDir := filepath.Join(homeDir, ".config", "goblin")
		manifest, err := LoadManifest(manifestDir + "/sources.yaml")
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

	case "update":
		fmt.Println("Vérification des mises à jour du manifest...")
		if err := UpdateManifest(); err != nil {
			log.Printf("⚠️ Impossible de mettre à jour le manifest : %v", err)
			log.Println("Utilisation de la version locale existante...")
		}

		// Charger le nouveau manifest
		homeDir, err := os.UserHomeDir()
		manifestDir := filepath.Join(homeDir, ".config", "goblin")
		manifest, err := LoadManifest(manifestDir + "/sources.yaml")
		if err != nil {
			log.Fatalf("Erreur lors du chargement du manifest : %v", err)
		}
		updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
		forceUpdate := updateCmd.Bool("force", false, "Force la mise à jour même si la version est identique")
		updateCmd.Parse(os.Args[2:])
		args := updateCmd.Args()

		if len(args) > 0 {
			// Mettre à jour un package spécifique
			packageName := args[0]
			result, err := UpdatePackage(packageName, manifest, *forceUpdate)
			if err != nil {
				log.Printf("Erreur lors de la mise à jour du package %s : %v", packageName, err)
				os.Exit(1)
			}

			// Afficher le résultat
			switch result.Status {
			case "success":
				if result.PreviousVersion != "" {
					fmt.Printf("Package %s mis à jour avec succès: %s -> %s\n", result.Name, result.PreviousVersion, result.NewVersion)
				} else {
					fmt.Printf("Package %s installé avec succès: %s\n", result.Name, result.NewVersion)
				}
			case "skipped":
				fmt.Printf("Package %s déjà à jour (%s)\n", result.Name, result.NewVersion)
			case "error":
				fmt.Printf("Erreur pour %s: %s\n", result.Name, result.Message)
			}
		} else {
			// Mettre à jour tous les packages
			fmt.Println("Mise à jour de tous les packages installés...")
			results, err := UpdateAllPackages(manifest, *forceUpdate)
			if err != nil {
				log.Fatalf("Erreur lors de la mise à jour des packages : %v", err)
			}

			// Afficher un résumé
			if len(results) == 0 {
				fmt.Println("Aucun package à mettre à jour.")
			} else {
				fmt.Println("\nRésumé des mises à jour:")
				fmt.Println("------------------------")

				updated := 0
				skipped := 0
				failed := 0

				for _, result := range results {
					switch result.Status {
					case "success":
						updated++
						if result.PreviousVersion != "" {
							fmt.Printf("✓ %s: %s -> %s\n", result.Name, result.PreviousVersion, result.NewVersion)
						} else {
							fmt.Printf("✓ %s: installé (%s)\n", result.Name, result.NewVersion)
						}
					case "skipped":
						skipped++
						fmt.Printf("- %s: déjà à jour (%s)\n", result.Name, result.NewVersion)
					case "error":
						failed++
						fmt.Printf("✗ %s: échec (%s)\n", result.Name, result.Message)
					}
				}

				fmt.Println("------------------------")
				fmt.Printf("Total: %d packages, %d mis à jour, %d ignorés, %d échecs\n",
					len(results), updated, skipped, failed)
			}
		}

	case "list":
		if err := ListPackages(); err != nil {
			log.Fatalf("Erreur lors de l'affichage des packages : %v", err)
		}

	case "remove":
		if len(os.Args) < 3 {
			fmt.Println("Usage: goblin remove <package>")
			os.Exit(1)
		}
		pkgName := os.Args[2]
		if err := UninstallPackage(pkgName); err != nil {
			log.Fatalf("Erreur lors de la désinstallation : %v", err)
		}
		fmt.Printf("Package %s désinstallé avec succès\n", pkgName)

	case "sync":
		manifest, err := LoadManifest("sources.yaml")
		if err != nil {
			log.Fatalf("Erreur lors du chargement du manifest : %v", err)
		}

		results, err := SyncPackages(manifest)
		if err != nil {
			log.Fatalf("Erreur lors de la synchronisation : %v", err)
		}

		fmt.Println("\nRésultats de la synchronisation:")
		for _, result := range results {
			fmt.Printf("- %s : %s (%s)\n", result.Name, result.Status, result.Message)
		}

	default:
		fmt.Printf("Commande inconnue : %s\n", command)
		Help()
		os.Exit(1)
	}
}
