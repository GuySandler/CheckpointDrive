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
	"strings"
	"time"
)

func ProcessGame(game *config.Game) error {
	info, err := os.Stat(game.Path)
	localExists := err == nil

	var driveFileName string
	var isDir bool

	if localExists {
		isDir = info.IsDir()
		if isDir {
			driveFileName = fmt.Sprintf("%s.tar.gz", game.Name)
		} else {
			driveFileName = fmt.Sprintf("%s%s", game.Name, filepath.Ext(game.Path))
		}
	} else {
		driveFileName = fmt.Sprintf("%s.tar.gz", game.Name)
	}

	fileID, driveTime, driveExists, err := gdrive.GetFileInfo(driveFileName)
	if err != nil {
		return fmt.Errorf("failed to check Drive file: %v", err)
	}

	if !localExists {
		if !driveExists {
			driveFileName = game.Name
			fileID, driveTime, driveExists, err = gdrive.GetFileInfo(driveFileName)
			if err != nil {
				return fmt.Errorf("failed to check Drive file: %v", err)
			}
			if !driveExists {
				return fmt.Errorf("game not found on Drive: %s", game.Name)
			}
		}
		isDir := strings.HasSuffix(driveFileName, ".tar.gz")
		return downloadFromDrive(game, fileID, driveFileName, isDir)
	}

	var targetPath string

	if isDir {
		targetPath = filepath.Join(os.TempDir(), fmt.Sprintf("%s.tar.gz", game.Name))

		if err := archiveFolder(game.Path, targetPath); err != nil {
			return fmt.Errorf("failed to archive folder: %v", err)
		}
		defer os.Remove(targetPath)
	} else {
		targetPath = game.Path
	}

	hash, err := getFileHash(targetPath)
	if err != nil {
		return fmt.Errorf("failed to get file hash: %v", err)
	}

	if hash == game.LastHash {
		if driveExists && game.LastSync != "" {
			driveNewer := isDriveNewer(driveTime, game.LastSync)
			if driveNewer {
				return downloadFromDrive(game, fileID, driveFileName, isDir)
			}
		}
		fmt.Printf("Already up to date: %s\n", game.Name)
		return nil
	}

	if driveExists && game.LastSync != "" && isDriveNewer(driveTime, game.LastSync) {
		return downloadFromDrive(game, fileID, driveFileName, isDir)
	}

	err = gdrive.Upload(driveFileName, targetPath)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	fmt.Printf("Uploaded: %s\n", game.Name)

	game.LastHash = hash
	game.LastSync = time.Now().UTC().Format(time.RFC3339Nano)
	config.SaveGame(*game)
	return nil
}

func isDriveNewer(driveTimeStr string, lastSync string) bool {
	if driveTimeStr == "" || lastSync == "" {
		return false
	}
	driveTime, err := time.Parse(time.RFC3339Nano, driveTimeStr)
	if err != nil {
		return false
	}
	syncTime, err := time.Parse(time.RFC3339Nano, lastSync)
	if err != nil {
		syncTime, err = time.Parse(time.RFC3339, lastSync)
		if err != nil {
			return false
		}
	}
	return driveTime.Sub(syncTime) > time.Second
}

func downloadFromDrive(game *config.Game, fileID string, driveFileName string, isDir bool) error {
	if isDir {
		tmpArchive := filepath.Join(os.TempDir(), driveFileName)
		defer os.Remove(tmpArchive)

		if err := gdrive.Download(fileID, tmpArchive); err != nil {
			return fmt.Errorf("failed to download archive: %v", err)
		}

		hash, err := getFileHash(tmpArchive)
		if err != nil {
			return fmt.Errorf("failed to hash archive: %v", err)
		}
		game.LastHash = hash

		if err := extractArchive(tmpArchive, filepath.Dir(game.Path)); err != nil {
			return fmt.Errorf("failed to extract archive: %v", err)
		}
	} else {
		if err := gdrive.Download(fileID, game.Path); err != nil {
			return fmt.Errorf("failed to download file: %v", err)
		}

		hash, err := getFileHash(game.Path)
		if err != nil {
			return fmt.Errorf("failed to hash downloaded file: %v", err)
		}
		game.LastHash = hash
	}

	game.LastSync = time.Now().UTC().Format(time.RFC3339Nano)
	config.SaveGame(*game)
	fmt.Printf("Downloaded: %s\n", game.Name)
	return nil
}

func extractArchive(archivePath string, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %v", err)
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
		case tar.TypeReg:
			base := filepath.Dir(target)
			if err := os.MkdirAll(base, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %v", err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %v", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file: %v", err)
			}
			f.Close()
		}
	}
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
