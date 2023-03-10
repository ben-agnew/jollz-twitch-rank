package main

type Account struct {
	Name    string   `json:"name"`
	Tag     string   `json:"tag"`
	Stats   []string `json:"stats"`
	Id      string   `json:"id"`
	Command string   `json:"command"`
}

type Configuration struct {
	TwitchUsername string    `env:"TWITCH_USER"`
	TwitchToken    string    `env:"TWITCH_TOKEN"`
	TwitchChannel  string    `env:"TWITCH_CHANNEL"`
	CacheUrl       string    `json:"cacheUrl"`
	Accounts       []Account `json:"accounts"`
	ValApiURL      string    `json:"valApiUrl"`
}
