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

func (r *Router) Get() (uint32, <-chan sshfxp.Message) {
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

func (r *Router) Resolve(payload interface{}) error {
	r.m.Lock()
	defer r.m.Unlock()

	var x sshfxp.Header
	var ok bool

	if x, ok = payload.(sshfxp.Header); !ok {
		return errors.New("payload must be of type sshfxp.Header")
	}

	if _, ok := payload.(sshfxp.Message); !ok {
		return errors.New("payload must be of type sshfxp.Message")
	}

	if res, ok := r.routes[x.GetID()]; ok {
		delete(r.routes, x.GetID())
		go func() {
			res <- payload.(sshfxp.Message)
		}()
		return nil
	}

	return errors.New("unknown id")
}
