package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/ben-agnew/jollz-twitch-rank/libs/cache"
	"github.com/gempir/go-twitch-irc/v3"
	log "github.com/sirupsen/logrus"
	"github.com/tkanos/gonfig"

	"strings"
)

var configuration Configuration
var redisCache cache.Cache

func main() {

	log.Info("Starting twitchbot...")
	err := gonfig.GetConf("config.json", &configuration)
	if err != nil {
		log.WithField("event", "load_config").Fatal(err)
		return
	}

	client := twitch.NewClient(configuration.TwitchUsername, configuration.TwitchToken)
	client.SetJoinRateLimiter(twitch.CreateVerifiedRateLimiter())

	redisCache = cache.NewCache(configuration.CacheUrl)

	err = redisCache.SetKeepTtl("jollz: current", configuration.Accounts[0].Name+"/"+configuration.Accounts[0].Tag)
	if err != nil {
		log.WithField("event", "user_command_cache_set").Error(err)
	}

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		go handleMessage(message, client)
	})
	client.Join(configuration.TwitchChannel)

	client.OnConnect(func() {
		log.WithField("event", "irc_connected").Info("IRC connected")
	})

	err = client.Connect()
	if err != nil {
		log.WithField("event", "irc_connect").Fatal(err)
		return
	}
}

func handleMessage(message twitch.PrivateMessage, client *twitch.Client) {

	if message.Channel == strings.ToLower(configuration.TwitchChannel) {
		switch strings.Split(strings.ToLower(message.Message), " ")[0] {
		case "!rank":
			rankCommand(&message, client)

		case "!accounts":
			accountsCommand(&message, client)

		case "!setcurrent":
			setCurrentCommand(&message, client)

		case "!current":
			currentCommand(&message, client)
		case "!bot":
			botCommand(&message, client)
		default:
			checkCommands(&message, client)
		}

	}
}

type CachedChannel struct {
	Name         string `json:"name"`
	Tag          string `json:"tag"`
	Rank         string `json:"rank"`
	Elo          int    `json:"elo"`
	Change       int    `json:"change"`
	RR           int    `json:"rr"`
	WinRate      string `json:"winrate"`
	AccountIndex int    `json:"account_index"`
}

func rankCommand(message *twitch.PrivateMessage, client *twitch.Client) {
	currentAccount, err := redisCache.Get("jollz: current")
	if err != nil {
		currentAccount = configuration.Accounts[0].Name + "/" + configuration.Accounts[0].Tag
		err = redisCache.SetKeepTtl("jollz: current", currentAccount)
		if err != nil {
			log.WithField("event", "user_command_cache_set").Error(err)
			return
		}

	}
	cachedStr, err := redisCache.Get("jollz:" + currentAccount)
	if err == nil {
		cachedObj := CachedChannel{}
		err = json.Unmarshal([]byte(cachedStr), &cachedObj)
		if err != nil {
			return
		}

		replyStr, err := GetRankString(cachedObj.Name, cachedObj.Tag, cachedObj.Rank, cachedObj.Elo, cachedObj.Change, cachedObj.RR, cachedObj.WinRate, configuration.Accounts[cachedObj.AccountIndex])
		if err != nil {
			client.Say(message.Channel, "Error getting rank")
			return
		}
		client.Say(message.Channel, replyStr)
		return
	} else {
		currentAccountIndex := getCurrentIndex(currentAccount)

		data, err := requestRank(configuration.Accounts[currentAccountIndex].Name, configuration.Accounts[currentAccountIndex].Tag)
		if err != nil {
			client.Say(message.Channel, "Error getting rank")
			return
		}

		// cache for 1 minute
		cachedObj := CachedChannel{
			Name:         data.Name,
			Tag:          data.Tag,
			Rank:         data.Rank,
			Elo:          data.Elo,
			Change:       data.Change,
			RR:           data.RR,
			WinRate:      data.WinRate,
			AccountIndex: currentAccountIndex,
		}
		cachedStr, err := json.Marshal(cachedObj)
		if err != nil {
			log.WithField("event", "user_command_cache_set").Error(err)
		}
		err = redisCache.SetWithTtl("jollz:"+currentAccount, string(cachedStr), time.Minute)
		if err != nil {
			log.WithField("event", "user_command_cache_set").Error(err)
		}

		replyStr, err := GetRankString(data.Name, data.Tag, data.Rank, data.Elo, data.Change, data.RR, data.WinRate, configuration.Accounts[currentAccountIndex])
		if err != nil {
			client.Say(message.Channel, "Error getting rank")
			return
		}
		client.Say(message.Channel, replyStr)
		return

	}

}

func accountsCommand(message *twitch.PrivateMessage, client *twitch.Client) {

	// check if user is mod or broadcaster

	if !modCaster(message) {
		// log.WithField("event", "accounts_command").Info(message.User.Name + " tried to use accounts command but is not mod or broadcaster")
		return
	}

	// get all accounts

	var accounts = configuration.Accounts

	var accountNames = []string{}

	for i, account := range accounts {

		accountNames = append(accountNames, strconv.Itoa(i+1)+". "+account.Name+" ("+account.Id+")")

	}
	log.WithField("event", "accounts_command").Info(message.User.Name + " used accounts command")

	client.Say(message.Channel, "Accounts: "+strings.Join(accountNames, ", "))
}

