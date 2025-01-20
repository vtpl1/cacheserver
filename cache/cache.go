package cache

import (
	"errors"
)

type Cache struct {
	requests chan request
	// cache map[string]*entry
	// sync.Mutex
}

type request struct {
	key      string
	response chan result
}

type resultValue []byte

type result struct {
	value resultValue
	err   error
}

type entry struct {
	res   result
	ready chan struct{}
}

type Func func(key string) (resultValue, error)

func NewCache(f Func) *Cache {
	cache := &Cache{requests: make(chan request)}
	go cache.server(f)
	return cache
}

func (c *Cache) Get(key string) (resultValue, error) {
	response := make(chan result)
	c.requests <- request{key, response}
	res := <-response
	return res.value, res.err
}

func (c *Cache) server(f Func) {
	cache := make(map[string]*entry)
	for req := range c.requests {
		e, ok := cache[req.key]
		if !ok {
			e = &entry{ready: make(chan struct{})}
			// fmt.Printf("Adding to list %s\n", req.key)
			cache[req.key] = e
			go e.call(f, req.key)

		} else {
			// fmt.Printf("In the list %s\n", req.key)
		}
		go e.deliver(req.response)
	}
}

func (e *entry) call(f Func, key string) {
	defer close(e.ready)
	defer func(e *entry) {
		if r := recover(); r != nil {
			e.res.err = errors.New("Recovered in f")
		}
	}(e)
	e.res.value, e.res.err = f(key)
}

func (e *entry) deliver(response chan<- result) {
	<-e.ready
	response <- e.res
}
