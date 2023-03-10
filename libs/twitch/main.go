package valApi

type RankData struct {
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Rank    string `json:"rank"`
	Elo     int    `json:"elo"`
	Change  int    `json:"change"`
	RR      int    `json:"rr"`
	WinRate string `json:"winrate"`
}

type GetRankResponse struct {
	Data Data `json:"data"`
}

type Data struct {
	Name        string      `json:"name"`
	Tag         string      `json:"tag"`
	CurrentData CurrentData `json:"current_data"`
	BySeason    BySeason    `json:"by_season"`
}

type CurrentData struct {
	Rank   string `json:"currenttierpatched"`
	Elo    int    `json:"elo"`
	RR     int    `json:"ranking_in_tier"`
	Change int    `json:"mmr_change_to_last_game"`
}

type BySeason struct {
	E1a1 Season `json:"e1a1"`
	E1a2 Season `json:"e1a2"`
	E1a3 Season `json:"e1a3"`
	E2a1 Season `json:"e2a1"`
	E2a2 Season `json:"e2a2"`
	E2a3 Season `json:"e2a3"`
	E3a1 Season `json:"e3a1"`
	E3a2 Season `json:"e3a2"`
	E3a3 Season `json:"e3a3"`
	E4a1 Season `json:"e4a1"`
	E4a2 Season `json:"e4a2"`
	E4a3 Season `json:"e4a3"`
	E5a1 Season `json:"e5a1"`
	E5a2 Season `json:"e5a2"`
	E5a3 Season `json:"e5a3"`
	E6a1 Season `json:"e6a1"`
	E6a2 Season `json:"e6a2"`
}

type Season struct {
	Wins  int    `json:"wins"`
	Games int    `json:"number_of_games"`
	Error string `json:"error"`
}
