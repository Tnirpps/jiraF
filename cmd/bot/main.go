package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/user/telegram-bot/internal/bot"
	"github.com/user/telegram-bot/internal/commands"
	"github.com/user/telegram-bot/internal/db"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	todoistToken := os.Getenv("TODOIST_API_TOKEN")
	if todoistToken == "" {
		log.Fatal("TODOIST_API_TOKEN is required")
	}

	dbManager, err := db.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbManager.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := dbManager.InitSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// Assert that our db.Manager implements the commands.DBManager interface
	var _ commands.DBManager = dbManager

	b, err := bot.New(telegramToken, todoistToken, dbManager)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}

	go func() {
		log.Println("Starting bot...")
		if err := b.Start(); err != nil {
			log.Fatalf("Error starting bot: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down bot...")
	b.Stop()
	log.Println("Bot stopped")
}
