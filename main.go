package main

import (
	"checkpointdrive/pkg/config"
	"checkpointdrive/pkg/daemon"
	"checkpointdrive/pkg/sync"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cpd",
	Short: "CheckpointDrive is a tool to backup your game saves to Google Drive.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CheckpointDrive: A tool to backup your game saves to Google Drive.")
		fmt.Println("future TUI will go here")
	},
}

var addCmd = &cobra.Command{
	Use:   "add [path] [name]",
	Short: "Add a game to sync",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		game := config.Game{Path: args[0], Name: args[1], Trigger: "manual"} // TODO
		config.SaveGame(game)
		fmt.Printf("Added game: %s", args[1])
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a game from sync",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config.RemoveGame(args[0])
		fmt.Printf("Removed game: %s", args[0])
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon [start, stop, restart, status]",
	Short: "Start, stop, restart or check the status of the daemon (systemctl)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "start":
			err := daemon.Start()
			if err != nil {
				fmt.Printf("Error starting daemon: %v\n", err)
			}
		case "stop":
			err := daemon.Stop()
			if err != nil {
				fmt.Printf("Error stopping daemon: %v\n", err)
			}
		case "restart":
			err := daemon.Stop()
			if err != nil {
				fmt.Printf("Error restarting daemon: %v\n", err)
			}
			err = daemon.Start()
			if err != nil {
				fmt.Printf("Error starting daemon: %v\n", err)
			}
		case "status":
			err := daemon.Status()
			if err != nil {
				fmt.Printf("Error checking daemon status: %v\n", err)
			}
		default:
			fmt.Println("Invalid argument. Use start, stop, restart or status.")
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all games being synced",
	Run: func(cmd *cobra.Command, args []string) {
		games := config.GetGames()
		if len(games) == 0 {
			fmt.Println("No games are being synced.")
			return
		}
		fmt.Println("Games being synced:")
		for _, game := range games {
			fmt.Printf("- %s (%s)\n", game.Name, game.Path)
		}
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync [name or *]",
	Short: "Sync a game by name or all by using *",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "*":
			games := config.GetGames()
			for _, game := range games {
				err := sync.ProcessGame(&game)
				if err != nil {
					fmt.Printf("Failed to sync game %s: %v\n", game.Name, err)
				} else {
					fmt.Printf("Successfully synced game: %s\n", game.Name)
				}
			}
		default:
			game, exists := config.GetGames()[args[0]]
			if !exists {
				fmt.Printf("Game not found: %s\n", args[0])
				return
			}
			err := sync.ProcessGame(&game)
			if err != nil {
				fmt.Printf("Failed to sync game %s: %v\n", game.Name, err)
			} else {
				fmt.Printf("Successfully synced game: %s\n", game.Name)
			}
		}
	},
}

var configCmd = &cobra.Command{ // TODO: better docs
	Use:   "config [edit, drive]",
	Short: "config the application",
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "edit":
			fmt.Println("Edit config file")
			cmd := exec.Command("nano", filepath.Join(os.Getenv("HOME"), ".config", "cpd", "config.json"))
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("Error opening config file: %v\n", err)
			}
		case "drive":
			fmt.Println("Config drive")
		default:
			fmt.Println("Invalid argument. Use edit or drive.")
		}
	},
}

func main() {
	config.InitConfig()

	rootCmd.AddCommand(addCmd, removeCmd, daemonCmd, listCmd, syncCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Errorf("error executing command: %v", err)
	}
}
