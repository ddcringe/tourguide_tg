package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"tg-bot/api"
	"tg-bot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleMessage обрабатывает текстовые сообщения
func HandleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	if update.Message.Text == "/start" {
		msg.Text = "👋 Привет! Я помогу найти интересные достопримечательности.\n\n📍 Отправь мне название города (например: \"Москва\", \"Санкт-Петербург\")\n🗺️ Или отправь свою геолокацию для поиска рядом с тобой"
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButtonLocation("📍 Отправить геолокацию"),
			),
		)
	} else {
		// Обрабатываем как название города
		go HandleCity(bot, update)
	}

	bot.Send(msg)
}

// HandleCity обрабатывает поиск достопримечательностей по городу
func HandleCity(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	cityName := update.Message.Text

	// Получаем достопримечательности по городу через API
	attractions, err := api.GetAttractionsByCity(cityName)
	if err != nil {
		log.Printf("Ошибка при запросе к API: %v", err)
		msg.Text = "❌ Ошибка при поиске достопримечательностей. Попробуйте позже."
	} else if len(attractions) == 0 {
		msg.Text = fmt.Sprintf("🏙️ В городе \"%s\" не найдено достопримечательностей 😢\nПопробуйте другой город или проверьте написание.", cityName)
	} else {
		msg.Text = fmt.Sprintf("🏙️ Достопримечательности в %s:\n\n%s", cityName, formatAttractionsList(attractions))
		msg.ReplyMarkup = createAttractionsKeyboard(attractions)
	}

	bot.Send(msg)
}

// HandleLocation обрабатывает сообщения с геолокацией
func HandleLocation(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	// Получаем достопримечательности вокруг локации
	// Используем небольшой радиус (0.001) для поиска поблизости
	attractions, err := api.GetAttractionsByLocation(
		update.Message.Location.Latitude,
		update.Message.Location.Longitude,
		0.001, // небольшой радиус для поиска поблизости
	)

	if err != nil {
		log.Printf("Ошибка при запросе к API: %v", err)
		msg.Text = "❌ Ошибка при поиске достопримечательностей. Попробуйте позже."
	} else if len(attractions) == 0 {
		msg.Text = "Рядом нет достопримечательностей 😢 Попробуйте увеличить радиус поиска."
	} else {
		msg.Text = "Рядом с вами:\n" + formatAttractionsList(attractions)
		msg.ReplyMarkup = createAttractionsKeyboard(attractions)
	}

	bot.Send(msg)
}

// HandleCallback обрабатывает callback-и от inline кнопок
func HandleCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	bot.Send(callback)

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "")
	if strings.HasPrefix(update.CallbackQuery.Data, "attraction_") {
		idStr := strings.TrimPrefix(update.CallbackQuery.Data, "attraction_")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			msg.Text = "Ошибка"
		} else {
			detail, err := api.GetAttractionDetail(id)
			if err != nil {
				msg.Text = "Ошибка при запросе к API"
			} else {
				msg.Text = formatAttractionDetail(detail)
			}
		}
	}

	bot.Send(msg)
}

// Вспомогательные функции для форматирования
func formatAttractionsList(attrs []models.Attraction) string {
	if len(attrs) == 0 {
		return "Достопримечательности не найдены"
	}

	var builder strings.Builder
	for i, attr := range attrs {
		ratingText := ""
		if attr.Rating > 0 {
			ratingText = fmt.Sprintf(" (⭐ %.1f)", attr.Rating)
		}
		builder.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, attr.Name, ratingText))

		// Добавляем адрес, если есть
		if attr.Address != "" {
			builder.WriteString(fmt.Sprintf("   📍 %s\n", attr.Address))
		}

		// Добавляем описание, если есть
		if attr.Description != "" {
			builder.WriteString(fmt.Sprintf("   📝 %s\n", attr.Description))
		}

		builder.WriteString("\n")
	}
	return builder.String()
}

func formatAttractionDetail(detail models.AttractionDetail) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("🏛️ *%s*\n\n", detail.Name))

	if detail.Address != "" {
		builder.WriteString(fmt.Sprintf("📍 *Адрес:* %s\n", detail.Address))
	}

	if detail.City != "" {
		builder.WriteString(fmt.Sprintf("🏙️ *Город:* %s\n", detail.City))
	}

	if detail.FullDescription != "" {
		builder.WriteString(fmt.Sprintf("\n📖 *Описание:* %s\n", detail.FullDescription))
	} else if detail.Description != "" {
		builder.WriteString(fmt.Sprintf("\n📖 *Описание:* %s\n", detail.Description))
	}

	if detail.WorkingHours != "" {
		builder.WriteString(fmt.Sprintf("🕒 *Часы работы:* %s\n", detail.WorkingHours))
	}

	if detail.Phone != "" {
		builder.WriteString(fmt.Sprintf("📞 *Телефон:* %s\n", detail.Phone))
	}

	if detail.Website != "" {
		builder.WriteString(fmt.Sprintf("🌐 *Сайт:* %s\n", detail.Website))
	}

	if detail.Cost != "" {
		builder.WriteString(fmt.Sprintf("💵 *Стоимость:* %s\n", detail.Cost))
	}

	if detail.Rating > 0 {
		builder.WriteString(fmt.Sprintf("\n⭐ *Рейтинг:* %.1f/5\n", detail.Rating))
	}

	// Добавляем фото, если есть
	if detail.MainPhotoURL != "" {
		builder.WriteString(fmt.Sprintf("\n📸 [Фото](%s)", detail.MainPhotoURL))
	}

	return builder.String()
}

func createAttractionsKeyboard(attrs []models.Attraction) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, attr := range attrs {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			attr.Name,
			"attraction_"+strconv.Itoa(attr.ID),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
