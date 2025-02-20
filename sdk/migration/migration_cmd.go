package migration

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"path/filepath"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	cometbftcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/os"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	cometbfttypes "github.com/cometbft/cometbft/types"
	rollkitstore "github.com/rollkit/rollkit/store"
	rollkittypes "github.com/rollkit/rollkit/types"
	"github.com/spf13/cobra"
)

// MigrateToRollkitCmd returns a command that migrates the data from the comnettBFT chain to rollup
func MigrateToRollkitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollup-migration",
		Short: "Migrate the data from the comnettBFT chain to rollup",
		Long:  "Migrate the data from the comnettBFT chain to rollup",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := cometbftcmd.ParseConfig(cmd)
			if err != nil {
				return err
			}

			blockStore, stateStore, err := loadStateAndBlockStore(config)
			if err != nil {
				return err
			}

			cometBFTstate, err := stateStore.Load()
			if err != nil {
				return err
			}

			lastBlockHeight := cometBFTstate.LastBlockHeight
			rollkitStore, err := loadRollkitStateStore(config.RootDir, config.DBPath)
			if err != nil {
				return err
			}

			rollkitState, err := rollkitStateFromCometBFTState(cometBFTstate)
			if err != nil {
				return err
			}

			err = rollkitStore.UpdateState(context.Background(), rollkitState)
			if err != nil {
				return err
			}

			for height := lastBlockHeight; height > 0; height-- {
				block := blockStore.LoadBlock(lastBlockHeight)
				header, data, signature := cometBlockToRollkit(block, cometBFTstate)

				err = rollkitStore.SaveBlockData(context.Background(), header, data, &signature)
				if err != nil {
					return err
				}

				// Only save extended commit info if vote extensions are enabled
				if cometBFTstate.ConsensusParams.ABCI.VoteExtensionsEnabled(block.Height) {
					extendedCommit := blockStore.LoadBlockExtendedCommit(lastBlockHeight)

					extendedCommitInfo := abci.ExtendedCommitInfo{
						Round: extendedCommit.Round,
					}

					for _, vote := range extendedCommit.ToExtendedVoteSet("", cometBFTstate.LastValidators).List() {
						power := int64(0)
						for _, v := range cometBFTstate.LastValidators.Validators {
							if bytes.Equal(v.Address.Bytes(), vote.ValidatorAddress) {
								power = v.VotingPower
								break
							}
						}

						extendedCommitInfo.Votes = append(extendedCommitInfo.Votes, abci.ExtendedVoteInfo{
							Validator: abci.Validator{
								Address: vote.ValidatorAddress,
								Power:   power,
							},
							VoteExtension:      vote.Extension,
							ExtensionSignature: vote.ExtensionSignature,
							BlockIdFlag:        cmtproto.BlockIDFlag(vote.CommitSig().BlockIDFlag),
						})
					}

					rollkitStore.SaveExtendedCommit(context.Background(), header.Height(), &extendedCommitInfo)
				}
			}

			log.Println("Migration completed successfully")
			return nil
		},
	}
	return cmd
}

