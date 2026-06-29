package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const serviceName = "cpd.service"

func getServicePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", serviceName)
}

func Start() error {
	servicePath := getServicePath()
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		if err := installService(servicePath); err != nil {
			return fmt.Errorf("failed to install daemon service: %v", err)
		}
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
	fmt.Println("Checking daemon status...")
	cmd := exec.Command("systemctl", "--user", "status", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
ExecStart=%s daemon start
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
