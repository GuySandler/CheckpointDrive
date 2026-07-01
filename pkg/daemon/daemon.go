package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"checkpointdrive/pkg/config"
	"checkpointdrive/pkg/sync"
)

const serviceName = "cpd.service"

func getServicePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", serviceName)
}

func Start() error {
	servicePath := getServicePath()
	if err := installService(servicePath); err != nil {
		return fmt.Errorf("failed to install daemon service: %v", err)
	}

	fmt.Println("Starting daemon service...")
	cmd := exec.Command("systemctl", "--user", "start", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start daemon service: %v", err)
	}

	fmt.Println("Daemon started successfully")
	return nil
}

func Stop() error {
	fmt.Println("Stopping daemon service...")
	cmd := exec.Command("systemctl", "--user", "stop", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop daemon service: %v", err)
	}
	fmt.Println("Daemon stopped successfully")
	return nil
}

func Status() error {
	cmd := exec.Command("systemctl", "--user", "status", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	return nil
}

func Run() {
	interval := config.GetDaemonInterval()
	fmt.Printf("CheckpointDrive daemon running (interval: %ds)\n", interval)

	for {
		runOnce()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func runOnce() {
	games := config.GetGames()
	now := time.Now()
	for _, game := range games {
		if game.DaemonExclude {
			continue
		}

		if game.Interval > 0 {
			if game.LastSync != "" {
				lastSync, err := time.Parse(time.RFC3339, game.LastSync)
				if err == nil {
					elapsed := now.Sub(lastSync)
					if elapsed.Seconds() < float64(game.Interval) {
						continue
					}
				}
			}
		}

		err := sync.ProcessGame(&game)
		if err != nil {
			fmt.Printf("Failed to sync game %s: %v\n", game.Name, err)
		}
	}
}

func installService(servicePath string) error {
	fmt.Println("Installing daemon service...")
	err := os.MkdirAll(filepath.Dir(servicePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create service directory: %v", err)
	}
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	serviceContent := fmt.Sprintf(
		`[Unit]
Description=CheckpointDrive Daemon
After=network.target
[Service]
ExecStart=%s daemon run
Restart=on-failure
[Install]
WantedBy=default.target`,
		execPath)

	err = os.WriteFile(servicePath, []byte(serviceContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	exec.Command("systemctl", "--user", "enable", serviceName).Run()
	return nil
}
