package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"tg-bot/models"
	"unicode/utf8"
)

func cleanUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue
			}
		}
		v = append(v, r)
	}
	return string(v)
}

// GetAttractionsByCity получает достопримечательности по городу
func GetAttractionsByCity(city string) ([]models.Attraction, error) {
	// Создаем запрос с городом
	cityReq := models.CityRequest{
		City: city,
	}

	// Конвертируем в JSON
	jsonData, err := json.Marshal(cityReq)
	if err != nil {
		return nil, err
	}

	// Создаем POST запрос
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/cities/", "https://tourguideyar.ru/api"), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Парсим ответ
	var cityResponse models.CityAPIResponse
	err = json.Unmarshal(body, &cityResponse)
	if err != nil {
		// Попробуем альтернативный формат ответа
		var altResponse models.APIResponse
		if altErr := json.Unmarshal(body, &altResponse); altErr == nil {
			return altResponse.Results, nil
		}
		return nil, err
	}

	return cityResponse.Attractions, nil
}

func GetAttractionsByLocation(lat, lon float64, radius float64) ([]models.Attraction, error) {
	// Формируем URL с query-параметрами
	url := fmt.Sprintf("%s/map/attractions/?lat=%f&lng=%f&radius=%f",
		"https://tourguideyar.ru/api", lat, lon, radius)

	// Создаем GET-запрос
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Парсим ответ
	var mapResponse models.MapAPIResponse
	err = json.Unmarshal(body, &mapResponse)
	if err != nil {
		// Попробуем альтернативный формат ответа
		var altResponse models.APIResponse
		if altErr := json.Unmarshal(body, &altResponse); altErr == nil {
			return altResponse.Results, nil
		}
		return nil, err
	}

	return mapResponse.Attractions, nil
}

func GetAttractionDetail(id int) (models.AttractionDetail, error) {
	var detail models.AttractionDetail

	resp, err := http.Get(fmt.Sprintf("%s/attractions/%d/", "https://tourguideyar.ru/api", id))
	if err != nil {
		return detail, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return detail, err
	}

	err = json.Unmarshal(body, &detail)
	detail.Name = cleanUTF8(detail.Name)
	detail.Address = cleanUTF8(detail.Address)
	detail.City = cleanUTF8(detail.City)
	detail.Description = cleanUTF8(detail.Description)
	detail.FullDescription = cleanUTF8(detail.FullDescription)
	detail.WorkingHours = cleanUTF8(detail.WorkingHours)
	detail.Phone = cleanUTF8(detail.Phone)
	detail.Website = cleanUTF8(detail.Website)
	detail.Cost = cleanUTF8(detail.Cost)
	detail.MainPhotoURL = cleanUTF8(detail.MainPhotoURL)
	return detail, err
}
