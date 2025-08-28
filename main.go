package main

import (
	"log"
	"os"
	"tg-bot/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем переменные окружения
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Создаем бота
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Настраиваем канал обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Обрабатываем обновления
	for update := range updates {
		if update.Message != nil {
			if update.Message.Location != nil {
				go handlers.HandleLocation(bot, update)
			} else if update.Message.Text != "" {
				if update.Message.Text == "/start" {
					go handlers.HandleMessage(bot, update)
				} else {
					go handlers.HandleCity(bot, update)
				}
			}
		} else if update.CallbackQuery != nil {
			go handlers.HandleCallback(bot, update)
		}
	}
}
