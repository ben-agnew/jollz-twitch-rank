package main

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	valApi "github.com/ben-agnew/jollz-twitch-rank/libs/twitch"
)

func GetRankString(name string, tag string, rank string, elo int, change int, rr int, winRate string, kills string, account Account) (string, error) {

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
			case "kills":
				stats = append(stats, "Kills: "+kills)
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

	// get total kills

	kills := requestKills(name, tag)

	var rankData = &valApi.RankData{

		Name:    rankRes.Data.Name,
		Tag:     rankRes.Data.Tag,
		Rank:    rankRes.Data.CurrentData.Rank,
		Elo:     rankRes.Data.CurrentData.Elo,
		RR:      rankRes.Data.CurrentData.RR,
		Change:  rankRes.Data.CurrentData.Change,
		WinRate: winRate,
		Kills:   kills,
	}

	return rankData, nil
}

func requestKills(name string, tag string) string {

	reqUrl := configuration.TrackerURL + name + "%23" + tag + "/weapons"

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		log.WithField("event", "new_request_trackernet").Error(err)
		return "Unknown"
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")

	res, err := httpClient.Do(req)
	if err != nil {
		log.WithField("event", "do_request_trackernet").Error(err)
		return "Unknown"
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return "Unknown"
	}

	b, err := io.ReadAll(res.Body)
	// b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
	if err != nil {
		log.Fatalln(err)
	}

	if b == nil {
		log.WithField("event", "request_kills").Error("body is nil")
		return "Unknown"
	}

	// return "Unknown", nil

	var stringMatch = `window.__INITIAL_STATE__ = `

	index := strings.Index(string(b), stringMatch)

	if index == -1 {
		log.WithField("event", "request_kills").Error("index is -1")

	}

	// find the index of the </script> tag

	// find all the indexes of the </script> tag

	endIndex := regexp.MustCompile(`<\/script>`).FindAllStringSubmatchIndex(string(b), -1)

	if endIndex == nil {
		log.WithField("event", "request_kills").Error("endIndex is nil")
		return "Unknown"
	}

	// get second last index

	secondLastIndex := endIndex[len(endIndex)-2]

	if secondLastIndex == nil {
		log.WithField("event", "request_kills").Error("secondLastIndex is nil")
		return "Unknown"
	}

	// select the data between the two indexes

	data := string(b)[index+len(stringMatch) : secondLastIndex[0]]

	if data == "" {
		log.WithField("event", "request_kills").Error("data is empty")
		return "Unknown"
	}

	// return "Unknown", nil

	// // unmarshal the json data

	var rankRes valApi.GetTrackerNetResponse
	err = json.NewDecoder(strings.NewReader(data)).Decode(&rankRes)
	if err != nil {
		log.WithField("event", "bad decode").Error(err)
		return "Unknown"
	}

	// find the total kills

	var totalKills string

	if rankRes.Stats.Profiles[0].Segments[0].Stats.Kills.DisplayValue != "" {
		totalKills = rankRes.Stats.Profiles[0].Segments[0].Stats.Kills.DisplayValue
	} else {
		totalKills = "Unknown"
	}

	return totalKills
}
