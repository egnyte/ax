package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
	"github.com/kardianos/osext"
	"github.com/spf13/cobra"
)

var (
	gitOrganisationName = "egnyte"
	gitRepositoryName   = "ax"
	upgradeCommand      = &cobra.Command{
		Use:   "upgrade",
		Short: "Check if an upgrade of Ax is available and install it",
		Run: func(cmd *cobra.Command, args []string) {
			if err := upgradeVersion(); err != nil {
				fmt.Println("Upgrade failed.")
			} else {
				fmt.Println("Upgrade has been completed successfully.")
			}
		},
	}
)

func getLatestReleaseData() *github.RepositoryRelease {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(context.Background(), gitOrganisationName, gitRepositoryName)
	if err != nil {
		fmt.Printf("Couldn't fetch release information: %v", err)
		os.Exit(1)
	}

	return release
}

func getLatestAssetLink() (string, error) {
	release := getLatestReleaseData()

	for _, element := range release.Assets {
		assetLink := element.GetBrowserDownloadURL()
		if strings.Contains(assetLink, runtime.GOOS) && strings.Contains(assetLink, runtime.GOARCH) {
			return assetLink, nil
		}
	}

	return "", errors.New("assset link couldn't be found")
}

func getLatestReleaseTag() string {
	release := getLatestReleaseData()
	return release.GetTagName()
}

func downloadFile(filePath string, url string) error {

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}

	return nil
}

func upgradeVersion() error {
	latestVersion := getLatestReleaseTag()

	if version == latestVersion {
		fmt.Println("Latest version is already installed.")
		return nil
	}

	fmt.Printf("New version detected. Current version: %s. Latest version: %s\n", version, latestVersion)
	inputScanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Do you want to upgrade Ax to the latest version? (yes/no): ")
	inputScanner.Scan()

	if text := inputScanner.Text(); strings.ToLower(text) != "yes" {
		fmt.Println("Aborting the upgrade.")
		os.Exit(1)
	}

	latestAssetURL, err := getLatestAssetLink()
	if err != nil {
		return err
	}

	extractionPath, err := ioutil.TempDir("/tmp", "egnyte_ax")
	if err != nil {
		return err
	}
	downloadPath := fmt.Sprintf("%s/ax.tar.gz", extractionPath)

	if _, err := os.Stat(extractionPath); os.IsNotExist(err) {
		os.Mkdir(extractionPath, 0755)
	}

	fmt.Println("Ax upgrade in progress...")

	if err := downloadFile(downloadPath, latestAssetURL); err != nil {
		return err
	}

	file, err := os.Open(downloadPath)
	if err != nil {
		return err
	}

	// extract the tar.gz file
	if err := extractTar(extractionPath, file); err != nil {
		return err
	}

	// switch old binary with a new one
	currentBinaryPath, _ := osext.Executable()
	if err := os.Rename(extractionPath+"/ax", currentBinaryPath); err != nil {
		return err
	}
	defer os.RemoveAll(extractionPath)

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