// cometBlockToRollkit converts a cometBFT block to a rollkit block
func cometBlockToRollkit(block *cometbfttypes.Block, cometBFTstate state.State) (*rollkittypes.SignedHeader, *rollkittypes.Data, rollkittypes.Signature) {
	var (
		header    *rollkittypes.SignedHeader
		data      *rollkittypes.Data
		signature rollkittypes.Signature
	)

	// find proposer signature
	for _, sig := range block.LastCommit.Signatures {
		if bytes.Equal(sig.ValidatorAddress.Bytes(), block.ProposerAddress.Bytes()) {
			signature = sig.Signature
			break
		}
	}

	header = &rollkittypes.SignedHeader{
		Header: rollkittypes.Header{
			BaseHeader: rollkittypes.BaseHeader{
				Height:  uint64(block.Height),
				Time:    uint64(block.Time.UnixNano()),
				ChainID: block.ChainID,
			},
			Version: rollkittypes.Version{
				Block: block.Version.Block,
				App:   block.Version.App,
			},
			LastHeaderHash:  block.LastCommitHash.Bytes(),
			LastCommitHash:  block.LastCommitHash.Bytes(),
			DataHash:        block.DataHash.Bytes(),
			ConsensusHash:   block.ConsensusHash.Bytes(),
			AppHash:         block.AppHash.Bytes(),
			LastResultsHash: block.LastResultsHash.Bytes(),
			ValidatorHash:   block.ValidatorsHash.Bytes(),
			ProposerAddress: block.ProposerAddress.Bytes(),
		},
		Signature: signature, // TODO: figure out this.
		Validators: &cometbfttypes.ValidatorSet{
			Validators: cometBFTstate.Validators.Validators,
			Proposer:   cometBFTstate.Validators.Proposer,
		},
	}

	data = &rollkittypes.Data{
		Metadata: &rollkittypes.Metadata{
			ChainID:      block.ChainID,
			Height:       uint64(block.Height),
			Time:         uint64(block.Time.UnixNano()),
			LastDataHash: block.DataHash.Bytes(),
		},
	}

	for _, tx := range block.Data.Txs {
		data.Txs = append(data.Txs, rollkittypes.Tx(tx))
	}
	return header, data, signature
}

func loadStateAndBlockStore(config *cfg.Config) (*store.BlockStore, state.Store, error) {
	dbType := dbm.BackendType(config.DBBackend)

	if !os.FileExists(filepath.Join(config.DBDir(), "blockstore.db")) {
		return nil, nil, fmt.Errorf("no blockstore found in %v", config.DBDir())
	}

	// Get BlockStore
	blockStoreDB, err := dbm.NewDB("blockstore", dbType, config.DBDir())
	if err != nil {
		return nil, nil, err
	}
	blockStore := store.NewBlockStore(blockStoreDB)

	if !os.FileExists(filepath.Join(config.DBDir(), "state.db")) {
		return nil, nil, fmt.Errorf("no statestore found in %v", config.DBDir())
	}

	// Get StateStore
	stateDB, err := dbm.NewDB("state", dbType, config.DBDir())
	if err != nil {
		return nil, nil, err
	}
	stateStore := state.NewStore(stateDB, state.StoreOptions{
		DiscardABCIResponses: config.Storage.DiscardABCIResponses,
	})

	return blockStore, stateStore, nil
}

func loadRollkitStateStore(rootDir, dbPath string) (rollkitstore.Store, error) {
	baseKV, err := rollkitstore.NewDefaultKVStore(rootDir, dbPath, "rollkit")
	if err != nil {
		return nil, err
	}

	store := rollkitstore.New(baseKV)
	return store, nil
}

func rollkitStateFromCometBFTState(cometBFTState state.State) (rollkittypes.State, error) {
	return rollkittypes.State{
		Version: cometBFTState.Version,

		ChainID:         cometBFTState.ChainID,
		InitialHeight:   uint64(cometBFTState.LastBlockHeight), // The initial height is the migration height
		LastBlockHeight: uint64(cometBFTState.LastBlockHeight),
		LastBlockID:     cometBFTState.LastBlockID,
		LastBlockTime:   cometBFTState.LastBlockTime,

		DAHeight: 1,

		ConsensusParams:                  cometBFTState.ConsensusParams.ToProto(),
		LastHeightConsensusParamsChanged: uint64(cometBFTState.LastHeightConsensusParamsChanged),

		LastResultsHash: cometBFTState.LastResultsHash,
		AppHash:         cometBFTState.AppHash,

		Validators:                  cometBFTState.Validators,
		NextValidators:              cometBFTState.NextValidators,
		LastValidators:              cometBFTState.LastValidators,
		LastHeightValidatorsChanged: cometBFTState.LastHeightValidatorsChanged,
	}, nil
}
