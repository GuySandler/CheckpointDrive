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
	if hash == game.lastHash {
		return nil
	}

	err = gdrive.Upload(driveFileName, targetPath)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	game.LastHash = hash
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
	defer gzipper.Close()

	tar := tar.NewReader(gzipper)
	defer tar.Close()

	return filepath.Walk(src, func(file string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(fileInfo, file)
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(filepath.Dir(src), file)
		if err != nil {
			return err
		}
		header.Name = relativePath
		if !fileInfo.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()
			if _, err := io.Copy(tar, data); err != nil {
				return err
			}
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
