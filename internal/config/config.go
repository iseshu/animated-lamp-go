package config

import (
	"flag"
	"log"
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

	flagToken     = flag.String("token", "", "Telegram Bot Token")
	flagAPIID     = flag.Int("api-id", 0, "Telegram API ID")
	flagAPIHash   = flag.String("api-hash", "", "Telegram API Hash")
	flagAuthUsers = flag.String("auth-users", "", "Admin IDs separated by space")
)

func loadEnvFiles(paths ...string) {
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, `"'`)
				if os.Getenv(key) == "" {
					_ = os.Setenv(key, val)
				}
			}
		}
	}
}

func Load() *Config {
	if !flag.Parsed() {
		flag.Parse()
	}

	loadEnvFiles(".env", "environment", "../.env")

	botToken := *flagToken
	if botToken == "" {
		botToken = getEnv("BOT_TOKEN", "")
	}
	if botToken == "" && flag.NArg() > 0 {
		botToken = flag.Arg(0)
	}

	apiID := *flagAPIID
	if apiID == 0 {
		apiID = parseInt(getEnv("API_ID", "0"))
	}

	apiHash := *flagAPIHash
	if apiHash == "" {
		apiHash = getEnv("API_HASH", "")
	}

	authUsersStr := *flagAuthUsers
	if authUsersStr == "" {
		authUsersStr = getEnv("AUTH_USERS", "")
	}

	if botToken == "" {
		var envKeys []string
		for _, e := range os.Environ() {
			pair := strings.SplitN(e, "=", 2)
			envKeys = append(envKeys, pair[0])
		}
		log.Printf("🔍 Detected environment variables in container: %v", envKeys)
		log.Println("💡 Tip: You can pass token directly in startup command: ./bot -token=\"YOUR_BOT_TOKEN\" -api-id=YOUR_ID -api-hash=\"YOUR_HASH\"")
	}

	return &Config{
		APIID:               apiID,
		APIHash:             apiHash,
		BotToken:            botToken,
		SessionName:         getEnv("SESSION_NAME", "screenshotbot"),
		AuthUsers:           parseInt64Slice(authUsersStr),
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
