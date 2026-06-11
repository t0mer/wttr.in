package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chubin/wttr.in/internal/domain"
)

// OWMConfig holds configuration for the OpenWeatherMap API.
type OWMConfig struct {
	APIKey  string `yaml:"apiKey"`
	BaseURL string `yaml:"baseUrl,omitempty"`
}

// OWMClient fetches weather from the OpenWeatherMap API.
type OWMClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOWMClient creates a new OpenWeatherMap client.
func NewOWMClient(cfg *OWMConfig) *OWMClient {
	base := "https://api.openweathermap.org/data/2.5"
	if cfg.BaseURL != "" {
		base = cfg.BaseURL
	}
	return &OWMClient{
		apiKey:     cfg.APIKey,
		baseURL:    base,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ── OWM response types ──────────────────────────────────────────────────────

type owmCurrentResp struct {
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Humidity  int     `json:"humidity"`
		Pressure  int     `json:"pressure"`
	} `json:"main"`
	Weather []struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"weather"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Wind struct {
		Speed float64 `json:"speed"` // m/s
		Deg   float64 `json:"deg"`
	} `json:"wind"`
	Visibility int `json:"visibility"` // meters
	Rain       struct {
		OneH float64 `json:"1h"`
	} `json:"rain"`
	Snow struct {
		OneH float64 `json:"1h"`
	} `json:"snow"`
}

type owmForecastResp struct {
	List []owmForecastItem `json:"list"`
}

type owmForecastItem struct {
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMax   float64 `json:"temp_max"`
		TempMin   float64 `json:"temp_min"`
		Humidity  int     `json:"humidity"`
		Pressure  int     `json:"pressure"`
	} `json:"main"`
	Weather []struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"weather"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Wind struct {
		Speed float64 `json:"speed"` // m/s
		Deg   float64 `json:"deg"`
		Gust  float64 `json:"gust"`
	} `json:"wind"`
	Visibility int     `json:"visibility"` // meters
	Pop        float64 `json:"pop"`        // probability of precipitation 0–1
	Rain       struct {
		ThreeH float64 `json:"3h"`
	} `json:"rain"`
	Snow struct {
		ThreeH float64 `json:"3h"`
	} `json:"snow"`
	DtTxt string `json:"dt_txt"` // "2023-01-01 15:00:00"
}

// GetWeather fetches current + 5-day forecast from OWM and returns WWO j1 JSON.
func (c *OWMClient) GetWeather(lat, lon float64, lang string) ([]byte, error) {
	owmLang := owmNormalizeLang(lang)

	curBody, err := c.get(fmt.Sprintf(
		"%s/weather?lat=%.6f&lon=%.6f&appid=%s&units=metric&lang=%s",
		c.baseURL, lat, lon, c.apiKey, owmLang,
	))
	if err != nil {
		return nil, err
	}

	var cur owmCurrentResp
	if err := json.Unmarshal(curBody, &cur); err != nil {
		return nil, fmt.Errorf("owm current unmarshal: %w", err)
	}

	fcBody, err := c.get(fmt.Sprintf(
		"%s/forecast?lat=%.6f&lon=%.6f&appid=%s&units=metric&lang=%s&cnt=40",
		c.baseURL, lat, lon, c.apiKey, owmLang,
	))
	if err != nil {
		return nil, err
	}

	var fc owmForecastResp
	if err := json.Unmarshal(fcBody, &fc); err != nil {
		return nil, fmt.Errorf("owm forecast unmarshal: %w", err)
	}

	return json.Marshal(c.convert(cur, fc, lat, lon))
}

