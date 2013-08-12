package main

import (
	"encoding/gob"
	"errors"
	"os"
	"sync"
)

type MultiMutex struct {
	Global sync.RWMutex
	local  map[string]*sync.RWMutex
}

func NewMultiMutex() (mm *MultiMutex) {
	mm = new(MultiMutex)
	mm.local = make(map[string]*sync.RWMutex)
	return
}

func (mm *MultiMutex) Lock(key string) {
	mm.Global.Lock()
	defer mm.Global.Unlock()
	if m, exists := mm.local[key]; exists {
		m.Lock()
	} else {
		m := &sync.RWMutex{}
		m.Lock()
		mm.local[key] = m
	}
}

func (mm *MultiMutex) Unlock(key string) error {
	mm.Global.Lock()
	defer mm.Global.Unlock()
	if m, exists := mm.local[key]; exists {
		m.Unlock()
		delete(mm.local, key)
		return nil
	} else {
		return errors.New("lock not present!")
	}
}

type Datastore struct {
	name    string
	data    map[string]string
	mutexes *MultiMutex
}

func (ds *Datastore) create(name string) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return
	}
	defer f.Close()
	ds.name = name
	ds.data = make(map[string]string)
	ds.mutexes = NewMultiMutex()
	return
}

func (ds *Datastore) load(name string) (err error) {
	f, err := os.Open(name)
	if err != nil {
		return
	}
	defer f.Close()

	ds.name = name
	ds.mutexes = NewMultiMutex()
	dec := gob.NewDecoder(f)
	err = dec.Decode(&ds.data)
	return
}

func (ds *Datastore) save() (err error) {
	f, err := os.Create(ds.name)
	if err != nil {
		return
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	err = enc.Encode(&ds.data)
	return
}

func OpenDatastore(name string) (ds *Datastore, err error) {
	ds = new(Datastore)
	if err = ds.load(name); os.IsNotExist(err) {
		err = ds.create(name)
	}
	return
}

func (ds *Datastore) Close() error {
	ds.mutexes.Global.Lock()
	defer ds.mutexes.Global.Unlock()
	return ds.save()
}

func (ds *Datastore) Get(key string) (value string, err error) {
	ds.mutexes.Lock(key)
	defer ds.mutexes.Unlock(key)
	value, exists := ds.data[key]
	if exists {
		return value, nil
	} else {
		return value, errors.New("key not present!")
	}
}

func (ds *Datastore) Set(key, value string) (err error) {
	ds.mutexes.Lock(key)
	defer ds.mutexes.Unlock(key)
	ds.data[key] = value
	return
}

func (ds *Datastore) Delete(key string) (err error) {
	ds.mutexes.Lock(key)
	defer ds.mutexes.Unlock(key)
	delete(ds.data, key)
	return
}
