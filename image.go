package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Image struct {
	Owner string
	Name  string
	Tag   string
}

type Token struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type TagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type Manifest struct {
	Name          string    `json:"name"`
	Tag           string    `json:"tag"`
	Architecture  string    `json:"architecture"`
	SchemaVersion int       `json:"schemaVersion"`
	Layers        []Layer   `json:"fsLayers"`
	History       []History `json:"history"`
}

type History struct {
	V1Compatibility string `json:"v1Compatibility"`
}

type Layer struct {
	Digest string `json:"blobSum"`
}

func parseImageName(imageS string) Image {
	i := Image{}
	if strings.Contains(imageS, ":") {
		imageSplit := strings.Split(imageS, ":")
		i.Name = imageSplit[0]
		i.Tag = imageSplit[1]
	} else {
		i.Name = imageS
	}

	if strings.Contains(i.Name, "/") {
		nameSplit := strings.Split(i.Name, "/")
		i.Owner = nameSplit[0]
		i.Name = nameSplit[1]
	} else {
		i.Owner = "library"
	}

	return i
}

func httpGet(url string, token string) []byte {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal("Error crafting request for ", url)
	}

	// Set token if one is provided
	if len(token) != 0 {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("Error completing request for ", url)
	}

	resS, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Error reading response for ", url)
	}

	return resS
}

func getToken(image Image) Token {
	tokenURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s/%s:pull", image.Owner, image.Name)
	tokenS := httpGet(tokenURL, "")

	var token Token
	err := json.Unmarshal(tokenS, &token)
	if err != nil {
		log.Fatal("Error parsing auth token")
	}

	return token
}

func getTags(image Image, token Token) []string {
	tagsURL := fmt.Sprintf("https://index.docker.io/v2/%s/%s/tags/list", image.Owner, image.Name)
	tagsS := httpGet(tagsURL, token.Token)

	var tags TagsResponse
	err := json.Unmarshal(tagsS, &tags)
	if err != nil {
		log.Fatal("Error parsing tags")
	}

	return tags.Tags
}

func getManifest(image Image, token Token) Manifest {
	manifestURL := fmt.Sprintf("https://index.docker.io/v2/%s/%s/manifests/%s", image.Owner, image.Name, image.Tag)
	manifestS := httpGet(manifestURL, token.Token)

	var manifest Manifest
	err := json.Unmarshal(manifestS, &manifest)
	if err != nil {
		log.Fatal("Error parsing manifest")
	}
	return manifest
}

func downloadLayer(image Image, layer string, outDir string, token Token) (cached bool) {
	digest := strings.Replace(layer, "sha256:", "", 1)
	outFile := filepath.Join(outDir, "layers", digest+".tar.gz")

	if _, err := os.Stat(outFile); err == nil {
		log.Println("[CACHED]", layer)
		return true
	}

	layerURL := fmt.Sprintf("https://index.docker.io/v2/%s/%s/blobs/%s", image.Owner, image.Name, layer)

	req, err := http.NewRequest(http.MethodGet, layerURL, nil)
	if err != nil {
		log.Fatal("Error crafting request for layer ", layer)
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("Error completing request for layer", layer)
	}

	out, _ := os.Create(outFile)
	defer out.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		log.Fatal("Error writing to ", outFile, "- ", err)
	}

	log.Println("[DOWNLOADED]", layer)
	return false
}

func processLayer(image Image, layer string, outDir string, token Token) {
	layerCached := downloadLayer(image, layer, outDir, token)
	if !layerCached {
		digest := strings.Replace(layer, "sha256:", "", 1)
		layerPath := filepath.Join(outDir, "layers", digest+".tar.gz")
		fsPath := filepath.Join(outDir, "fs")
		untar(layerPath, fsPath)
	}
}

func DownloadImage(imageS string, outDir string) string {
	image := parseImageName(imageS)
	token := getToken(image)

	// No tag provided, let's infer the best one
	// If the "latest" tag exists, use that
	// Otherwise, use whatever the last tag returned is
	if len(image.Tag) == 0 {
		tags := getTags(image, token)
		image.Tag = tags[len(tags)-1]
		for _, tag := range tags {
			if tag == "latest" {
				image.Tag = "latest"
			}
		}
	}

	manifest := getManifest(image, token)

	fsDir := fmt.Sprintf("%s_%s_%s", image.Owner, image.Name, image.Tag)
	fullPath := filepath.Join(outDir, fsDir)
	layersPath := filepath.Join(fullPath, "layers")
	fsPath := filepath.Join(fullPath, "fs")
	_, err := os.Stat(fullPath)
	if err != nil {
		// Directory doesn't exist, create it
		os.Mkdir(fullPath, 0755)
		os.Mkdir(layersPath, 0755)
		os.Mkdir(fsPath, 0755)
	}

	// Download and extract each layer
	for _, layer := range manifest.Layers {
		processLayer(image, layer.Digest, fullPath, token)
	}

	return fsPath
}

func untar(path string, outDir string) {
	log.Println("[UNTAR]", path)
	cmd := exec.Command("tar", "-xf", path, "-C", outDir)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Run()
	cmd.Wait()
}
