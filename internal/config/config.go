package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIID               int
	APIHash             string
	BotToken            string
	SessionName         string
	AuthUsers           []int64
	LogChannel          int64
	StreamHost          string
	IAMHeader           string
	MaxWorkers          int
	MaxProcessesPerUser int
	SlowSpeedDelay      time.Duration
	ProcessTimeout      time.Duration
	DBPath              string
	Debug               bool
	MaxTrimDuration     int
}

var (
	Colors = []string{
		"white", "black", "red", "blue", "green", "yellow",
		"orange", "purple", "brown", "gold", "silver", "pink",
	}

	FontSizesName = []string{"Small", "Medium", "Large"}
	FontSizes     = []int{30, 40, 50}

	Positions = []string{
		"Top Left", "Top Center", "Top Right",
		"Center Left", "Centered", "Center Right",
		"Bottom Left", "Bottom Center", "Bottom Right",
	}
)

func Load() *Config {
	return &Config{
		APIID:               parseInt(getEnv("API_ID", "0")),
		APIHash:             getEnv("API_HASH", ""),
		BotToken:            getEnv("BOT_TOKEN", ""),
		SessionName:         getEnv("SESSION_NAME", "screenshotbot"),
		AuthUsers:           parseInt64Slice(getEnv("AUTH_USERS", "")),
		LogChannel:          parseInt64(getEnv("LOG_CHANNEL", "0")),
		StreamHost:          getEnv("HOST", ""),
		IAMHeader:           getEnv("IAM_HEADER", ""),
		MaxWorkers:          parseInt(getEnv("MAX_WORKERS", "2")),
		MaxProcessesPerUser: parseInt(getEnv("MAX_PROCESSES_PER_USER", "1")),
		SlowSpeedDelay:      time.Duration(parseInt(getEnv("SLOW_SPEED_DELAY", "5"))) * time.Second,
		ProcessTimeout:      time.Duration(parseInt(getEnv("TIMEOUT", "300"))) * time.Second,
		DBPath:              getEnv("DB_PATH", "data/bot.db"),
		Debug:               parseBool(getEnv("DEBUG", "false")),
		MaxTrimDuration:     parseInt(getEnv("MAX_TRIM_DURATION", "600")),
	}
}

func (c *Config) IsAdmin(userID int64) bool {
	for _, adminID := range c.AuthUsers {
		if adminID == userID {
			return true
		}
	}
	return false
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseBool(s string) bool {
	v, _ := strconv.ParseBool(s)
	return v
}

func parseInt64Slice(s string) []int64 {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Fields(s)
	res := make([]int64, 0, len(parts))
	for _, p := range parts {
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			res = append(res, id)
		}
	}
	return res
}
