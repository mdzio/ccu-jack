package rtcfg

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mdzio/go-logging"
)

const writeDelay = 3000 * time.Millisecond

var log = logging.Get("rtcfg")

// Store holds a runtime configuration.
type Store struct {
	FileName string
	Config   Config
	timer    *time.Timer
	modified bool
	mtx      sync.RWMutex
}

// Read loads the runtime config from file.
func (s *Store) Read() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	// open file
	file, err := os.Open(s.FileName)
	if err != nil {
		return fmt.Errorf("Opening of configuration file %s failed: %v", s.FileName, err)
	}
	defer file.Close()
	// read file
	dec := json.NewDecoder(file)
	err = dec.Decode(&s.Config)
	if err != nil {
		return fmt.Errorf("Reading of configuration file %s failed: %v", s.FileName, err)
	}
	log.Infof("Configuration loaded from file: %s", s.FileName)
	// configure hostname, if missing
	if s.Config.Host.Name == "" {
		name, err := os.Hostname()
		if err != nil {
			return err
		}
		s.Config.Host.Name = name
		s.modified = true
	}
	// encrypt passwords
	for _, u := range s.Config.Users {
		if u.Password != "" {
			u.SetPassword(u.Password)
			s.modified = true
		}
	}
	// configure BINRPC, if missing
	if s.Config.BINRPC.Port == 0 {
		s.Config.BINRPC.Port = 2123
		s.modified = true
	}
	// save, if modified
	if s.modified {
		s.delayedWrite()
	}
	return nil
}

// Write stores the runtime config immediately into file.
func (s *Store) Write() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	// stop timer
	if s.timer != nil {
		s.timer.Stop()
	}
	// save to file
	if s.modified {
		// open file
		file, err := os.Create(s.FileName)
		if err != nil {
			return fmt.Errorf("Opening of configuration file %s failed: %v", s.FileName, err)
		}
		defer file.Close()
		// write file
		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		err = enc.Encode(s.Config)
		if err != nil {
			return fmt.Errorf("Writing of configuration file %s failed: %v", s.FileName, err)
		}
		s.modified = false
		log.Debugf("Configuration saved to file: %s", s.FileName)
	}
	return nil
}

// Close discards a pending write operation.
func (s *Store) Close() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	// stop timer
	if s.timer != nil {
		s.timer.Stop()
	}
	s.modified = false
}

// Lock locks the store for writing (q.v. Locker interface).
func (s *Store) Lock() {
	s.mtx.Lock()
}

// Unlock unlocks the store (q.v. Locker interface).
func (s *Store) Unlock() {
	s.modified = true
	s.delayedWrite()
	s.mtx.Unlock()
}

func (s *Store) delayedWrite() {
	// start delayed write
	if s.timer != nil {
		s.timer.Reset(writeDelay)
	} else {
		s.timer = time.AfterFunc(writeDelay, func() {
			err := s.Write()
			if err != nil {
				log.Error(err)
			}
		})
	}
}

// RLock locks the store for reading.
func (s *Store) RLock() {
	s.mtx.RLock()
}

// RUnlock unlocks the store.
func (s *Store) RUnlock() {
	s.mtx.RUnlock()
}

// RLocker returns a Locker interface for RLock and RUnlock.
func (s *Store) RLocker() sync.Locker {
	return s.mtx.RLocker()
}

// View executes a function which reads the runtime config.
func (s *Store) View(fn func(*Config) error) error {
	s.RLock()
	defer s.RUnlock()
	return fn(&s.Config)
}

// Update executes a function which updates the runtime config. If fn returns no
// error, a delayed save to file is triggered.
func (s *Store) Update(fn func(*Config) error) error {
	s.Lock()
	defer s.Unlock()
	// execute modifying function
	if err := fn(&s.Config); err != nil {
		return err
	}
	return nil
}
