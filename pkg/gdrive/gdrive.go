package gdrive

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

//go:embed credentials.json
var credentialsFS embed.FS

func init() {
	var err error
	credentialsJSON, err = credentialsFS.ReadFile("credentials.json")
	if err != nil {
		panic(fmt.Sprintf("failed to load embedded credentials: %v", err))
	}
}

var credentialsJSON []byte

func getClient(config *oauth2.Config) *oauth2.Token {
	home, _ := os.UserHomeDir()
	tokenFile := filepath.Join(home, ".config", "cpd", "token.json")
	file, err := os.Open(tokenFile)
	if err == nil {
		defer file.Close()
		token := &oauth2.Token{}
		json.NewDecoder(file).Decode(token)
		return token
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link and authorize the app: \n%v\n", authURL)
	openBrowser(authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		panic(fmt.Errorf("unable to read the authentication code: %v", err))
	}
	token, _ := config.Exchange(context.TODO(), authCode)

	os.MkdirAll(filepath.Dir(tokenFile), 0755)
	file2, _ := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer file2.Close()
	json.NewEncoder(file2).Encode(token)

	return token
}

func getDriveService() (*drive.Service, error) {
	ctx := context.Background()

	config, err := google.ConfigFromJSON(credentialsJSON, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("unable to create config: %v", err)
	}

	home, _ := os.UserHomeDir()
	tokFile := filepath.Join(home, ".config", "cpd", "token.json")

	f, err := os.Open(tokFile)
	if err != nil {
		return nil, fmt.Errorf("you are not logged in. Run 'cpd login' first")
	}
	defer f.Close()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("unable to decode token: %v", err)
	}

	client := config.Client(ctx, tok)
	return drive.NewService(ctx, option.WithHTTPClient(client))
}

func Upload(gameName string, filePath string) error {
	service, err := getDriveService()
	if err != nil {
		return fmt.Errorf("unable to create Drive service: %v", err)
	}

	folderId := ""
	query := "name='CheckpointDrive' and mimeType='application/vnd.google-apps.folder' and trashed=false"
	result, err := service.Files.List().Q(query).Do()
	if err != nil {
		return fmt.Errorf("unable to search for CheckpointDrive folder: %v", err)
	}

	if len(result.Files) == 0 {
		folder := &drive.File{Name: "CheckpointDrive", MimeType: "application/vnd.google-apps.folder"}
		createdFolder, err := service.Files.Create(folder).Do()
		if err != nil {
			return fmt.Errorf("unable to create CheckpointDrive folder: %v", err)
		}
		folderId = createdFolder.Id
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

func Authenticate() error {
	config, err := google.ConfigFromJSON(credentialsJSON, drive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("unable to parse credentials file: %v", err)
	}

	config.RedirectURL = "http://localhost:8080"
	codeCh := make(chan string)
	errCh := make(chan error)

	serve := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in the callback request")
			http.Error(w, "No code in the callback request", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "Authentication successful! You can close this window.")
		codeCh <- code
	})

	go func() {
		if err := serve.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to start HTTP server: %v", err)
		}
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Opening browser to connect your Google Drive...\nIf it does not open, use the following link: \n%v\n", authURL)
	openBrowser(authURL)

	select {
	case code := <-codeCh:
		token, err := config.Exchange(context.Background(), code)
		if err != nil {
			return fmt.Errorf("unable to exchange code for token: %v", err)
		}
		home, _ := os.UserHomeDir()
		tokenFile := filepath.Join(home, ".config", "cpd", "token.json")

		os.MkdirAll(filepath.Dir(tokenFile), 0755)
		file, err := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("unable to create token file: %v", err)
		}
		defer file.Close()
		json.NewEncoder(file).Encode(token)
		fmt.Println("Successfully connected to Google Drive!")
	case err := <-errCh:
		return err
	}

	if err := serve.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %v", err)
	}

	return nil
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	_ = err
}
