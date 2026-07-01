package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Game struct {
	Name          string
	Path          string
	Trigger       string
	Interval      int
	DaemonExclude bool
	LastHash      string
	LastSync      string
}

var AppConfig struct {
	Games          map[string]Game `mapstructure:"games"`
	DaemonInterval int             `mapstructure:"daemon_interval"`
}

const DefaultDaemonInterval = 5

func InitConfig() {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "cpd")
	os.MkdirAll(configPath, 0755)

	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	viper.SetDefault("daemon_interval", DefaultDaemonInterval)

	if err := viper.ReadInConfig(); err != nil {
		viper.Set("games", map[string]Game{})
		viper.SafeWriteConfig()
	}
	viper.Unmarshal(&AppConfig)
}

func GetDaemonInterval() int {
	if AppConfig.DaemonInterval <= 0 {
		return DefaultDaemonInterval
	}
	return AppConfig.DaemonInterval
}

func SetDaemonInterval(minutes int) {
	AppConfig.DaemonInterval = minutes
	viper.Set("daemon_interval", minutes)
	viper.WriteConfig()
}

func SaveGame(game Game) {
	if AppConfig.Games == nil {
		AppConfig.Games = make(map[string]Game)
	}
	AppConfig.Games[game.Name] = game
	viper.Set("games", AppConfig.Games)
	viper.WriteConfig()
}

func RemoveGame(name string) {
	delete(AppConfig.Games, name)
	viper.Set("games", AppConfig.Games)
	viper.WriteConfig()
}

func GetGames() map[string]Game {
	return AppConfig.Games
}
