package types

type Player struct {
	Id     string `json:"id"`
	Points uint64 `json:"balance"`
	Backers []string `json:"-"`
}

type Tournament struct {
	Id      string
	Players map[string]*Player
	Deposit uint64
}
