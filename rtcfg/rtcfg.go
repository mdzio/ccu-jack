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
	root     Root
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
	err = dec.Decode(&s.root)
	if err != nil {
		return fmt.Errorf("Reading of configuration file %s failed: %v", s.FileName, err)
	}
	log.Infof("Configuration loaded from file: %s", s.FileName)
	return nil
}

// Write stores the runtime config immediately into file.
func (s *Store) Write() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	// stop timer
	s.timer.Stop()
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
		err = enc.Encode(s.root)
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
	s.timer.Stop()
	s.modified = false
}

// View executes a function which reads the runtime config.
func (s *Store) View(fn func(*Root) error) error {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return fn(&s.root)
}

// Update executes a function which updates the runtime config. If fn returns no
// error, a delayed save to file is triggered.
func (s *Store) Update(fn func(*Root) error) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	// execute modifying function
	if err := fn(&s.root); err != nil {
		return err
	}
	s.modified = true
	// start delayed save
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
	return nil
}
