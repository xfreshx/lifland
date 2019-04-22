package main

import (
	"github.com/xfreshx/lifland/storage"
	"github.com/xfreshx/lifland/types"
	"github.com/xfreshx/lifland/utils"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`Social tournament service. Please register players and start the tournament.`))
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

	p, err := storage.GetConn().GetPlayer(playerId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	err = takePlayer(p, points)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
	}

	if err = storage.GetConn().SetPlayer(p); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
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

	p, err := storage.GetConn().GetPlayer(playerId)
	if err != nil {
		if err == storage.ErrNoPlayer {
			p = &types.Player{
				Id: playerId,
				Backers: make(map[string]bool),
			}
			log.Println("no such player, will be created")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
	}

	p.Points += points

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
		Id: tournamentId,
		Deposit: deposit,
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

	t, err := storage.GetConn().GetTournament(tournamentId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	p, err := storage.GetConn().GetPlayer(playerId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	if p.Points < t.Deposit {
		backers, ok := params["backerId"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("player has insufficient score and no backers provided")
			return
		}

		// share deposit among all backers + a player himself
		pointsPerBacker := t.Deposit / uint64(len(backers) + 1)

		for _, backerId := range backers {

			b, err := storage.GetConn().GetPlayer(backerId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}

			err = takePlayer(b, pointsPerBacker)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}

			p.Points += pointsPerBacker
			p.Backers[backerId] = true

			err = storage.GetConn().SetMultiPlayer(b, p)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}
		}
	}

	err = takePlayer(p, t.Deposit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	err = storage.GetConn().SetPlayer(p)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	t.Players = append(t.Players, p.Id)
	if err = storage.GetConn().SetTournament(t); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
}

//todo: Why without tournamentId? Why not delete tournament after resulting?
func ResultTournamentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	tournamentResult := struct {
		Winners []struct{
			PlayerId string `json:"playerId"`
			Prize     uint64 `json:"prize"`
		} `json:"winners"`
	}{}

	err := decoder.Decode(&tournamentResult)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	for _, winner := range tournamentResult.Winners {
		p, err := storage.GetConn().GetPlayer(winner.PlayerId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}

		p.Points += winner.Prize
		err = storage.GetConn().SetPlayer(p)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}

		if len(p.Backers) > 0 {
			// pay back in the same ratio
			pointsPerBacker := winner.Prize / uint64(len(p.Backers) + 1)

			for backerId, _  := range p.Backers {

				err = takePlayer(p, pointsPerBacker)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Println(err.Error())
					return
				}

				b, err := storage.GetConn().GetPlayer(backerId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Println(err.Error())
					return
				}

				b.Points += pointsPerBacker
				delete(p.Backers, backerId)

				err = storage.GetConn().SetMultiPlayer(b, p)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Println(err.Error())
					return
				}
			}
		}
	}
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