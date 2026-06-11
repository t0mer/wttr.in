package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chubin/wttr.in/internal/domain"
)

// OpenMeteoConfig holds optional configuration for the Open-Meteo API (no API key required).
type OpenMeteoConfig struct {
	BaseURL string `yaml:"baseUrl,omitempty"`
}

// OpenMeteoClient fetches weather from the free Open-Meteo API.
type OpenMeteoClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOpenMeteoClient creates a new Open-Meteo client. cfg may be nil.
func NewOpenMeteoClient(cfg *OpenMeteoConfig) *OpenMeteoClient {
	base := "https://api.open-meteo.com/v1/forecast"
	if cfg != nil && cfg.BaseURL != "" {
		base = cfg.BaseURL
	}
	return &OpenMeteoClient{
		baseURL:    base,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ── Open-Meteo response types ──────────────────────────────────────────────

type openMeteoResponse struct {
	Current openMeteoCurrent `json:"current"`
	Hourly  openMeteoHourly  `json:"hourly"`
	Daily   openMeteoDaily   `json:"daily"`
}

type openMeteoCurrent struct {
	Temperature     float64 `json:"temperature_2m"`
	ApparentTemp    float64 `json:"apparent_temperature"`
	RelHumidity     int     `json:"relative_humidity_2m"`
	Precipitation   float64 `json:"precipitation"`
	WeatherCode     int     `json:"weather_code"`
	CloudCover      int     `json:"cloud_cover"`
	WindSpeed       float64 `json:"wind_speed_10m"`
	WindDirection   float64 `json:"wind_direction_10m"`
	WindGusts       float64 `json:"wind_gusts_10m"`
	SurfacePressure float64 `json:"surface_pressure"`
	Visibility      float64 `json:"visibility"` // meters
}

type openMeteoHourly struct {
	Time          []string  `json:"time"`
	Temperature   []float64 `json:"temperature_2m"`
	ApparentTemp  []float64 `json:"apparent_temperature"`
	Precipitation []float64 `json:"precipitation"`
	WeatherCode   []int     `json:"weather_code"`
	WindSpeed     []float64 `json:"wind_speed_10m"`
	WindDirection []float64 `json:"wind_direction_10m"`
	Humidity      []int     `json:"relative_humidity_2m"`
	Visibility    []float64 `json:"visibility"`
	CloudCover    []int     `json:"cloud_cover"`
	UVIndex       []float64 `json:"uv_index"`
	WindGusts     []float64 `json:"wind_gusts_10m"`
	PrecipProb    []int     `json:"precipitation_probability"`
}

type openMeteoDaily struct {
	Time        []string  `json:"time"`
	WeatherCode []int     `json:"weather_code"`
	MaxTemp     []float64 `json:"temperature_2m_max"`
	MinTemp     []float64 `json:"temperature_2m_min"`
	Sunrise     []string  `json:"sunrise"`
	Sunset      []string  `json:"sunset"`
}

// GetWeather fetches weather from Open-Meteo and returns WWO j1-compatible JSON.
func (c *OpenMeteoClient) GetWeather(lat, lon float64, lang string) ([]byte, error) {
	url := fmt.Sprintf(
		"%s?latitude=%.6f&longitude=%.6f"+
			"&current=temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m,wind_gusts_10m,surface_pressure,visibility"+
			"&hourly=temperature_2m,apparent_temperature,precipitation,weather_code,wind_speed_10m,wind_direction_10m,relative_humidity_2m,visibility,cloud_cover,uv_index,wind_gusts_10m,precipitation_probability"+
			"&daily=weather_code,temperature_2m_max,temperature_2m_min,sunrise,sunset"+
			"&timezone=auto&wind_speed_unit=kmh&forecast_days=3",
		c.baseURL, lat, lon,
	)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("open-meteo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("open-meteo read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo status %d: %s", resp.StatusCode, body)
	}

	var omr openMeteoResponse
	if err := json.Unmarshal(body, &omr); err != nil {
		return nil, fmt.Errorf("open-meteo unmarshal: %w", err)
	}

	return json.Marshal(c.convert(omr, lat, lon))
}

func (c *OpenMeteoClient) convert(omr openMeteoResponse, lat, lon float64) domain.Weather {
	cur := omr.Current
	wwoCode := mapCode(wmoToWWO, cur.WeatherCode)
	visKm := cur.Visibility / 1000.0

	return domain.Weather{
		Request: []domain.Request{{
			Type:  "LatLon",
			Query: fmt.Sprintf("Lat %.6f and Lon %.6f", lat, lon),
		}},
		CurrentCondition: []domain.CurrentCondition{{
			TempC:          ftoa0(cur.Temperature),
			TempF:          ftoa0(celsiusToFahrenheit(cur.Temperature)),
			FeelsLikeC:     ftoa0(cur.ApparentTemp),
			FeelsLikeF:     ftoa0(celsiusToFahrenheit(cur.ApparentTemp)),
			Humidity:       itoa(cur.RelHumidity),
			PrecipMM:       ftoa(cur.Precipitation),
			PrecipInches:   ftoa(cur.Precipitation * 0.0393701),
			Cloudcover:     itoa(cur.CloudCover),
			WindspeedKmph:  ftoa0(cur.WindSpeed),
			WindspeedMiles: ftoa0(kmphToMph(cur.WindSpeed)),
			WinddirDegree:  ftoa0(cur.WindDirection),
			Winddir16Point: degreesToWindDir16(cur.WindDirection),
			Pressure:       ftoa0(cur.SurfacePressure),
			PressureInches: ftoa(mbToInHg(cur.SurfacePressure)),
			Visibility:     ftoa0(visKm),
			VisibilityMiles: ftoa(kmToMiles(visKm)),
			WeatherCode:    itoa(wwoCode),
			WeatherDesc:    []domain.ValueItem{{Value: wmoDesc(cur.WeatherCode)}},
			UVIndex:        "0",
		}},
		Weather: c.convertDays(omr),
	}
}

func (c *OpenMeteoClient) convertDays(omr openMeteoResponse) []domain.WeatherDay {
	days := make([]domain.WeatherDay, 0, len(omr.Daily.Time))
	h := omr.Hourly

	for i, date := range omr.Daily.Time {
		maxT := safeF64(omr.Daily.MaxTemp, i)
		minT := safeF64(omr.Daily.MinTemp, i)

		day := domain.WeatherDay{
			Date:     date,
			MaxTempC: ftoa0(maxT),
			MaxTempF: ftoa0(celsiusToFahrenheit(maxT)),
			MinTempC: ftoa0(minT),
			MinTempF: ftoa0(celsiusToFahrenheit(minT)),
			AvgTempC: ftoa0((maxT + minT) / 2),
			AvgTempF: ftoa0(celsiusToFahrenheit((maxT + minT) / 2)),
			UVIndex:  "0",
		}

		if i < len(omr.Daily.Sunrise) && i < len(omr.Daily.Sunset) {
			day.Astronomy = []domain.Astronomy{{
				Sunrise: openMeteoFormatTime(omr.Daily.Sunrise[i]),
				Sunset:  openMeteoFormatTime(omr.Daily.Sunset[i]),
			}}
		}

		// Daily hourly data is at indices [i*24 .. i*24+23]; take every 3rd hour.
		base := i * 24
		for _, slot := range []int{0, 3, 6, 9, 12, 15, 18, 21} {
			idx := base + slot
			if idx >= len(h.Time) {
				break
			}
			wwoCode := mapCode(wmoToWWO, safeInt(h.WeatherCode, idx))
			visKm := safeF64(h.Visibility, idx) / 1000.0
			temp := safeF64(h.Temperature, idx)
			feels := safeF64(h.ApparentTemp, idx)
			wind := safeF64(h.WindSpeed, idx)
			windDir := safeF64(h.WindDirection, idx)

			day.Hourly = append(day.Hourly, domain.Hourly{
				Time:           itoa(slot * 100),
				TempC:          ftoa0(temp),
				TempF:          ftoa0(celsiusToFahrenheit(temp)),
				FeelsLikeC:     ftoa0(feels),
				FeelsLikeF:     ftoa0(celsiusToFahrenheit(feels)),
				PrecipMM:       ftoa(safeF64(h.Precipitation, idx)),
				PrecipInches:   ftoa(safeF64(h.Precipitation, idx) * 0.0393701),
				WeatherCode:    itoa(wwoCode),
				WindspeedKmph:  ftoa0(wind),
				WindspeedMiles: ftoa0(kmphToMph(wind)),
				WinddirDegree:  itoa(int(windDir)),
				Winddir16Point: degreesToWindDir16(windDir),
				WindGustKmph:   ftoa0(safeF64(h.WindGusts, idx)),
				WindGustMiles:  ftoa(kmphToMph(safeF64(h.WindGusts, idx))),
				Humidity:       itoa(safeInt(h.Humidity, idx)),
				Visibility:     ftoa0(visKm),
				VisibilityMiles: ftoa(kmToMiles(visKm)),
				Cloudcover:     itoa(safeInt(h.CloudCover, idx)),
				UVIndex:        ftoa0(safeF64(h.UVIndex, idx)),
				ChanceOfRain:   itoa(safeInt(h.PrecipProb, idx)),
				WeatherDesc:    []domain.ValueItem{{Value: wmoDesc(safeInt(h.WeatherCode, idx))}},
			})
		}

		days = append(days, day)
	}
	return days
}

// openMeteoFormatTime converts "2023-01-01T06:30" → "06:30 AM".
func openMeteoFormatTime(s string) string {
	t, err := time.Parse("2006-01-02T15:04", s)
	if err != nil {
		return s
	}
	return t.Format("03:04 PM")
}

// wmoDesc returns an English description for a WMO weather interpretation code.
func wmoDesc(code int) string {
	descs := map[int]string{
		0: "Clear sky", 1: "Mainly clear", 2: "Partly cloudy", 3: "Overcast",
		45: "Fog", 48: "Depositing rime fog",
		51: "Light drizzle", 53: "Moderate drizzle", 55: "Dense drizzle",
		56: "Light freezing drizzle", 57: "Heavy freezing drizzle",
		61: "Slight rain", 63: "Moderate rain", 65: "Heavy rain",
		66: "Light freezing rain", 67: "Heavy freezing rain",
		71: "Slight snowfall", 73: "Moderate snowfall", 75: "Heavy snowfall", 77: "Snow grains",
		80: "Slight rain showers", 81: "Moderate rain showers", 82: "Violent rain showers",
		85: "Slight snow showers", 86: "Heavy snow showers",
		95: "Thunderstorm", 96: "Thunderstorm with slight hail", 99: "Thunderstorm with heavy hail",
	}
	if d, ok := descs[code]; ok {
		return d
	}
	return "Partly cloudy"
}
