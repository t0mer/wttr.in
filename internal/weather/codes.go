package weather

import (
	"fmt"
	"strings"
)

// wmoToWWO maps WMO weather interpretation codes (used by Open-Meteo) to WWO codes.
var wmoToWWO = map[int]int{
	0: 113, 1: 116, 2: 116, 3: 122,
	45: 143, 48: 143,
	51: 266, 53: 266, 55: 266,
	56: 281, 57: 284,
	61: 293, 63: 302, 65: 308,
	66: 311, 67: 314,
	71: 320, 73: 329, 75: 338, 77: 320,
	80: 353, 81: 356, 82: 359,
	85: 368, 86: 371,
	95: 389, 96: 386, 99: 395,
}

// owmToWWO maps OpenWeatherMap condition codes to WWO codes.
var owmToWWO = map[int]int{
	200: 389, 201: 389, 202: 389, 210: 386, 211: 386, 212: 389, 221: 386, 230: 386, 231: 386, 232: 386,
	300: 263, 301: 266, 302: 266, 310: 263, 311: 266, 312: 266, 313: 353, 314: 356, 321: 353,
	500: 293, 501: 302, 502: 308, 503: 308, 504: 308, 511: 311, 520: 353, 521: 356, 522: 359, 531: 356,
	600: 320, 601: 329, 602: 338, 611: 317, 612: 362, 613: 365, 615: 317, 616: 317, 620: 368, 621: 371, 622: 371,
	701: 143, 711: 143, 721: 143, 731: 143, 741: 143, 751: 143, 761: 143, 762: 143, 771: 200, 781: 200,
	800: 113, 801: 116, 802: 116, 803: 119, 804: 122,
}

// weatherAPIToWWO maps WeatherAPI.com condition codes to WWO codes.
var weatherAPIToWWO = map[int]int{
	1000: 113, 1003: 116, 1006: 119, 1009: 122,
	1030: 143, 1063: 176, 1066: 179, 1069: 182, 1072: 185,
	1087: 200, 1114: 227, 1117: 230, 1135: 248, 1147: 260,
	1150: 263, 1153: 266, 1168: 281, 1171: 284,
	1180: 293, 1183: 296, 1186: 299, 1189: 302, 1192: 305, 1195: 308,
	1198: 311, 1201: 314, 1204: 317, 1207: 320,
	1210: 323, 1213: 326, 1216: 329, 1219: 332, 1222: 335, 1225: 338,
	1237: 350, 1240: 353, 1243: 356, 1246: 359,
	1249: 362, 1252: 365, 1255: 368, 1258: 371,
	1261: 374, 1264: 377,
	1273: 386, 1276: 389, 1279: 392, 1282: 395,
}

// mapCode translates a provider-native code to a WWO code; returns 116 (partly cloudy) on miss.
func mapCode(table map[int]int, code int) int {
	if wwo, ok := table[code]; ok {
		return wwo
	}
	return 116
}

var windDirs = []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}

func degreesToWindDir16(degrees float64) string {
	idx := int((degrees+11.25)/22.5) % 16
	return windDirs[idx]
}

func celsiusToFahrenheit(c float64) float64 { return c*9/5 + 32 }
func kmphToMph(k float64) float64           { return k * 0.621371 }
func mpsToKmph(m float64) float64           { return m * 3.6 }
func mbToInHg(mb float64) float64           { return mb * 0.02953 }
func kmToMiles(km float64) float64          { return km * 0.621371 }

func itoa(i int) string     { return fmt.Sprintf("%d", i) }
func ftoa(f float64) string { return fmt.Sprintf("%.1f", f) }
func ftoa0(f float64) string { return fmt.Sprintf("%.0f", f) }

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func safeF64(s []float64, i int) float64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func safeInt(s []int, i int) int {
	if i < len(s) {
		return s[i]
	}
	return 0
}
