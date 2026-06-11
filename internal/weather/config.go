package weather

// Config holds weather provider selection and per-provider settings.
type Config struct {
	// Provider selects the weather backend: "wwo", "open_meteo", "owm", "weatherapi".
	// Defaults to "wwo" when a wwo block is present, otherwise "open_meteo".
	Provider string `yaml:"provider"`

	WWO        *WWOConfig        `yaml:"wwo"`
	OpenMeteo  *OpenMeteoConfig  `yaml:"open_meteo"`
	OWM        *OWMConfig        `yaml:"owm"`
	WeatherAPI *WeatherAPIConfig `yaml:"weatherapi"`
}
