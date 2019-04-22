package types

import (
	"bytes"
	"encoding/gob"
)

type Player struct {
	Id     string `json:"id"`
	Points uint64 `json:"balance"`
	Backers map[string]bool `json:"-"`
}

func (p *Player) ToBytes() ([]byte, error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	if err := e.Encode(p); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (p *Player) FromBytes(val []byte) error {
	b := bytes.NewBuffer(val)
	d := gob.NewDecoder(b)

	return d.Decode(p)
}

type Tournament struct {
	Id      string
	Players []string
	Deposit uint64
}

func (t *Tournament) ToBytes() ([]byte, error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	if err := e.Encode(t); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (t *Tournament) FromBytes(val []byte) error {
	b := bytes.NewBuffer(val)
	d := gob.NewDecoder(b)

	return d.Decode(t)
}
