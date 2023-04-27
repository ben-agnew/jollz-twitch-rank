package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	valApi "github.com/ben-agnew/jollz-twitch-rank/libs/twitch"
)

func GetRankString(name string, tag string, rank string, elo int, change int, rr int, winRate string, account Account) (string, error) {

	// check what stats are requested

	var stats []string
	if len(account.Stats) == 0 {

		stats = []string{"Current Rank: " + rank, "Elo: " + strconv.Itoa(elo), "RR: " + strconv.Itoa(rr)}

	} else {

		for _, stat := range account.Stats {
			switch stat {
			case "rank":
				stats = append(stats, "Current Rank: "+rank)
			case "elo":
				stats = append(stats, "Elo: "+strconv.Itoa(elo))
			case "rr":
				stats = append(stats, "RR: "+strconv.Itoa(rr))
			case "winrate":
				stats = append(stats, "Winrate: "+winRate)
			case "change":
				stats = append(stats, "Change: "+strconv.Itoa(change))
			}

		}
	}

	return account.Id + ": " + strings.Join(stats, " | "), nil

}

type PlayerNotFoundError struct{}

func (p PlayerNotFoundError) Error() string {
	return "Player not found"
}

var httpClient = http.Client{
	Timeout: time.Second * 10,
}

func requestRank(name string, tag string) (*valApi.RankData, error) {

	reqUrl := configuration.ValApiURL + name + "/" + tag

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		log.WithField("event", "new_request_trackernet").Error(err)
		return nil, err
	}
	req.Header.Set("User-Agent", "yannismate-api/services/twitchbot")

	res, err := httpClient.Do(req)
	if err != nil {
		log.WithField("event", "do_request_trackernet").Error(err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, &PlayerNotFoundError{}
	}

	var rankRes valApi.GetRankResponse
	err = json.NewDecoder(res.Body).Decode(&rankRes)
	if err != nil {
		log.WithField("event", "read_body_trackernet").Error(err)
		return nil, err
	}

	// find total games played

	var totalGames int
	var totalWins int
	values := reflect.ValueOf(rankRes.Data.BySeason)
	for i := 0; i < values.NumField(); i++ {
		field := values.Field(i)
		if field.Kind() == reflect.Struct {
			season := field.Interface().(valApi.Season)
			if season.Error == "" {
				totalGames += season.Games
				totalWins += season.Wins
			}
		}
	}

	// calculate winrate

	var winRate string
	if totalGames == 0 {
		winRate = "0%"
	} else {
		// rount to 2 decimal places
		winRate = strconv.FormatFloat(float64(totalWins)/float64(totalGames)*100, 'f', 2, 64) + "%"
	}

	// add winrate to response

	var rankData = &valApi.RankData{

		Name:    rankRes.Data.Name,
		Tag:     rankRes.Data.Tag,
		Rank:    rankRes.Data.CurrentData.Rank,
		Elo:     rankRes.Data.CurrentData.Elo,
		RR:      rankRes.Data.CurrentData.RR,
		Change:  rankRes.Data.CurrentData.Change,
		WinRate: winRate,
	}

	return rankData, nil
}
