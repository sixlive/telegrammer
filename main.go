package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// AppConfig holds the application configuration.
type AppConfig struct {
	BotKey   string `mapstructure:"bot_key"`
	UserID   int64  `mapstructure:"user_id"`
	Filepath string // This will be set by a flag, not by viper directly.
	Server   bool   // Flag to run in server mode
}

func initConfig() *AppConfig {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/telegrammer")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("TELEGRAMMER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Error reading config file, %s", err)
	}

	var config AppConfig
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	return &config
}

func main() {
	config := initConfig()

	// Setup flags and override config if flags are provided.
	pflag.StringVarP(&config.Filepath, "file", "f", "", "Filepath to file")
	pflag.BoolVar(&config.Server, "server", false, "Run in server mode to listen for new messages")
	pflag.Parse()

	if config.Server {
		runServerMode(config)
		return
	}

	messageText := pflag.Arg(0) // Get the first non-flag command-line argument.

	bot, err := tgbotapi.NewBotAPI(config.BotKey)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	if err := sendMessage(bot, config.UserID, messageText, config.Filepath); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	notifySuccess()
}

func runServerMode(config *AppConfig) {
	bot, err := tgbotapi.NewBotAPI(config.BotKey)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Delete the existing webhook
	_, err = bot.RemoveWebhook()
	if err != nil {
		log.Fatalf("Failed to remove webhook: %v", err)
	}

	// Wait a bit to ensure webhook is fully deleted
	time.Sleep(time.Second * 1)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("Failed to get updates channel: %v", err)
	}

	for update := range updates {
		if update.Message != nil {
			displayDebugData(update)
            os.Exit(0)
		}
	}
}

func displayDebugData(update tgbotapi.Update) {
	// Check if the update contains a message
	if update.Message == nil {
		log.Println("Received update does not contain a message.")
		return
	}

	// Marshal the message part of the update to JSON for readability
	messageData, err := json.MarshalIndent(update.Message, "", "  ")
	if err != nil {
		log.Printf("Error marshalling message data: %v", err)
		return
	}

	// Create a Lipgloss style for the message data display
	messageStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(1, 2).
		Margin(1).
		BorderForeground(lipgloss.Color("63"))

	// Render the message data with the style and print it
	fmt.Println(messageStyle.Render(string(messageData)))
}

func sendMessage(bot *tgbotapi.BotAPI, userID int64, messageText, filepath string) error {
	if filepath != "" {
		return sendDocument(bot, userID, messageText, filepath)
	}
	return sendTextMessage(bot, userID, messageText)
}

func sendDocument(bot *tgbotapi.BotAPI, userID int64, messageText, filepath string) error {
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	fileUpload := tgbotapi.FileBytes{Name: filepath, Bytes: fileBytes}
	msg := tgbotapi.NewDocumentUpload(userID, fileUpload)
	msg.Caption = messageText
	_, err = bot.Send(msg)
	return err
}

func sendTextMessage(bot *tgbotapi.BotAPI, userID int64, messageText string) error {
	msg := tgbotapi.NewMessage(userID, messageText)
	_, err := bot.Send(msg)
	return err
}

func notifySuccess() {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	fmt.Println(style.Render("Message sent!"))
}
