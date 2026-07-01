package sync

import (
	"archive/tar"
	"checkpointdrive/pkg/config"
	"checkpointdrive/pkg/gdrive"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func ProcessGame(game *config.Game) error {
	info, err := os.Stat(game.Path)
	if err != nil {
		return fmt.Errorf("unable to access game path: %v", err)
	}

	var targetPath string
	var driveFileName string

	if info.IsDir() {
		targetPath = filepath.Join(os.TempDir(), fmt.Sprintf("%s.tar.gz", game.Name))
		driveFileName = fmt.Sprintf("%s.tar.gz", game.Name)

		if err := archiveFolder(game.Path, targetPath); err != nil {
			return fmt.Errorf("failed to archive folder: %v", err)
		}
		defer os.Remove(targetPath)
	} else {
		targetPath = game.Path
		driveFileName = fmt.Sprintf("%s%s", game.Name, filepath.Ext(game.Path))
	}

	hash, err := getFileHash(targetPath)
	if err != nil {
		return fmt.Errorf("failed to get file hash: %v", err)
	}
	if hash == game.LastHash {
		fmt.Printf("Already up to date: %s\n", game.Name)
		return nil
	}

	err = gdrive.Upload(driveFileName, targetPath)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	fmt.Printf("Uploaded: %s\n", game.Name)

	game.LastHash = hash
	game.LastSync = time.Now().UTC().Format(time.RFC3339)
	config.SaveGame(*game)
	return nil
}

func archiveFolder(src string, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %v", err)
	}
	defer out.Close()

	gzipper := gzip.NewWriter(out)
	gzipper.ModTime = time.Time{}
	defer gzipper.Close()

	tw := tar.NewWriter(gzipper)
	defer tw.Close()

	return filepath.Walk(src, func(file string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(fileInfo, "")
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(filepath.Dir(src), file)
		if err != nil {
			return err
		}
		header.Name = relativePath
		header.ModTime = time.Time{}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fileInfo.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, data); err != nil {
				data.Close()
				return err
			}
			data.Close()
		}
		return nil
	})
}

func getFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %v", err)
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", fmt.Errorf("failed to compute hash: %v", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
