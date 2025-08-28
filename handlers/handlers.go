// handlers/handlers.go
package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"tg-bot/api"
	"tg-bot/models"
	"unicode/utf8"

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

	// Очищаем название города
	cityName := cleanUTF8(update.Message.Text)

	// Получаем достопримечательности по городу через API
	attractions, err := api.GetAttractionsByCity(cityName)
	if err != nil {
		log.Printf("Ошибка при запросе к API: %v", err)
		msg.Text = "❌ Ошибка при поиске достопримечательностей. Попробуйте позже."
		bot.Send(msg)
		return
	}

	// Очищаем полученные данные
	for i := range attractions {
		attractions[i].Name = cleanUTF8(attractions[i].Name)
		attractions[i].Address = cleanUTF8(attractions[i].Address)
		attractions[i].Description = cleanUTF8(attractions[i].Description)
		attractions[i].City = cleanUTF8(attractions[i].City)
	}

	if len(attractions) == 0 {
		msg.Text = safeFormat("🏙️ В городе \"%s\" не найдено достопримечательностей 😢\nПопробуйте другой город или проверьте написание.", cityName)
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
	for i := range attractions {
		attractions[i].Name = cleanUTF8(attractions[i].Name)
		attractions[i].Address = cleanUTF8(attractions[i].Address)
		attractions[i].Description = cleanUTF8(attractions[i].Description)
		attractions[i].City = cleanUTF8(attractions[i].City)
	}

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
func cleanUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// Если строка содержит невалидные UTF8 символы, очищаем их
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue // Пропускаем невалидный символ
			}
		}
		v = append(v, r)
	}
	return string(v)
}

// Функция для безопасного форматирования строки
func safeFormat(format string, args ...interface{}) string {
	// Очищаем все аргументы
	cleanArgs := make([]interface{}, len(args))
	for i, arg := range args {
		if s, ok := arg.(string); ok {
			cleanArgs[i] = cleanUTF8(s)
		} else {
			cleanArgs[i] = arg
		}
	}
	return fmt.Sprintf(format, cleanArgs...)
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
		header = safeFormat("🏙️ Достопримечательности в %s (стр. %d/%d):\n\n",
			state.City, page+1, state.TotalPages)
	} else {
		header = safeFormat("📍 Достопримечательности рядом с вами (стр. %d/%d):\n\n",
			page+1, state.TotalPages)
	}

	// Формируем сообщение
	var builder strings.Builder
	builder.WriteString(header)

	for i := start; i < end; i++ {
		attr := state.Attractions[i]

		// Очищаем все текстовые поля
		cleanName := cleanUTF8(attr.Name)
		cleanAddress := cleanUTF8(attr.Address)
		cleanDescription := cleanUTF8(attr.Description)

		ratingText := ""
		if attr.Rating > 0 {
			ratingText = safeFormat(" (⭐ %.1f)", attr.Rating)
		}

		builder.WriteString(safeFormat("%d. %s%s\n", i+1, cleanName, ratingText))

		if cleanAddress != "" {
			builder.WriteString(safeFormat("   📍 %s\n", truncateString(cleanAddress, 50)))
		}

		if cleanDescription != "" {
			builder.WriteString(safeFormat("   📝 %s\n", truncateString(cleanDescription, 50)))
		}

		builder.WriteString("\n")
	}

	// Создаем клавиатуру с пагинацией
	keyboard := createPaginationKeyboard(page, state.TotalPages, start, end)

	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML" // Используем HTML parse mode для лучшей совместимости
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

	// Очищаем все текстовые поля
	cleanName := cleanUTF8(detail.Name)
	cleanAddress := cleanUTF8(detail.Address)
	cleanCity := cleanUTF8(detail.City)
	cleanFullDescription := cleanUTF8(detail.FullDescription)
	cleanDescription := cleanUTF8(detail.Description)
	cleanWorkingHours := cleanUTF8(detail.WorkingHours)
	cleanPhone := cleanUTF8(detail.Phone)
	cleanWebsite := cleanUTF8(detail.Website)
	cleanCost := cleanUTF8(detail.Cost)

	builder.WriteString(safeFormat("<b>🏛️ %s</b>\n\n", cleanName))

	if cleanAddress != "" {
		builder.WriteString(safeFormat("📍 <b>Адрес:</b> %s\n", cleanAddress))
	}

	if cleanCity != "" {
		builder.WriteString(safeFormat("🏙️ <b>Город:</b> %s\n", cleanCity))
	}

	if cleanFullDescription != "" {
		builder.WriteString(safeFormat("\n📖 <b>Описание:</b> %s\n", truncateString(cleanFullDescription, 200)))
	} else if cleanDescription != "" {
		builder.WriteString(safeFormat("\n📖 <b>Описание:</b> %s\n", truncateString(cleanDescription, 200)))
	}

	if cleanWorkingHours != "" {
		builder.WriteString(safeFormat("🕒 <b>Часы работы:</b> %s\n", cleanWorkingHours))
	}

	if cleanPhone != "" {
		builder.WriteString(safeFormat("📞 <b>Телефон:</b> %s\n", cleanPhone))
	}

	if cleanWebsite != "" {
		builder.WriteString(safeFormat("🌐 <b>Сайт:</b> %s\n", cleanWebsite))
	}

	if cleanCost != "" {
		builder.WriteString(safeFormat("💵 <b>Стоимость:</b> %s\n", cleanCost))
	}

	if detail.Rating > 0 {
		builder.WriteString(safeFormat("\n⭐ <b>Рейтинг:</b> %.1f/5\n", detail.Rating))
	}

	// Добавляем фото, если есть
	if detail.MainPhotoURL != "" {
		cleanPhotoURL := cleanUTF8(detail.MainPhotoURL)
		builder.WriteString(safeFormat("\n📸 <a href=\"%s\">Фото</a>", cleanPhotoURL))
	}

	return builder.String()
}