func setCurrentCommand(message *twitch.PrivateMessage, client *twitch.Client) {
	// check if user is mod or broadcaster

	if !modCaster(message) {
		// log.WithField("event", "setcurrent_command").Info(message.User.Name + " tried to use setcurrent command but is not mod or broadcaster")

		return
	}

	// get selected account

	msg := strings.TrimPrefix(strings.ToLower(message.Message), "!setcurrent ")
	// check if can be converted to int
	index, err := strconv.Atoi(msg)
	if err != nil {
		// get account by name
		for i, account := range configuration.Accounts {
			// convert to lowercase
			if strings.ToLower(account.Name) == strings.ToLower(msg) {
				index = i
				break
			}
			// check if name was not found

			if i == len(configuration.Accounts)-1 {
				client.Say(message.Channel, "@"+message.User.DisplayName+": Account not found")
				return
			}

		}

		// get account by index

		account := configuration.Accounts[index]

		// set current account

		err = redisCache.SetKeepTtl("jollz: current", account.Name+"/"+account.Tag)
		if err != nil {

			log.WithField("event", "user_command_cache_set").Error(err)
			client.Say(message.Channel, "@"+message.User.DisplayName+": Error setting account")
			return
		}
		log.WithField("event", "set_current_command").Info(message.User.Name + " set current account to " + account.Name)
		client.Say(message.Channel, "@"+message.User.DisplayName+": Current account set to "+account.Name+" ("+account.Id+")")

	} else {
		// check if index is valid

		if index > len(configuration.Accounts) || index < 1 {
			client.Say(message.Channel, "@"+message.User.DisplayName+": Invalid account number")
			return
		}

		// get account by index

		account := configuration.Accounts[index-1]

		// set current account

		err = redisCache.SetKeepTtl("jollz: current", account.Name+"/"+account.Tag)
		if err != nil {
			log.WithField("event", "user_command_cache_set").Error(err)
			client.Say(message.Channel, "@"+message.User.DisplayName+": Error setting account")
			return
		}
		log.WithField("event", "set_current_command").Info(message.User.Name + " set current account to " + account.Name)
		client.Say(message.Channel, "@"+message.User.DisplayName+": Current account set to "+account.Name+" ("+account.Id+")")

	}

}

func currentCommand(message *twitch.PrivateMessage, client *twitch.Client) {
	// get current account

	currentAccount, err := redisCache.Get("jollz: current")
	if err != nil {
		client.Say(message.Channel, "@"+message.User.DisplayName+": Error getting current account")
		return
	}

	account := configuration.Accounts[getCurrentIndex(currentAccount)]
	client.Say(message.Channel, "@"+message.User.DisplayName+": Current account is "+account.Name+" ("+account.Id+")")
}

func getCurrentIndex(currentAccount string) int {
	for i, account := range configuration.Accounts {
		if account.Name+"/"+account.Tag == currentAccount {
			return i
		}
	}
	return 0
}

func checkCommands(message *twitch.PrivateMessage, client *twitch.Client) {

	// check if message is command

	if strings.HasPrefix(message.Message, "!") {

		// check if command is in list accounts

		for _, account := range configuration.Accounts {

			if strings.ToLower(strings.Split(message.Message, " ")[0]) == strings.ToLower(account.Command) {
				// get rank for account

				// check if account is cached

				cachedAccount, err := redisCache.Get("jollz:" + account.Name + "/" + account.Tag)
				if err != nil {
					data, err := requestRank(account.Name, account.Tag)
					if err != nil {
						client.Say(message.Channel, "Error getting rank")
						return
					}

					// cache for 1 minute
					cachedObj := CachedChannel{
						Name:         data.Name,
						Tag:          data.Tag,
						Rank:         data.Rank,
						Elo:          data.Elo,
						Change:       data.Change,
						RR:           data.RR,
						WinRate:      data.WinRate,
						AccountIndex: getCurrentIndex(account.Name + "/" + account.Tag),
					}
					cachedStr, err := json.Marshal(cachedObj)
					if err != nil {
						log.WithField("event", "user_command_cache_set").Error(err)
					}
					err = redisCache.SetWithTtl("jollz:"+account.Name+"/"+account.Tag, string(cachedStr), time.Minute)
					if err != nil {
						log.WithField("event", "user_command_cache_set").Error(err)
					}

					replyStr, err := GetRankString(data.Name, data.Tag, data.Rank, data.Elo, data.Change, data.RR, data.WinRate, account)
					if err != nil {
						client.Say(message.Channel, "Error getting rank")
						return
					}
					client.Say(message.Channel, replyStr)
					return
				} else {
					var cachedObj CachedChannel
					err = json.Unmarshal([]byte(cachedAccount), &cachedObj)
					if err != nil {
						log.WithField("event", "user_command_cache_set").Error(err)
					}
					replyStr, err := GetRankString(cachedObj.Name, cachedObj.Tag, cachedObj.Rank, cachedObj.Elo, cachedObj.Change, cachedObj.RR, cachedObj.WinRate, account)
					if err != nil {
						client.Say(message.Channel, "Error getting rank")
						return
					}
					client.Say(message.Channel, replyStr)
					return
				}

			}
		}

	}
}

func botCommand(message *twitch.PrivateMessage, client *twitch.Client) {
	client.Say(message.Channel, "@"+message.User.DisplayName+": Hello")
}

// check if user is mod or broadcaster

func modCaster(message *twitch.PrivateMessage) bool {

	isMod, ok := message.Tags["mod"]
	if ok && isMod != "1" {
		if strings.ToLower(message.Tags["display-name"]) != strings.ToLower(message.Channel) {
			return false
		}
	}
	return true

}
