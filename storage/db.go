package storage

import (
	"../types"
	"bytes"
	"encoding/gob"
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
		var b bytes.Buffer
		e := gob.NewEncoder(&b)
		if err := e.Encode(t); err != nil {
			return err
		}

		return txn.Set([]byte("tournament"+t.Id), b.Bytes())
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

		b := bytes.NewBuffer(val)
		d := gob.NewDecoder(b)
		if err := d.Decode(&t); err != nil {
			return err
		}

		return nil
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

func (s *store) SetPlayer(p *types.Player) error {
	return s.db.Update(func(txn *badger.Txn) error {
		var b bytes.Buffer
		e := gob.NewEncoder(&b)
		if err := e.Encode(p); err != nil {
			return err
		}

		return txn.Set([]byte("player"+p.Id), b.Bytes())
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

		b := bytes.NewBuffer(val)
		d := gob.NewDecoder(b)
		if err := d.Decode(&p); err != nil {
			return err
		}

		return nil
	})

	if err != nil && err == badger.ErrKeyNotFound {
		err = ErrNoPlayer
	}

	return &p, err
}

func (s *store) SafeBack(player, backer *types.Player) error {

	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	var pBuf, bBuf bytes.Buffer
	e := gob.NewEncoder(&pBuf)
	if err := e.Encode(player); err != nil {
		return err
	}

	err := txn.Set([]byte("player" + player.Id), pBuf.Bytes())
	if err != nil {
		return err
	}

	e = gob.NewEncoder(&bBuf)
	if err := e.Encode(backer); err != nil {
		return err
	}

	err = txn.Set([]byte("player" + backer.Id), bBuf.Bytes())
	if err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}