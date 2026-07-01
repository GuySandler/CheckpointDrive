package main

import (
	"checkpointdrive/pkg/config"
	"checkpointdrive/pkg/daemon"
	"checkpointdrive/pkg/gdrive"
	"checkpointdrive/pkg/sync"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

var addInterval int
var addNoDaemon bool

var addCmd = &cobra.Command{
	Use:   "add [path] [name]",
	Short: "Add a game to sync",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		game := config.Game{
			Path:          args[0],
			Name:          args[1],
			Trigger:       "manual",
			Interval:      addInterval,
			DaemonExclude: addNoDaemon,
		}
		config.SaveGame(game)
		fmt.Printf("Added game: %s\n", args[1])
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
	Use:   "daemon [start, stop, restart, status, run]",
	Short: "Manage the daemon (systemctl) or run the sync loop",
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
		case "run":
			daemon.Run()
		default:
			fmt.Println("Invalid argument. Use start, stop, restart, status or run.")
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
			excluded := ""
			if game.DaemonExclude {
				excluded = " [no-daemon]"
			}
			intervalInfo := ""
			if game.Interval > 0 {
				intervalInfo = fmt.Sprintf(" (interval: %ds)", game.Interval)
			}
			fmt.Printf("- %s (%s)%s%s\n", game.Name, game.Path, intervalInfo, excluded)
		}
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync [optional: names separated by commas]",
	Short: "Sync all or spesify names (minecraft,factorio)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			games := config.GetGames()
			for _, game := range games {
				err := sync.ProcessGame(&game)
				if err != nil {
					fmt.Printf("Failed to sync game %s: %v\n", game.Name, err)
				}
			}
		} else {
			games := strings.Split(args[0], ",")
			allGames := config.GetGames()
			for _, name := range games {
				game, exists := allGames[name]
				if !exists {
					fmt.Printf("Game not found: %s\n", name)
					return
				}
				err := sync.ProcessGame(&game)
				if err != nil {
					fmt.Printf("Failed to sync game %s: %v\n", game.Name, err)
				}
			}
		}
	},
}

var configCmd = &cobra.Command{
	Use:   "config [edit, drive, set <key> <value>]",
	Short: "Configure the application",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Usage: cpd config [edit, drive, set <key> <value>]")
			return
		}
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
			err := gdrive.Authenticate()
			if err != nil {
				fmt.Printf("Error authenticating with Google Drive: %v\n", err)
			}
		case "set":
			if len(args) < 3 {
				fmt.Println("Usage: cpd config set <key> <value>")
				return
			}
			switch args[1] {
			case "daemon-interval":
				seconds, err := strconv.Atoi(args[2])
				if err != nil {
					fmt.Printf("Invalid interval: %v\n", err)
					return
				}
				config.SetDaemonInterval(seconds)
				fmt.Printf("Daemon interval set to %d seconds\n", seconds)
			default:
				fmt.Printf("Unknown config key: %s\n", args[1])
			}
		default:
			fmt.Printf("Invalid argument. Use edit, drive, or set.\n")
		}
	},
}

func main() {
	addCmd.Flags().IntVarP(&addInterval, "interval", "i", 0, "Sync interval in seconds (0 = use daemon default)")
	addCmd.Flags().BoolVarP(&addNoDaemon, "no-daemon", "n", false, "Exclude from daemon auto-sync")

	config.InitConfig()

	rootCmd.AddCommand(addCmd, removeCmd, daemonCmd, listCmd, syncCmd, configCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Errorf("error executing command: %v", err)
	}
}