func (c *OWMClient) get(url string) ([]byte, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("owm request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("owm read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("owm status %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

func (c *OWMClient) convert(cur owmCurrentResp, fc owmForecastResp, lat, lon float64) domain.Weather {
	curCode := 116
	curDesc := "Partly cloudy"
	if len(cur.Weather) > 0 {
		curCode = mapCode(owmToWWO, cur.Weather[0].ID)
		curDesc = capitalize(cur.Weather[0].Description)
	}
	windKmph := mpsToKmph(cur.Wind.Speed)
	visKm := float64(cur.Visibility) / 1000.0
	precip := cur.Rain.OneH + cur.Snow.OneH

	return domain.Weather{
		Request: []domain.Request{{
			Type:  "LatLon",
			Query: fmt.Sprintf("Lat %.6f and Lon %.6f", lat, lon),
		}},
		CurrentCondition: []domain.CurrentCondition{{
			TempC:          ftoa0(cur.Main.Temp),
			TempF:          ftoa0(celsiusToFahrenheit(cur.Main.Temp)),
			FeelsLikeC:     ftoa0(cur.Main.FeelsLike),
			FeelsLikeF:     ftoa0(celsiusToFahrenheit(cur.Main.FeelsLike)),
			Humidity:       itoa(cur.Main.Humidity),
			PrecipMM:       ftoa(precip),
			PrecipInches:   ftoa(precip * 0.0393701),
			Cloudcover:     itoa(cur.Clouds.All),
			WindspeedKmph:  ftoa0(windKmph),
			WindspeedMiles: ftoa0(kmphToMph(windKmph)),
			WinddirDegree:  ftoa0(cur.Wind.Deg),
			Winddir16Point: degreesToWindDir16(cur.Wind.Deg),
			Pressure:       itoa(cur.Main.Pressure),
			PressureInches: ftoa(mbToInHg(float64(cur.Main.Pressure))),
			Visibility:     ftoa0(visKm),
			VisibilityMiles: ftoa(kmToMiles(visKm)),
			WeatherCode:    itoa(curCode),
			WeatherDesc:    []domain.ValueItem{{Value: curDesc}},
			UVIndex:        "0",
		}},
		Weather: c.convertDays(fc),
	}
}

func (c *OWMClient) convertDays(fc owmForecastResp) []domain.WeatherDay {
	// Group 3h forecast items by date; maintain insertion order.
	type dayEntry struct {
		items []owmForecastItem
	}
	byDate := make(map[string]*dayEntry)
	var dateOrder []string

	for _, item := range fc.List {
		if len(item.DtTxt) < 10 {
			continue
		}
		date := item.DtTxt[:10]
		if _, exists := byDate[date]; !exists {
			byDate[date] = &dayEntry{}
			dateOrder = append(dateOrder, date)
		}
		byDate[date].items = append(byDate[date].items, item)
	}

	days := make([]domain.WeatherDay, 0, len(dateOrder))
	for _, date := range dateOrder {
		items := byDate[date].items

		maxTemp := items[0].Main.TempMax
		minTemp := items[0].Main.TempMin
		for _, it := range items {
			if it.Main.TempMax > maxTemp {
				maxTemp = it.Main.TempMax
			}
			if it.Main.TempMin < minTemp {
				minTemp = it.Main.TempMin
			}
		}

		day := domain.WeatherDay{
			Date:     date,
			MaxTempC: ftoa0(maxTemp),
			MaxTempF: ftoa0(celsiusToFahrenheit(maxTemp)),
			MinTempC: ftoa0(minTemp),
			MinTempF: ftoa0(celsiusToFahrenheit(minTemp)),
			AvgTempC: ftoa0((maxTemp + minTemp) / 2),
			AvgTempF: ftoa0(celsiusToFahrenheit((maxTemp + minTemp) / 2)),
			UVIndex:  "0",
		}

		for _, item := range items {
			t, err := time.Parse("2006-01-02 15:04:05", item.DtTxt)
			if err != nil {
				continue
			}

			wwoCode := 116
			desc := "Partly cloudy"
			if len(item.Weather) > 0 {
				wwoCode = mapCode(owmToWWO, item.Weather[0].ID)
				desc = capitalize(item.Weather[0].Description)
			}

			windKmph := mpsToKmph(item.Wind.Speed)
			gustKmph := mpsToKmph(item.Wind.Gust)
			visKm := float64(item.Visibility) / 1000.0
			precip := item.Rain.ThreeH + item.Snow.ThreeH
			chanceRain := int(item.Pop * 100)

			day.Hourly = append(day.Hourly, domain.Hourly{
				Time:           itoa(t.Hour() * 100),
				TempC:          ftoa0(item.Main.Temp),
				TempF:          ftoa0(celsiusToFahrenheit(item.Main.Temp)),
				FeelsLikeC:     ftoa0(item.Main.FeelsLike),
				FeelsLikeF:     ftoa0(celsiusToFahrenheit(item.Main.FeelsLike)),
				PrecipMM:       ftoa(precip),
				PrecipInches:   ftoa(precip * 0.0393701),
				WeatherCode:    itoa(wwoCode),
				WindspeedKmph:  ftoa0(windKmph),
				WindspeedMiles: ftoa0(kmphToMph(windKmph)),
				WinddirDegree:  ftoa0(item.Wind.Deg),
				Winddir16Point: degreesToWindDir16(item.Wind.Deg),
				WindGustKmph:   ftoa0(gustKmph),
				WindGustMiles:  ftoa(kmphToMph(gustKmph)),
				Humidity:       itoa(item.Main.Humidity),
				Visibility:     ftoa0(visKm),
				VisibilityMiles: ftoa(kmToMiles(visKm)),
				Cloudcover:     itoa(item.Clouds.All),
				Pressure:       itoa(item.Main.Pressure),
				PressureInches: ftoa(mbToInHg(float64(item.Main.Pressure))),
				UVIndex:        "0",
				ChanceOfRain:   itoa(chanceRain),
				WeatherDesc:    []domain.ValueItem{{Value: desc}},
			})
		}

		days = append(days, day)
	}
	return days
}

// owmNormalizeLang maps wttr.in lang codes to OWM equivalents where they differ.
func owmNormalizeLang(lang string) string {
	if lang == "" {
		return "en"
	}
	overrides := map[string]string{
		"zh": "zh_cn", "zh-tw": "zh_tw",
	}
	if v, ok := overrides[lang]; ok {
		return v
	}
	return lang
}
