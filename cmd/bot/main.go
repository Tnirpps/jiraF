package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/user/telegram-bot/internal/bot"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	// Get Telegram bot token
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	// Get Todoist API token
	todoistToken := os.Getenv("TODOIST_API_TOKEN")
	if todoistToken == "" {
		log.Fatal("TODOIST_API_TOKEN is required")
	}

	// Create the bot with the tokens
	b, err := bot.New(telegramToken, todoistToken)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}

	go func() {
		log.Println("Starting bot...")
		if err := b.Start(); err != nil {
			log.Fatalf("Error starting bot: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the bot
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down bot...")
	b.Stop()
	log.Println("Bot stopped")
}
