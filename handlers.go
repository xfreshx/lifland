package main

import (
	"./storage"
	"./types"
	"encoding/json"
	"net/http"
	"strconv"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`Social tournament service. Please register players and start the tournament.`))
}

func TakeHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	params := r.URL.Query()
	playerId := params.Get("playerId")
	points := params.Get("points")

	if playerId == "" || points == "" {
		errorHandler(w, http.StatusBadRequest, "invalid params given: " + playerId + " " + points)
		return
	}

	uPoints, err := strconv.ParseUint(points, 10, 64)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid points given: " + points)
		return
	}

	p, err := storage.GetConn().GetPlayer(playerId)
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
	}

	if p.Points >= uPoints {
		p.Points -= uPoints
	} else {
		errorHandler(w, http.StatusInternalServerError, "not enough points")
		return
	}

	if err = storage.GetConn().SetPlayer(p); err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
	}
}

func FundHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	params := r.URL.Query()
	playerId := params.Get("playerId")
	points := params.Get("points")

	if playerId == "" || points == "" {
		errorHandler(w, http.StatusBadRequest, "invalid params given: " + playerId + " " + points)
		return
	}

	uPoints, err := strconv.ParseUint(points, 10, 64)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid points given: " + points)
		return
	}

	p, err := storage.GetConn().GetPlayer(playerId)
	if err != nil {
		if err == storage.ErrNoPlayer {
			p = &types.Player{
				Id: playerId,
			}
		} else {
			errorHandler(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	p.Points += uPoints

	if err = storage.GetConn().SetPlayer(p); err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
	}
}

func AnnounceTournamentHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	params := r.URL.Query()
	tournamentId := params.Get("tournamentId")
	deposit := params.Get("deposit")

	if tournamentId == "" || deposit == "" {
		errorHandler(w, http.StatusBadRequest, "invalid params given: " + tournamentId + " " + deposit)
		return
	}

	uDeposit, err := strconv.ParseUint(deposit, 10, 64)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid deposit given: " + deposit)
		return
	}

	if uDeposit == 0 {
		errorHandler(w, http.StatusBadRequest, "invalid deposit given: " + deposit)
		return
	}

	t := &types.Tournament{
		Id: tournamentId,
		Deposit: uDeposit,
		Players: make(map[string]*types.Player),
	}

	if err = storage.GetConn().SetTournament(t); err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
	}
}

func JoinTournamentHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	params := r.URL.Query()
	tournamentId := params.Get("tournamentId")
	playerId := params.Get("playerId")

	if playerId == "" || tournamentId == "" {
		errorHandler(w, http.StatusBadRequest, "invalid params given: " + playerId + " " + tournamentId)
		return
	}

	t, err := storage.GetConn().GetTournament(tournamentId)
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}

	p, err := storage.GetConn().GetPlayer(playerId)
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}

	if p.Points < t.Deposit {
		backers, ok := params["backerId"]
		if !ok {
			errorHandler(w, http.StatusBadRequest, "player has insufficient score and no backers provided")
			return
		}

		pointsPerBacker := t.Deposit / uint64(len(backers) + 1)

		for _, backerId := range backers {
			b, err := storage.GetConn().GetPlayer(backerId)
			if err != nil {
				errorHandler(w, http.StatusInternalServerError, err.Error())
				return
			}

			if b.Points < pointsPerBacker {
				errorHandler(w, http.StatusInternalServerError, "backer " + backerId + " has insufficient funds")
				return
			}

			p.Backers = append(p.Backers, backerId)

			b.Points -= pointsPerBacker
			p.Points += pointsPerBacker

			err = storage.GetConn().SafeBack(p, b)
			if err != nil {
				errorHandler(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	p.Points -= t.Deposit
	t.Players[p.Id] = p

	if err = storage.GetConn().SetPlayer(p); err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
	}

	if err = storage.GetConn().SetTournament(t); err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
	}
}

func ResultTournamentHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
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
		errorHandler(w, http.StatusInternalServerError, err.Error())
	}

	for _, winner := range tournamentResult.Winners {
		p, err := storage.GetConn().GetPlayer(winner.PlayerId)
		if err != nil {
			errorHandler(w, http.StatusInternalServerError, err.Error())
			return
		}

		p.Points += winner.Prize
		if err = storage.GetConn().SetPlayer(p); err != nil {
			errorHandler(w, http.StatusInternalServerError, err.Error())
			return
		}

		if len(p.Backers) > 0 {
			pointsPerBacker := winner.Prize / uint64(len(p.Backers) + 1)

			for _, backerId := range p.Backers {
				b, err := storage.GetConn().GetPlayer(backerId)
				if err != nil {
					errorHandler(w, http.StatusInternalServerError, err.Error())
					return
				}

				p.Backers = append(p.Backers, backerId)

				b.Points += pointsPerBacker
				p.Points -= pointsPerBacker

				err = storage.GetConn().SafeBack(p, b)
				if err != nil {
					errorHandler(w, http.StatusInternalServerError, err.Error())
					return
				}
			}
		}
	}
}

func BalanceHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	params := r.URL.Query()
	playerId := params.Get("playerId")

	p, err := storage.GetConn().GetPlayer(playerId)
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}

	j, err := json.Marshal(p)
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Write(j)
}

func ResetHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}

	err := storage.GetConn().Reset()
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
	}
}

func errorHandler(w http.ResponseWriter, httpCode int, msg string)  {
	w.WriteHeader(httpCode)
	w.Write([]byte(msg))
}

func methodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}

	return true
}