package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Manifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

func DownloadImageAsTar(image string) (string, *v1.ConfigFile) {
	img, err := crane.Pull(image)
	confFile, _ := img.ConfigFile()

	if err != nil {
		log.Fatal("Error pulling image ", image, ": ", err)
	}
	tmpDir := filepath.Join(os.TempDir(), strings.ReplaceAll(image, ":", "."))

	_, err = os.Stat(tmpDir)
	if !os.IsNotExist(err) { // idk why os.IsExist doesn't work?
		log.Println("Already downloaded image", image)
		return tmpDir, confFile
	} else {
		err = os.Mkdir(tmpDir, 0755)
		if err != nil {
			log.Fatal("Could not create dir", tmpDir)
		}
	}

	imagePath := filepath.Join(tmpDir, "image.tar")
	log.Println("Saving image at", imagePath)
	err = crane.SaveLegacy(img, image, imagePath)
	if err != nil {
		log.Fatal("Error saving image:", err)
	}
	untar(imagePath, tmpDir)
	ProcessImageLayers(tmpDir)

	return tmpDir, confFile
}

func untar(path string, outDir string) {
	log.Println("UNTAR", path)
	cmd := exec.Command("tar", "-xf", path, "-C", outDir)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Run()
	cmd.Wait()
}

func ProcessImageLayers(path string) {
	var manifests []Manifest
	manifestJSON, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		log.Fatal("Error reading manifest: ", err)
	}

	err = json.Unmarshal(manifestJSON, &manifests)
	if err != nil || len(manifests) == 0 {
		log.Fatal("Error processing container manifest: ", err)
	}

	manifest := manifests[0]
	for _, layer := range manifest.Layers {
		layerTarPath := filepath.Join(path, layer)
		untar(layerTarPath, path)
	}
}
