// handlers/handlers.go
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

// Добавим тип для определения типа поиска
type SearchType int

const (
	SearchTypeCity SearchType = iota
	SearchTypeLocation
)

// Обновим структуру для хранения состояния пагинации
type PaginationState struct {
	Type        SearchType
	City        string
	Location    *tgbotapi.Location
	Attractions []models.Attraction
	Page        int
	TotalPages  int
}

// Глобальная map для хранения состояний пагинации по chatID
var paginationStates = make(map[int64]*PaginationState)

func HandleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	if update.Message.Text == "/start" {
		msg.Text = "Привет! Я помогу найти интересные достопримечательности.\n\n Отправь мне название города (например: \"Москва\", \"Санкт-Петербург\")\n🗺️ Или отправь свою геолокацию для поиска рядом с тобой"
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButtonLocation(" Отправить геолокацию"),
			),
		)
	} else {
		// Обрабатываем как название города
		go HandleCity(bot, update)
	}

	bot.Send(msg)
}

// обрабатывает поиск достопримечательностей по городу
func HandleCity(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	cityName := update.Message.Text

	// Получаем достопримечательности по городу через API
	attractions, err := api.GetAttractionsByCity(cityName)
	if err != nil {
		log.Printf("Ошибка при запросе к API: %v", err)
		msg.Text = " Ошибка при поиске достопримечательностей. Попробуйте позже."
		bot.Send(msg)
		return
	}

	if len(attractions) == 0 {
		msg.Text = fmt.Sprintf("🏙️ В городе \"%s\" не найдено достопримечательностей \nПопробуйте другой город или проверьте написание.", cityName)
		bot.Send(msg)
		return
	}

	// Сохраняем состояние пагинации
	pageSize := 5
	totalPages := (len(attractions) + pageSize - 1) / pageSize

	paginationStates[update.Message.Chat.ID] = &PaginationState{
		Type:        SearchTypeCity,
		City:        cityName,
		Location:    nil,
		Attractions: attractions,
		Page:        0,
		TotalPages:  totalPages,
	}

	// Отправляем первую страницу
	sendAttractionsPage(bot, update.Message.Chat.ID, 0)
}

// обрабатывает сообщения с геолокацией
func HandleLocation(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	// Получаем достопримечательности вокруг локации
	attractions, err := api.GetAttractionsByLocation(
		update.Message.Location.Latitude,
		update.Message.Location.Longitude,
		0.01,
	)

	if err != nil {
		log.Printf("Ошибка при запросе геолокации: %v", err)
		msg.Text = " Ошибка при поиске достопримечательностей по геолокации."
		bot.Send(msg)
		return
	}

	if len(attractions) == 0 {
		msg.Text = " Рядом с вами не найдено достопримечательностей \nПопробуйте увеличить радиус поиска или отправьте название города."
		bot.Send(msg)
		return
	}

	// Сохраняем состояние пагинации
	pageSize := 5
	totalPages := (len(attractions) + pageSize - 1) / pageSize

	// Сохраняем копию локации
	locationCopy := &tgbotapi.Location{
		Latitude:  update.Message.Location.Latitude,
		Longitude: update.Message.Location.Longitude,
	}

	paginationStates[update.Message.Chat.ID] = &PaginationState{
		Type:        SearchTypeLocation,
		City:        "",
		Location:    locationCopy,
		Attractions: attractions,
		Page:        0,
		TotalPages:  totalPages,
	}

	// Отправляем первую страницу
	sendAttractionsPage(bot, update.Message.Chat.ID, 0)
}

// отправляет страницу с достопримечательностями
func sendAttractionsPage(bot *tgbotapi.BotAPI, chatID int64, page int) {
	state, exists := paginationStates[chatID]
	if !exists || len(state.Attractions) == 0 {
		return
	}

	// Проверяем границы страницы
	if page < 0 {
		page = 0
	}
	if page >= state.TotalPages {
		page = state.TotalPages - 1
	}

	state.Page = page
	pageSize := 5
	start := page * pageSize
	end := start + pageSize
	if end > len(state.Attractions) {
		end = len(state.Attractions)
	}

	// Формируем заголовок сообщения в зависимости от типа поиска
	var header string
	if state.Type == SearchTypeCity {
		header = fmt.Sprintf(" Достопримечательности в %s (стр. %d/%d):\n\n",
			state.City, page+1, state.TotalPages)
	} else {
		header = fmt.Sprintf(" Достопримечательности рядом с вами (стр. %d/%d):\n\n",
			page+1, state.TotalPages)
	}

	// Формируем сообщение
	var builder strings.Builder
	builder.WriteString(header)

	for i := start; i < end; i++ {
		attr := state.Attractions[i]
		ratingText := ""
		if attr.Rating > 0 {
			ratingText = fmt.Sprintf(" ( %.1f)", attr.Rating)
		}

		builder.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, attr.Name, ratingText))

		if attr.Address != "" {
			builder.WriteString(fmt.Sprintf("    %s\n", truncateString(attr.Address, 50)))
		}

		if attr.Description != "" {
			builder.WriteString(fmt.Sprintf("    %s\n", truncateString(attr.Description, 50)))
		}

		builder.WriteString("\n")
	}

	// Создаем клавиатуру с пагинацией
	keyboard := createPaginationKeyboard(page, state.TotalPages, start, end)

	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// создает клавиатуру для пагинации
