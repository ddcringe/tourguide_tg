package models

//упрощенная модель достопримечательности
type Attraction struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	City         string  `json:"city"`
	Address      string  `json:"address"`
	Description  string  `json:"description_short"`
	Rating       float64 `json:"average_rating"`
	MainPhotoURL string  `json:"main_photo_url"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

//детальная информация о достопримечательности
type AttractionDetail struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	City            string   `json:"city"`
	Address         string   `json:"address"`
	Description     string   `json:"description_short"`
	FullDescription string   `json:"description"`
	WorkingHours    string   `json:"working_hours"`
	Phone           string   `json:"phone_number"`
	Website         string   `json:"website"`
	Cost            string   `json:"cost"`
	Rating          float64  `json:"average_rating"`
	MainPhotoURL    string   `json:"main_photo_url"`
	Latitude        float64  `json:"latitude"`
	Longitude       float64  `json:"longitude"`
	Photos          []string `json:"additional_photos"`
}
type CityRequest struct {
	City string `json:"city"`
}

type LocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    int     `json:"radius,omitempty"` // в метрах
}
type APIResponse struct {
	Count    int          `json:"count"`
	Next     string       `json:"next"`
	Previous string       `json:"previous"`
	Results  []Attraction `json:"results"`
}

type CityAPIResponse struct {
	Attractions []Attraction `json:"attractions"`
	City        string       `json:"city"`
	Count       int          `json:"count"`
}
type MapAPIResponse struct {
	Attractions []Attraction `json:"attractions"`
	Count       int          `json:"count"`
	Radius      float64      `json:"radius"`
}
