package sftp

import (
	"errors"
	"sync"

	"github.com/nethack42/go-sftp/sshfxp"
)

type Router struct {
	m sync.Mutex

	routes map[uint32]chan<- sshfxp.Message

	nextID uint32
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[uint32]chan<- sshfxp.Message),
	}
}

func (r *Router) Get(x sshfxp.Message) (uint32, <-chan sshfxp.Message) {
	r.m.Lock()
	defer r.m.Unlock()

	var id uint32

	ch := make(chan sshfxp.Message, 1)

	// Loop until we find a message ID not already in use
	for {
		id = r.nextID
		r.nextID = r.nextID + 1

		if _, ok := r.routes[id]; ok {
			continue
		}

		break
	}

	r.routes[id] = ch

	return id, ch
}

func (r *Router) Resolve(x sshfxp.Message) error {
	r.m.Lock()
	defer r.m.Unlock()

	id := x.Meta().ID

	if res, ok := r.routes[id]; ok {
		res <- x
		return nil
	}

	return errors.New("unknown id")
}
