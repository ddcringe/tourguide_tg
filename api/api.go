package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"tg-bot/models"
)

var baseURL = os.Getenv("API_URL")

type CityRequest struct {
	City string `json:"city"`
	//Tags string `json:"tags,omitempty"`
}

// получает достопримечательности по городу
func GetAttractionsByCity(city string) ([]models.Attraction, error) {
	// Создаем запрос с городом
	cityReq := CityRequest{
		City: city,
		//Tags: tags,
	}

	// Конвертируем в JSON
	jsonData, err := json.Marshal(cityReq)
	if err != nil {
		return nil, err
	}

	// Создаем POST запрос
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/cities/", baseURL), bytes.NewBuffer(jsonData))
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

	var attractions []models.Attraction
	err = json.Unmarshal(body, &attractions)
	if err != nil {
		return nil, err
	}

	return attractions, nil
}

// получает достопримечательности по координатам
func GetAttractionsByLocation(lat, lon float64, radius float64) ([]models.Attraction, error) {
	// Формируем URL с query-параметрами
	url := fmt.Sprintf("%s/map/attractions/?lat=%f&lng=%f&radius=%f",
		baseURL, lat, lon, radius)

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

	var attractions []models.Attraction
	err = json.Unmarshal(body, &attractions)
	if err != nil {
		return nil, err
	}

	return attractions, nil
}

// получает детальную информацию о достопримечательности
func GetAttractionDetail(id int) (models.AttractionDetail, error) {
	var detail models.AttractionDetail

	resp, err := http.Get(fmt.Sprintf("%s/attractions/%d/", baseURL, id))
	if err != nil {
		return detail, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return detail, err
	}

	err = json.Unmarshal(body, &detail)
	return detail, err
}
