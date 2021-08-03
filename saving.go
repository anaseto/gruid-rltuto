// This file handles game saving.

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	// We register Entity types so that gob can encode them.
	gob.Register(&Player{})
	gob.Register(&Monster{})
	gob.Register(&HealingPotion{})
	gob.Register(&LightningScroll{})
	gob.Register(&ConfusionScroll{})
	gob.Register(&FireballScroll{})
}

// EncodeGame uses the gob package of the standard library to encode the game
// so that it can be saved to a file.
func EncodeGame(g *game) ([]byte, error) {
	data := bytes.Buffer{}
	enc := gob.NewEncoder(&data)
	err := enc.Encode(g)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data.Bytes())
	w.Close()
	return buf.Bytes(), nil
}

// DecodeGame uses the gob package from the standard library to decode a saved
// game.
func DecodeGame(data []byte) (*game, error) {
	buf := bytes.NewReader(data)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	dec := gob.NewDecoder(r)
	g := &game{}
	err = dec.Decode(g)
	if err != nil {
		return nil, err
	}
	r.Close()
	return g, nil
}

// DataDir returns the directory for saving application's data, which depends
// on the platform. It builds the directory if it does not exist already.
func DataDir() (string, error) {
	var xdg string
	if runtime.GOOS == "windows" {
		// Windows
		xdg = os.Getenv("LOCALAPPDATA")
	} else {
		// Linux, BSD, etc.
		xdg = os.Getenv("XDG_DATA_HOME")
	}
	if xdg == "" {
		xdg = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	dataDir := filepath.Join(xdg, "gruid-rltuto")
	_, err := os.Stat(dataDir)
	if err != nil {
		err = os.MkdirAll(dataDir, 0755)
		if err != nil {
			return dataDir, fmt.Errorf("building data directory: %v\n", err)
		}
	}
	return dataDir, nil
}

// SaveFile saves data to a file with a given filename. The data is first
// written to a temporary file and then renamed, to avoid corrupting any
// previous file with same filename in case of an error occurs while writing
// the file (for example due to an electric power outage).
func SaveFile(filename string, data []byte) error {
	dataDir, err := DataDir()
	if err != nil {
		return err
	}
	tempSaveFile := filepath.Join(dataDir, "temp-"+filename)
	f, err := os.OpenFile(tempSaveFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	saveFile := filepath.Join(dataDir, filename)
	if err := os.Rename(f.Name(), saveFile); err != nil {
		return err
	}
	return err
}

// LoadFile opens a file with given filename in the game's data directory, and
// returns its content or an error.
func LoadFile(filename string) ([]byte, error) {
	dataDir, err := DataDir()
	if err != nil {
		return nil, fmt.Errorf("could not read game's data directory: %s", dataDir)
	}
	fp := filepath.Join(dataDir, filename)
	_, err = os.Stat(fp)
	if err != nil {
		return nil, fmt.Errorf("no such file: %s", filename)
	}
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// RemoveDataFile removes a file in the game's data directory.
func RemoveDataFile(filename string) error {
	dataDir, err := DataDir()
	if err != nil {
		return err
	}
	dataFile := filepath.Join(dataDir, filename)
	_, err = os.Stat(dataFile)
	if err == nil {
		err := os.Remove(dataFile)
		if err != nil {
			return err
		}
	}
	return nil
}
