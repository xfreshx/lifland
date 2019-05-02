package types

import (
	"encoding/json"
	"log"
)

type Player struct {
	Id     string `json:"id"`
	Points uint64 `json:"balance"`
	Backers map[string]interface{} `json:"-"`
}

func (p *Player) GetBackersJson() string {
	b, err := json.Marshal(p.Backers)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(b)
}

func (p *Player) SetBackers(j string)  {
	err := json.Unmarshal([]byte(j), &p.Backers)
	if err != nil {
		log.Println(err)
	}
}

type Tournament struct {
	Id      string
	Players map[string]interface{}
	Deposit uint64
}

func (t *Tournament) GetPlayersJson() string {
	b, err := json.Marshal(t.Players)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(b)
}

func (t *Tournament) SetPlayers(j string)  {
	err := json.Unmarshal([]byte(j), &t.Players)
	if err != nil {
		log.Println(err)
	}
}

