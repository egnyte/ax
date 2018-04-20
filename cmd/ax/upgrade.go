package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
	"github.com/kardianos/osext"
)

func getLatestReleaseData() *github.RepositoryRelease {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "egnyte", "ax")
	if err != nil {
		fmt.Printf("Couldn't get a release information")
		os.Exit(1)
	}

	return release
}

func getLatestAssetLink() string {
	release := getLatestReleaseData()
	var assetDownloadLink string

	for _, element := range release.Assets {
		assetLink := element.GetBrowserDownloadURL()
		if strings.Contains(assetLink, runtime.GOOS) && strings.Contains(assetLink, runtime.GOARCH) {
			assetDownloadLink = assetLink
			break
		}
	}

	return assetDownloadLink
}

func getLatestReleaseTag() string {
	release := getLatestReleaseData()
	return release.GetTagName()
}

func downloadFile(filepath string, url string) error {

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func upgradeVersion() error {
	if latestVersion := getLatestReleaseTag(); version != latestVersion {
		fmt.Printf("New version detected. Current version: %s. Latest version: %s\n", version, latestVersion)
		inputScanner := bufio.NewScanner(os.Stdin)
		fmt.Print("Do you want to upgrade an Ax to its latest version? (yes/no): ")
		inputScanner.Scan()
		text := inputScanner.Text()

		if strings.ToLower(text) == "yes" || strings.ToLower(text) == "y" {
			latestAssetURL := getLatestAssetLink()
			downloadPath := "/tmp/egnyte_ax/ax.tar.gz"
			extractionPath := "/tmp/egnyte_ax/"

			if _, err := os.Stat(extractionPath); os.IsNotExist(err) {
				os.Mkdir(extractionPath, 0755)
			}

			fmt.Println("Ax upgrade in progress...")

			err := downloadFile(downloadPath, latestAssetURL)
			if err != nil {
				return err
			}

			file, err := os.Open(downloadPath)
			if err != nil {
				return err
			}
			defer os.Remove(downloadPath)

			// extract the tar.gz file
			err = extractTar(extractionPath, file)
			if err != nil {
				return err
			}

			// switch old binary with a new one
			currentBinaryPath, _ := osext.Executable()
			err = os.Rename(extractionPath+"ax", currentBinaryPath)
			if err != nil {
				return err
			}
			defer os.RemoveAll(extractionPath)
		} else {
			fmt.Println("Aborting the upgrade.")
			os.Exit(1)
		}
	} else {
		fmt.Println("Latest version is already installed.")
	}
	return nil
}

func extractTar(dst string, r io.Reader) error {

	gzr, err := gzip.NewReader(r)
	defer gzr.Close()
	if err != nil {
		return err
	}

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		}
	}
}
