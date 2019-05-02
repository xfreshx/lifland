package main

import (
	"encoding/json"
	"errors"
	"github.com/xfreshx/lifland/storage"
	"github.com/xfreshx/lifland/types"
	"github.com/xfreshx/lifland/utils"
	"log"
	"net/http"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`Social tournament service. Please register players and start the tournament.`))
}

func TakeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()

	playerId, err := utils.GetStringURLParam(params, "playerId")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid playerId given")
		return
	}

	points, err := utils.GetUintURLParam(params, "points")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid points given")
		return
	}

	commit := false
	err = storage.GetConn().BeginTransaction()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	defer storage.GetConn().FinalizeTransaction(&commit)

	p, err := storage.GetConn().GetPlayersForUpdate([]string{playerId})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	err = takePlayer(p[0], points)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
	}

	if err = storage.GetConn().UpdatePlayer(p[0]); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	commit = true
}

func FundHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()

	playerId, err := utils.GetStringURLParam(params, "playerId")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid playerId given")
		return
	}

	points, err := utils.GetUintURLParam(params, "points")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid points given")
		return
	}

	p := &types.Player{
		Id:      playerId,
		Points:  points,
		Backers: make(map[string]interface{}),
	}

	if err = storage.GetConn().SetPlayer(p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
}

func AnnounceTournamentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()

	tournamentId, err := utils.GetStringURLParam(params, "tournamentId")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid tournamentId given")
		return
	}

	deposit, err := utils.GetUintURLParam(params, "deposit")
	if err != nil || deposit == 0 {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid deposit given")
		return
	}

	t := &types.Tournament{
		Id:      tournamentId,
		Deposit: deposit,
		Players: make(map[string]interface{}),
	}

	if err = storage.GetConn().SetTournament(t); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
	}
}

func JoinTournamentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()

	tournamentId, err := utils.GetStringURLParam(params, "tournamentId")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid tournamentId given")
		return
	}

	playerId, err := utils.GetStringURLParam(params, "playerId")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid playerId given")
		return
	}

	commit := false
	err = storage.GetConn().BeginTransaction()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	defer storage.GetConn().FinalizeTransaction(&commit)

	t, err := storage.GetConn().GetTournamentForUpdate(tournamentId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	p, err := storage.GetConn().GetPlayersForUpdate([]string{playerId})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	if p[0].Points < t.Deposit {
		backers, ok := params["backerId"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("player has insufficient score and no backers provided")
			return
		}

		// share deposit among all backers + a player himself
		pointsPerBacker := t.Deposit / uint64(len(backers)+1)

		backerPlayers, err := storage.GetConn().GetPlayersForUpdate(backers)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}

		for _, b := range backerPlayers {
			err = takePlayer(b, pointsPerBacker)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}

			p[0].Points += pointsPerBacker
			p[0].Backers[b.Id] = true

			err = storage.GetConn().UpdatePlayer(p[0])
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}

			err = storage.GetConn().UpdatePlayer(b)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}
		}
	}

	err = takePlayer(p[0], t.Deposit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	err = storage.GetConn().UpdatePlayer(p[0])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	t.Players[p[0].Id] = true
	if err = storage.GetConn().UpdateTournament(t); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	commit = true
}

func ResultTournamentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	tournamentResult := struct {
		TournamentId string `json:"tournamentId"`
		Winners      []struct {
			PlayerId string `json:"playerId"`
			Prize    uint64 `json:"prize"`
		} `json:"winners"`
	}{}

	err := decoder.Decode(&tournamentResult)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	if tournamentResult.TournamentId == "" {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("no tournament id provided")
		return
	}

	commit := false
	err = storage.GetConn().BeginTransaction()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	defer storage.GetConn().FinalizeTransaction(&commit)

	t, err := storage.GetConn().GetTournamentForUpdate(tournamentResult.TournamentId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	for _, winner := range tournamentResult.Winners {

		if _, found := t.Players[winner.PlayerId]; !found {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("no such player registered in the tournament")
			return
		}

		p, err := storage.GetConn().GetPlayersForUpdate([]string{winner.PlayerId})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}

		p[0].Points += winner.Prize
		err = storage.GetConn().UpdatePlayer(p[0])
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}

		if len(p[0].Backers) > 0 {
			// pay back in the same ratio
			pointsPerBacker := winner.Prize / uint64(len(p[0].Backers)+1)

			var backersId []string
			for backerId, _ := range p[0].Backers {
				backersId = append(backersId, backerId)
			}

			backerPlayers, err := storage.GetConn().GetPlayersForUpdate(backersId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}

			for _, b := range backerPlayers {

				err = takePlayer(p[0], pointsPerBacker)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Println(err.Error())
					return
				}

				b.Points += pointsPerBacker
				delete(p[0].Backers, b.Id)

				err = storage.GetConn().UpdatePlayer(b)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Println(err.Error())
					return
				}
			}
		}

		err = storage.GetConn().UpdatePlayer(p[0])
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
	}

	err = storage.GetConn().DeleteTournament(tournamentResult.TournamentId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	commit = true
}

func BalanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()

	playerId, err := utils.GetStringURLParam(params, "playerId")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("invalid playerId given")
		return
	}

	p, err := storage.GetConn().GetPlayer(playerId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	j, err := json.Marshal(p)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	_, err = w.Write(j)
	if err != nil {
		log.Println(err.Error())
	}
}

func ResetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := storage.GetConn().Reset()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
	}
}

func takePlayer(p *types.Player, points uint64) error {

	if p.Points >= points {
		p.Points -= points
	} else {
		return errors.New("not enough points to take")
	}

	return nil
}
