package storage

import (
"database/sql"
"errors"
_ "github.com/lib/pq"
"github.com/rubenv/sql-migrate"
"github.com/xfreshx/lifland/types"
"log"
"os"
"strings"
"sync"
)

//TODO: go-bindata -pkg storage migrations/... must be included in a build process

var s = &store{}
var once sync.Once

type store struct {
	db *sql.DB
}

func GetConn() *store {
	once.Do(func() {

		dbConnStr := os.Getenv("db")
		if dbConnStr == "" {
			dbConnStr = "postgres://postgres:example@127.0.0.1:5432/lifland?sslmode=disable&connect_timeout=10"
		}

		var err error
		s.db, err = sql.Open("postgres", dbConnStr)
		if err != nil {
			log.Fatal("Unable to open database file: ", err)
		}

		migrations := &migrate.AssetMigrationSource{
			Asset:    Asset,
			AssetDir: AssetDir,
			Dir:      "migrations",
		}

		_, err = migrate.Exec(s.db, "postgres", migrations, migrate.Up)
		if err != nil {
			log.Fatal("Unable to migrate: ", err)
		}
	})

	return s
}

func (s *store) Reset() error {
	if s.db == nil {
		return errors.New("storage is not initialized")
	}

	_, err := s.db.Exec("DELETE FROM players;")
	if err != nil {
		return err
	}

	_, err = s.db.Exec("DELETE FROM tournaments;")
	if err != nil {
		return err
	}

	return nil
}

func (s *store) Close() {
	if s.db != nil {
		_ = s.db.Close()
	}
}

func (s *store) SetTournament(t *types.Tournament) error {
	if s.db == nil {
		return errors.New("storage is not initialized")
	}

	var commit bool

	_, err := s.db.Exec("BEGIN;")
	if err != nil {
		return err
	}
	defer s.FinalizeTransaction(&commit)


	stmt, err := s.db.Prepare(
		`INSERT INTO tournaments (id, deposit, players) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (id)
			DO UPDATE 
				SET deposit = EXCLUDED.deposit, players = EXCLUDED.players;`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(t.Id, t.Deposit, t.GetPlayersJson())
	if err != nil {
		return err
	}

	commit = true
	return nil
}

func (s *store) GetTournamentForUpdate(id string) (*types.Tournament, error) {
	var t types.Tournament

	if s.db == nil {
		return &t, errors.New("storage is not initialized")
	}

	stmt, err := s.db.Prepare("SELECT * FROM tournaments WHERE id = $1 FOR UPDATE;")
	if err != nil {
		return &t, err
	}

	playersStr := sql.NullString{}
	err = stmt.QueryRow(id).Scan(&t.Id, &t.Deposit, &playersStr)
	if err != nil {
		return &t, err
	}

	t.SetPlayers(playersStr.String)

	return &t, nil
}

func (s *store) UpdateTournament(t *types.Tournament) error {
	if s.db == nil {
		return errors.New("storage is not initialized")
	}

	stmt, err := s.db.Prepare(`UPDATE tournaments SET deposit = $2, players = $3 WHERE id = $1;`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(t.Id, t.Deposit, t.GetPlayersJson())
	if err != nil {
		return err
	}

	return nil
}

func (s *store) DeleteTournament(id string) error {
	if s.db == nil {
		return errors.New("storage is not initialized")
	}

	var commit bool
	_, err := s.db.Exec("BEGIN;")
	if err != nil {
		return err
	}
	defer s.FinalizeTransaction(&commit)

	stmt, err := s.db.Prepare("DELETE FROM tournaments WHERE id = $1;")
	if err != nil {
		return err
	}

	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}

	commit = true
	return nil
}

func (s *store) GetPlayersForUpdate(ids []string) ([]*types.Player, error) {
	pp := []*types.Player{}

	if s.db == nil {
		return pp, errors.New("storage is not initialized")
	}

	stmt, err := s.db.Prepare("SELECT * FROM players WHERE id = ANY($1::text[]) FOR UPDATE;")
	if err != nil {
		return pp, err
	}

	params := "{" + strings.Join(ids, ",") + "}"

	rows, err := stmt.Query(params)
	if err != nil {
		return pp, err
	}

	for rows.Next() {
		p := new(types.Player)

		backersStr := sql.NullString{}
		err = rows.Scan(&p.Id, &p.Points, &backersStr)
		if err != nil {
			log.Println(err)
			continue
		}

		p.SetBackers(backersStr.String)

		pp = append(pp, p)
	}

	return pp, nil
}

func (s *store) GetPlayer(id string) (*types.Player, error) {
	var p types.Player

	if s.db == nil {
		return &p, errors.New("storage is not initialized")
	}

	stmt, err := s.db.Prepare("SELECT * FROM players WHERE id = $1;")
	if err != nil {
		return &p, err
	}

	backersStr := sql.NullString{}
	err = stmt.QueryRow(id).Scan(&p.Id, &p.Points, &backersStr)
	if err != nil {
		return &p, err
	}

	p.SetBackers(backersStr.String)

	return &p, err
}

func (s *store) UpdatePlayer(p *types.Player) error {
	if s.db == nil {
		return errors.New("storage is not initialized")
	}

	stmt, err := s.db.Prepare(`UPDATE players SET points = $2, backers = $3 WHERE id = $1;`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(p.Id, p.Points, p.GetBackersJson())
	if err != nil {
		return err
	}

	return nil
}

func (s *store) SetPlayer(p *types.Player) error {
	if s.db == nil {
		return errors.New("storage is not initialized")
	}

	var commit bool

	_, err := s.db.Exec("BEGIN;")
	if err != nil {
		return err
	}
	defer s.FinalizeTransaction(&commit)


	stmt, err := s.db.Prepare(
		`INSERT INTO players (id, points, backers) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (id)
			DO UPDATE 
				SET points = players.points + EXCLUDED.points, backers = EXCLUDED.backers;`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(p.Id, p.Points, p.GetBackersJson())
	if err != nil {
		return err
	}

	commit = true
	return nil
}


func (s *store) BeginTransaction() error {
	if s.db == nil {
		return errors.New("storage is not initialized")
	}

	_, err := s.db.Exec("BEGIN;")
	if err != nil {
		return err
	}

	return nil
}

func  (s *store) FinalizeTransaction(commit *bool) {
	var err error
	if *commit {
		_, err = s.db.Exec("COMMIT;")
	} else {
		_, err = s.db.Exec("ROLLBACK;")
	}

	if err != nil {
		log.Println(err)
	}
}