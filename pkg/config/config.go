package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Game struct {
	Name     string `mapstructure:"name"`
	Path     string `mapstructure:"path"`
	Trigger  string `mapstructure:"trigger"`
	Interval int    `mapstructure:"interval"`
	LastHash string `mapstructure:"last_hash"`
}

var AppConfig struct {
	Games map[string]Game `mapstructure:"games"`
}

func InitConfig() {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "cpd")
	os.MkdirAll(configPath, 0755)

	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		viper.Set("games", map[string]Game{})
		viper.SafeWriteConfig()
	}
	viper.Unmarshal(&AppConfig)
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
