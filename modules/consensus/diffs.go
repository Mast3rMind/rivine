package consensus

import (
	"errors"

	"github.com/rivine/rivine/build"
	"github.com/rivine/rivine/encoding"
	"github.com/rivine/rivine/modules"

	"github.com/NebulousLabs/bolt"
)

var (
	errApplySiafundPoolDiffMismatch = errors.New("committing a siafund pool diff with an invalid 'previous' field")
	errDiffsNotGenerated            = errors.New("applying diff set before generating errors")
	errInvalidSuccessor             = errors.New("generating diffs for a block that's an invalid successsor to the current block")
	errWrongAppliedDiffSet          = errors.New("applying a diff set that isn't the current block")
	errWrongRevertDiffSet           = errors.New("reverting a diff set that isn't the current block")
)

// commitDiffSetSanity performs a series of sanity checks before committing a
// diff set.
func commitDiffSetSanity(tx *bolt.Tx, pb *processedBlock, dir modules.DiffDirection) {
	// This function is purely sanity checks.
	if !build.DEBUG {
		return
	}

	// Diffs should have already been generated for this node.
	if !pb.DiffsGenerated {
		panic(errDiffsNotGenerated)
	}

	// Current node must be the input node's parent if applying, and
	// current node must be the input node if reverting.
	if dir == modules.DiffApply {
		parent, err := getBlockMap(tx, pb.Block.ParentID)
		if build.DEBUG && err != nil {
			panic(err)
		}
		if parent.Block.ID() != currentBlockID(tx) {
			panic(errWrongAppliedDiffSet)
		}
	} else {
		if pb.Block.ID() != currentBlockID(tx) {
			panic(errWrongRevertDiffSet)
		}
	}
}

// commitCoinOutputDiff applies or reverts a SiacoinOutputDiff.
func commitCoinOutputDiff(tx *bolt.Tx, scod modules.CoinOutputDiff, dir modules.DiffDirection) {
	if scod.Direction == dir {
		addCoinOutput(tx, scod.ID, scod.CoinOutput)
	} else {
		removeCoinOutput(tx, scod.ID)
	}
}

// commitBlockStakeOutputDiff applies or reverts a Siafund output diff.
func commitBlockStakeOutputDiff(tx *bolt.Tx, sfod modules.BlockStakeOutputDiff, dir modules.DiffDirection) {
	if sfod.Direction == dir {
		addBlockStakeOutput(tx, sfod.ID, sfod.BlockStakeOutput)
	} else {
		removeBlockStakeOutput(tx, sfod.ID)
	}
}

// commitNodeDiffs commits all of the diffs in a block node.
func commitNodeDiffs(tx *bolt.Tx, pb *processedBlock, dir modules.DiffDirection) {
	if dir == modules.DiffApply {
		for _, scod := range pb.CoinOutputDiffs {
			commitCoinOutputDiff(tx, scod, dir)
		}
		for _, sfod := range pb.BlockStakeOutputDiffs {
			commitBlockStakeOutputDiff(tx, sfod, dir)
		}
	} else {
		for i := len(pb.CoinOutputDiffs) - 1; i >= 0; i-- {
			commitCoinOutputDiff(tx, pb.CoinOutputDiffs[i], dir)
		}
		for i := len(pb.BlockStakeOutputDiffs) - 1; i >= 0; i-- {
			commitBlockStakeOutputDiff(tx, pb.BlockStakeOutputDiffs[i], dir)
		}
	}
}

// updateCurrentPath updates the current path after applying a diff set.
func updateCurrentPath(tx *bolt.Tx, pb *processedBlock, dir modules.DiffDirection) {
	// Update the current path.
	if dir == modules.DiffApply {
		pushPath(tx, pb.Block.ID())
	} else {
		popPath(tx)
	}
}

// commitDiffSet applies or reverts the diffs in a blockNode.
func commitDiffSet(tx *bolt.Tx, pb *processedBlock, dir modules.DiffDirection) {
	// Sanity checks - there are a few so they were moved to another function.
	if build.DEBUG {
		commitDiffSetSanity(tx, pb, dir)
	}

	commitNodeDiffs(tx, pb, dir)
	updateCurrentPath(tx, pb, dir)
}

// generateAndApplyDiff will verify the block and then integrate it into the
// consensus state. These two actions must happen at the same time because
// transactions are allowed to depend on each other. We can't be sure that a
// transaction is valid unless we have applied all of the previous transactions
// in the block, which means we need to apply while we verify.
func generateAndApplyDiff(tx *bolt.Tx, pb *processedBlock) error {
	// Sanity check - the block being applied should have the current block as
	// a parent.
	if build.DEBUG && pb.Block.ParentID != currentBlockID(tx) {
		panic(errInvalidSuccessor)
	}

	// Validate and apply each transaction in the block. They cannot be
	// validated all at once because some transactions may not be valid until
	// previous transactions have been applied.
	for _, txn := range pb.Block.Transactions {
		err := validTransaction(tx, txn)
		if err != nil {
			return err
		}
		applyTransaction(tx, pb, txn)
	}

	// DiffsGenerated are only set to true after the block has been fully
	// validated and integrated. This is required to prevent later blocks from
	// being accepted on top of an invalid block - if the consensus set ever
	// forks over an invalid block, 'DiffsGenerated' will be set to 'false',
	// requiring validation to occur again. when 'DiffsGenerated' is set to
	// true, validation is skipped, therefore the flag should only be set to
	// true on fully validated blocks.
	pb.DiffsGenerated = true

	// Add the block to the current path and block map.
	bid := pb.Block.ID()
	blockMap := tx.Bucket(BlockMap)
	updateCurrentPath(tx, pb, modules.DiffApply)

	// Sanity check preparation - set the consensus hash at this height so that
	// during reverting a check can be performed to assure consistency when
	// adding and removing blocks. Must happen after the block is added to the
	// path.
	if build.DEBUG {
		pb.ConsensusChecksum = consensusChecksum(tx)
	}

	return blockMap.Put(bid[:], encoding.Marshal(*pb))
}
