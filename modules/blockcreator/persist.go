package blockcreator

import (
	"os"
	"path/filepath"

	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/persist"
	"github.com/rivine/rivine/types"
)

const (
	logFile      = modules.BlockCreatorDir + ".log"
	settingsFile = modules.BlockCreatorDir + ".json"
)

var (
	settingsMetadata = persist.Metadata{
		Header:  "BlockCreatorDir Settings",
		Version: "0.0.1",
	}
)

type (
	// persist contains all of the persistent miner data.
	persistence struct {
		RecentChange modules.ConsensusChangeID
		Height       types.BlockHeight
		ParentID     types.BlockID
	}
)

// initSettings loads the settings file if it exists and creates it if it
// doesn't.
func (b *BlockCreator) initSettings() error {
	filename := filepath.Join(b.persistDir, settingsFile)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return b.save()
	} else if err != nil {
		return err
	}
	return b.load()
}

// initPersist initializes the persistence of the block creator.
func (b *BlockCreator) initPersist() error {
	// Create the miner directory.
	err := os.MkdirAll(b.persistDir, 0700)
	if err != nil {
		return err
	}

	// Add a logger.
	b.log, err = persist.NewFileLogger(filepath.Join(b.persistDir, logFile))
	if err != nil {
		return err
	}

	return b.initSettings()
}

// load loads the block creator persistence from disk.
func (b *BlockCreator) load() error {
	return persist.LoadFile(settingsMetadata, &b.persist, filepath.Join(b.persistDir, settingsFile))
}

// save saves the block creator persistence to disk.
func (b *BlockCreator) save() error {
	return persist.SaveFile(settingsMetadata, b.persist, filepath.Join(b.persistDir, settingsFile))
}

// saveSync saves the block creator persistence to disk, and then syncs to disk.
func (b *BlockCreator) saveSync() error {
	return persist.SaveFileSync(settingsMetadata, b.persist, filepath.Join(b.persistDir, settingsFile))
}
