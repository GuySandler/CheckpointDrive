package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func getClient(config *oauth2.Config) *oauth2.Token {
	home, _ := os.UserHomeDir()
	tokenFile := filepath.Join(home, ".config", "cpd", "token.json")
	file, err := os.Open(tokenFile)
	if err != nil {
		defer file.Close()
		token := &oauth2.Token{}
		json.NewDecoder(file).Decode(token)
		return token
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link and type the authorization code: \n%v\n", authURL)
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		panic(fmt.Errorf("unable to read the authentication code: %v", err))
	}
	token, _ := config.Exchange(context.TODO(), authCode)
	file2, _ := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer file2.Close()
	json.NewEncoder(file2).Encode(token)

	return token
}

func getDriveService() (*drive.Service, error) {
	ctx := context.Background()
	home, _ := os.UserHomeDir()
	credentialsFile := filepath.Join(home, ".config", "cpd", "credentials.json")
	credentials, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %v", err)
	}

	config, err := google.ConfigFromJSON(credentials, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("unable to create config: %v", err)
	}

	client := config.Client(ctx, getClient(config))
	return drive.NewService(ctx, option.WithHTTPClient(client))
}

func Upload(gameName string, filePath string) error {
	service, err := getDriveService()
	if err != nil {
		return fmt.Errorf("unable to create Drive service: %v", err)
	}

	folderId := ""
	query := "name='Checkpoint' and mimeType='application/vnd.google-apps.folder' and trashed=false"
	result, err := service.Files.List().Q(query).Do()
	if len(result.Files) == 0 {
		folder := &drive.File{Name: "CheckpointDrive", MimeType: "application/vnd.google-apps.folder"}
		folder, _ = service.Files.Create(folder).Do()
		folderId = folder.Id
	} else {
		folderId = result.Files[0].Id
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to open file: %v", err)
	}
	defer file.Close()

	driveFile := &drive.File{Name: gameName, Parents: []string{folderId}}
	_, err = service.Files.Create(driveFile).Media(file).Do()
	return err
}
