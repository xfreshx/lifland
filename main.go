package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/xfreshx/lifland/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {

	db := storage.GetConn()
	defer db.Close()

	r := mux.NewRouter()

	r.HandleFunc("/", RootHandler)
	r.HandleFunc("/take", TakeHandler)
	r.HandleFunc("/fund", FundHandler)
	r.HandleFunc("/announceTournament", AnnounceTournamentHandler)
	r.HandleFunc("/joinTournament", JoinTournamentHandler)
	r.HandleFunc("/resultTournament", ResultTournamentHandler)
	r.HandleFunc("/balance", BalanceHandler)
	r.HandleFunc("/reset", ResetHandler)

	srv := &http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Unable to start server: " + err.Error())
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	srv.Shutdown(ctx)

	log.Println("Shutting down...")
	os.Exit(0)
}
