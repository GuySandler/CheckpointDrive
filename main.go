package main

import (
	"checkpointdrive/pkg/config"
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cpd",
	Short: "CheckpointDrive is a tool to backup your game saves to Google Drive.",
	Run: func(cmd *cobra.Command, args []string) {
		// TUI
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
	Short: "Start, stop, restart or check the status of the daemon (systemMD)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "start":
			fmt.Println("Starting daemon...")

		case "stop":
			fmt.Println("Stopping daemon...")
		case "restart":
			fmt.Println("Restarting daemon...")
		case "status":
			fmt.Println("Checking daemon status...")
		default:
			fmt.Println("Invalid argument. Use start, stop, restart or status.")
		}
	},
}
