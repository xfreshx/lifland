package storage

import (
	"github.com/xfreshx/lifland/types"
	"errors"
	"github.com/dgraph-io/badger"
	"log"
	"os"
	"sync"
)

var s = &store{}
var once sync.Once

var ErrNoPlayer = errors.New("no such player")
var ErrNoTournament = errors.New("no such tournament")

type store struct {
	db *badger.DB
}

func GetConn() *store {
	once.Do(func() {

		dbPath := os.Getenv("db")
		if dbPath == "" {
			dbPath = "/tmp/lifland"
		}

		opts := badger.DefaultOptions
		opts.Dir = dbPath
		opts.ValueDir = dbPath

		var err error
		s.db, err = badger.Open(opts)
		if err != nil {
			log.Fatal("Unable to open database file (%s): %v", dbPath, err)
		}
	})

	return s
}

func (s *store) Reset() error {
	if s.db == nil {
		return errors.New("storage is not initialized")
	}

	return s.db.DropAll()
}

func (s *store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *store) SetTournament(t *types.Tournament) error {
	return s.db.Update(func(txn *badger.Txn) error {
		b, err := t.ToBytes()
		if err != nil {
			return err
		}

		return txn.Set([]byte("tournament"+t.Id), b)
	})
}

func (s *store) GetTournament(id string) (*types.Tournament, error) {
	var t types.Tournament
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("tournament" + id))
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return t.FromBytes(val)
	})

	if err != nil && err == badger.ErrKeyNotFound {
		err = ErrNoTournament
	}

	return &t, err
}

func (s *store) DeleteTournament(id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("tournament"+id))
	})
}

func (s *store) SetMultiPlayer(pp ...*types.Player) error {
	txn := s.db.NewTransaction(true)
	defer txn.Discard()


	for _, p := range pp {
		b, err := p.ToBytes()
		if err != nil {
			return err
		}

		err = txn.Set([]byte("player"+p.Id), b)
		if err != nil {
			return err
		}
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *store) SetPlayer(p *types.Player) error {
	return s.db.Update(func(txn *badger.Txn) error {
		b, err := p.ToBytes()
		if err != nil {
			return err
		}

		return txn.Set([]byte("player"+p.Id), b)
	})
}

func (s *store) GetPlayer(id string) (*types.Player, error) {
	var p types.Player
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("player" + id))
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return p.FromBytes(val)
	})

	if err != nil && err == badger.ErrKeyNotFound {
		err = ErrNoPlayer
	}

	return &p, err
}