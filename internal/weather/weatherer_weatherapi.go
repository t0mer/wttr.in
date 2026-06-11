package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chubin/wttr.in/internal/domain"
)

// WeatherAPIConfig holds configuration for the WeatherAPI.com service.
type WeatherAPIConfig struct {
	APIKey  string `yaml:"apiKey"`
	BaseURL string `yaml:"baseUrl,omitempty"`
}

// WeatherAPIClient fetches weather from WeatherAPI.com.
type WeatherAPIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewWeatherAPIClient creates a new WeatherAPI client.
func NewWeatherAPIClient(cfg *WeatherAPIConfig) *WeatherAPIClient {
	base := "https://api.weatherapi.com/v1"
	if cfg.BaseURL != "" {
		base = cfg.BaseURL
	}
	return &WeatherAPIClient{
		apiKey:     cfg.APIKey,
		baseURL:    base,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ── WeatherAPI response types ───────────────────────────────────────────────

type wapiResponse struct {
	Current  wapiCurrent `json:"current"`
	Forecast struct {
		Forecastday []wapiForecastDay `json:"forecastday"`
	} `json:"forecast"`
}

type wapiCurrent struct {
	TempC      float64 `json:"temp_c"`
	TempF      float64 `json:"temp_f"`
	FeelslikeC float64 `json:"feelslike_c"`
	FeelslikeF float64 `json:"feelslike_f"`
	Humidity   int     `json:"humidity"`
	PrecipMM   float64 `json:"precip_mm"`
	PressureMB float64 `json:"pressure_mb"`
	UV         float64 `json:"uv"`
	WindKph    float64 `json:"wind_kph"`
	WindMph    float64 `json:"wind_mph"`
	WindDegree int     `json:"wind_degree"`
	WindDir    string  `json:"wind_dir"`
	Cloud      int     `json:"cloud"`
	VisKm      float64 `json:"vis_km"`
	Condition  struct {
		Code int    `json:"code"`
		Text string `json:"text"`
	} `json:"condition"`
}

type wapiForecastDay struct {
	Date  string `json:"date"`
	Astro struct {
		Sunrise          string `json:"sunrise"`
		Sunset           string `json:"sunset"`
		Moonrise         string `json:"moonrise"`
		Moonset          string `json:"moonset"`
		MoonPhase        string `json:"moon_phase"`
		MoonIllumination string `json:"moon_illumination"`
	} `json:"astro"`
	Day struct {
		MaxtempC float64 `json:"maxtemp_c"`
		MaxtempF float64 `json:"maxtemp_f"`
		MintempC float64 `json:"mintemp_c"`
		MintempF float64 `json:"mintemp_f"`
		AvgtempC float64 `json:"avgtemp_c"`
		AvgtempF float64 `json:"avgtemp_f"`
		UV       float64 `json:"uv"`
	} `json:"day"`
	Hour []wapiHour `json:"hour"`
}

type wapiHour struct {
	Time         string  `json:"time"` // "2023-01-01 00:00"
	TempC        float64 `json:"temp_c"`
	TempF        float64 `json:"temp_f"`
	FeelslikeC   float64 `json:"feelslike_c"`
	FeelslikeF   float64 `json:"feelslike_f"`
	Humidity     int     `json:"humidity"`
	PrecipMM     float64 `json:"precip_mm"`
	ChanceOfRain int     `json:"chance_of_rain"`
	ChanceOfSnow int     `json:"chance_of_snow"`
	WindKph      float64 `json:"wind_kph"`
	WindMph      float64 `json:"wind_mph"`
	WindDegree   int     `json:"wind_degree"`
	WindDir      string  `json:"wind_dir"`
	GustKph      float64 `json:"gust_kph"`
	PressureMB   float64 `json:"pressure_mb"`
	Cloud        int     `json:"cloud"`
	VisKm        float64 `json:"vis_km"`
	UV           float64 `json:"uv"`
	Condition    struct {
		Code int    `json:"code"`
		Text string `json:"text"`
	} `json:"condition"`
}

// GetWeather fetches a 3-day forecast from WeatherAPI and returns WWO j1 JSON.
func (c *WeatherAPIClient) GetWeather(lat, lon float64, lang string) ([]byte, error) {
	url := fmt.Sprintf(
		"%s/forecast.json?key=%s&q=%.6f,%.6f&days=3&aqi=no&alerts=no&lang=%s",
		c.baseURL, c.apiKey, lat, lon, wapiNormalizeLang(lang),
	)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("weatherapi request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("weatherapi read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weatherapi status %d: %s", resp.StatusCode, body)
	}

	var wr wapiResponse
	if err := json.Unmarshal(body, &wr); err != nil {
		return nil, fmt.Errorf("weatherapi unmarshal: %w", err)
	}

	return json.Marshal(c.convert(wr, lat, lon))
}

func (c *WeatherAPIClient) convert(r wapiResponse, lat, lon float64) domain.Weather {
	cur := r.Current
	wwoCode := mapCode(weatherAPIToWWO, cur.Condition.Code)

	return domain.Weather{
		Request: []domain.Request{{
			Type:  "LatLon",
			Query: fmt.Sprintf("Lat %.6f and Lon %.6f", lat, lon),
		}},
		CurrentCondition: []domain.CurrentCondition{{
			TempC:          ftoa0(cur.TempC),
			TempF:          ftoa0(cur.TempF),
			FeelsLikeC:     ftoa0(cur.FeelslikeC),
			FeelsLikeF:     ftoa0(cur.FeelslikeF),
			Humidity:       itoa(cur.Humidity),
			PrecipMM:       ftoa(cur.PrecipMM),
			PrecipInches:   ftoa(cur.PrecipMM * 0.0393701),
			Cloudcover:     itoa(cur.Cloud),
			WindspeedKmph:  ftoa0(cur.WindKph),
			WindspeedMiles: ftoa0(cur.WindMph),
			WinddirDegree:  itoa(cur.WindDegree),
			Winddir16Point: cur.WindDir,
			Pressure:       ftoa0(cur.PressureMB),
			PressureInches: ftoa(mbToInHg(cur.PressureMB)),
			Visibility:     ftoa0(cur.VisKm),
			VisibilityMiles: ftoa(kmToMiles(cur.VisKm)),
			WeatherCode:    itoa(wwoCode),
			WeatherDesc:    []domain.ValueItem{{Value: cur.Condition.Text}},
			UVIndex:        ftoa0(cur.UV),
		}},
		Weather: c.convertDays(r.Forecast.Forecastday),
	}
}

func (c *WeatherAPIClient) convertDays(days []wapiForecastDay) []domain.WeatherDay {
	result := make([]domain.WeatherDay, 0, len(days))
	for _, d := range days {
		day := domain.WeatherDay{
			Date:     d.Date,
			MaxTempC: ftoa0(d.Day.MaxtempC),
			MaxTempF: ftoa0(d.Day.MaxtempF),
			MinTempC: ftoa0(d.Day.MintempC),
			MinTempF: ftoa0(d.Day.MintempF),
			AvgTempC: ftoa0(d.Day.AvgtempC),
			AvgTempF: ftoa0(d.Day.AvgtempF),
			UVIndex:  ftoa0(d.Day.UV),
			Astronomy: []domain.Astronomy{{
				Sunrise:          d.Astro.Sunrise,
				Sunset:           d.Astro.Sunset,
				Moonrise:         d.Astro.Moonrise,
				Moonset:          d.Astro.Moonset,
				MoonPhase:        d.Astro.MoonPhase,
				MoonIllumination: d.Astro.MoonIllumination,
			}},
		}

		// WeatherAPI provides 24 hourly entries; keep only the 8 canonical 3h slots.
		for _, h := range d.Hour {
			hour, err := wapiParseHour(h.Time)
			if err != nil || hour%3 != 0 {
				continue
			}
			wwoCode := mapCode(weatherAPIToWWO, h.Condition.Code)
			day.Hourly = append(day.Hourly, domain.Hourly{
				Time:           itoa(hour * 100),
				TempC:          ftoa0(h.TempC),
				TempF:          ftoa0(h.TempF),
				FeelsLikeC:     ftoa0(h.FeelslikeC),
				FeelsLikeF:     ftoa0(h.FeelslikeF),
				PrecipMM:       ftoa(h.PrecipMM),
				PrecipInches:   ftoa(h.PrecipMM * 0.0393701),
				ChanceOfRain:   itoa(h.ChanceOfRain),
				ChanceOfSnow:   itoa(h.ChanceOfSnow),
				WeatherCode:    itoa(wwoCode),
				WindspeedKmph:  ftoa0(h.WindKph),
				WindspeedMiles: ftoa0(h.WindMph),
				WinddirDegree:  itoa(h.WindDegree),
				Winddir16Point: h.WindDir,
				WindGustKmph:   ftoa0(h.GustKph),
				WindGustMiles:  ftoa(kmphToMph(h.GustKph)),
				Humidity:       itoa(h.Humidity),
				Visibility:     ftoa0(h.VisKm),
				VisibilityMiles: ftoa(kmToMiles(h.VisKm)),
				Cloudcover:     itoa(h.Cloud),
				Pressure:       ftoa0(h.PressureMB),
				PressureInches: ftoa(mbToInHg(h.PressureMB)),
				UVIndex:        ftoa0(h.UV),
				WeatherDesc:    []domain.ValueItem{{Value: h.Condition.Text}},
			})
		}

		result = append(result, day)
	}
	return result
}

// wapiParseHour extracts the hour (0-23) from "YYYY-MM-DD HH:MM".
func wapiParseHour(s string) (int, error) {
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		return 0, err
	}
	return t.Hour(), nil
}

// wapiNormalizeLang passes lang through; WeatherAPI uses standard lang codes.
func wapiNormalizeLang(lang string) string {
	if lang == "" {
		return "en"
	}
	return lang
}