func createPaginationKeyboard(currentPage, totalPages, start, end int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Кнопки навигации
	var navButtons []tgbotapi.InlineKeyboardButton

	if currentPage > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("page_%d", currentPage-1)))
	}

	if currentPage < totalPages-1 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("page_%d", currentPage+1)))
	}

	if len(navButtons) > 0 {
		rows = append(rows, navButtons)
	}

	// Кнопки выбора достопримечательностей
	for i := start; i < end; i++ {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🏛️ %d", i+1),
			fmt.Sprintf("attraction_%d", i),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// обрабатывает callback-и от inline кнопок
func HandleCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	bot.Send(callback)

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "")

	data := update.CallbackQuery.Data

	if strings.HasPrefix(data, "page_") {
		// Обработка пагинации
		pageStr := strings.TrimPrefix(data, "page_")
		page, err := strconv.Atoi(pageStr)
		if err == nil {
			sendAttractionsPage(bot, update.CallbackQuery.Message.Chat.ID, page)
		}
		return
	}

	if strings.HasPrefix(data, "attraction_") {
		// Обработка выбора достопримечательности
		indexStr := strings.TrimPrefix(data, "attraction_")
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			msg.Text = "Ошибка выбора"
		} else {
			state, exists := paginationStates[update.CallbackQuery.Message.Chat.ID]
			if exists && index >= 0 && index < len(state.Attractions) {
				detail, err := api.GetAttractionDetail(state.Attractions[index].ID)
				if err != nil {
					msg.Text = " Ошибка при загрузке деталей"
				} else {
					msg.Text = formatAttractionDetail(detail)
					// Добавляем кнопку назад к списку
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("↩️ Назад к списку",
								fmt.Sprintf("page_%d", state.Page)),
						),
					)
				}
			} else {
				msg.Text = "Достопримечательность не найдена"
			}
		}
	}

	bot.Send(msg)
}

func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

// формирует детальное описание достопримечательности
func formatAttractionDetail(detail models.AttractionDetail) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf(" *%s*\n\n", detail.Name))

	if detail.Address != "" {
		builder.WriteString(fmt.Sprintf(" *Адрес:* %s\n", detail.Address))
	}

	if detail.City != "" {
		builder.WriteString(fmt.Sprintf("*Город:* %s\n", detail.City))
	}

	if detail.FullDescription != "" {
		builder.WriteString(fmt.Sprintf("\n*Описание:* %s\n", truncateString(detail.FullDescription, 200)))
	} else if detail.Description != "" {
		builder.WriteString(fmt.Sprintf("\n *Описание:* %s\n", truncateString(detail.Description, 200)))
	}

	if detail.WorkingHours != "" {
		builder.WriteString(fmt.Sprintf("*Часы работы:* %s\n", detail.WorkingHours))
	}

	if detail.Phone != "" {
		builder.WriteString(fmt.Sprintf(" *Телефон:* %s\n", detail.Phone))
	}

	if detail.Website != "" {
		builder.WriteString(fmt.Sprintf(" *Сайт:* %s\n", detail.Website))
	}

	if detail.Cost != "" {
		builder.WriteString(fmt.Sprintf(" *Стоимость:* %s\n", detail.Cost))
	}

	if detail.Rating > 0 {
		builder.WriteString(fmt.Sprintf("\n *Рейтинг:* %.1f/5\n", detail.Rating))
	}

	// Добавляем фото, если есть
	if detail.MainPhotoURL != "" {
		builder.WriteString(fmt.Sprintf("\n [Фото](%s)", detail.MainPhotoURL))
	}

	return builder.String()
}
