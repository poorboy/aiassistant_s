package config

type Config struct {
	Port          string
	DBPath        string
	DeepSeekKey   string
	DeepSeekURL   string
	DeepSeekModel string
}

func Load() *Config {
	return &Config{
		Port:          "41400",
		DBPath:        "./data/assdata.db",
		DeepSeekKey:   "",
		DeepSeekURL:   "https://api.deepseek.com",
		DeepSeekModel: "deepseek-v4-flash",
	}
}
