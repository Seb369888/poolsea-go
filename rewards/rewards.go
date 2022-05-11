package rewards

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/rocket-pool/rocketpool-go/rocketpool"
	"github.com/rocket-pool/rocketpool-go/utils/eth"
)

// Info for a rewards snapshot event
type RewardsEvent struct {
	Index                *big.Int
	Block                *big.Int
	RewardsPerNetworkRPL []*big.Int
	RewardsPerNetworkETH []*big.Int
	MerkleRoot           common.Hash
	MerkleTreeCID        string
	IntervalStartTime    time.Time
	IntervalEndTime      time.Time
}

// Get the index of the active rewards period
func GetRewardIndex(rp *rocketpool.RocketPool, opts *bind.CallOpts) (*big.Int, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return nil, err
	}
	index := new(*big.Int)
	if err := rocketRewardsPool.Call(opts, index, "getRewardIndex"); err != nil {
		return nil, fmt.Errorf("Could not get current reward index: %w", err)
	}
	return *index, nil
}

// Get the timestamp that the current rewards interval started
func GetClaimIntervalTimeStart(rp *rocketpool.RocketPool, opts *bind.CallOpts) (time.Time, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return time.Time{}, err
	}
	unixTime := new(*big.Int)
	if err := rocketRewardsPool.Call(opts, unixTime, "getClaimIntervalTimeStart"); err != nil {
		return time.Time{}, fmt.Errorf("Could not get claim interval time start: %w", err)
	}
	return time.Unix((*unixTime).Int64(), 0), nil
}

// Get the number of seconds in a claim interval
func GetClaimIntervalTime(rp *rocketpool.RocketPool, opts *bind.CallOpts) (time.Duration, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return 0, err
	}
	unixTime := new(*big.Int)
	if err := rocketRewardsPool.Call(opts, unixTime, "getClaimIntervalTime"); err != nil {
		return 0, fmt.Errorf("Could not get claim interval time: %w", err)
	}
	return time.Duration((*unixTime).Int64()) * time.Second, nil
}

// Get the percent of checkpoint rewards that goes to node operators
func GetNodeOperatorRewardsPercent(rp *rocketpool.RocketPool, opts *bind.CallOpts) (*big.Int, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return nil, err
	}
	perc := new(*big.Int)
	if err := rocketRewardsPool.Call(opts, perc, "getClaimingContractPerc", "rocketClaimNode"); err != nil {
		return nil, fmt.Errorf("Could not get node operator rewards percent: %w", err)
	}
	return *perc, nil
}

// Get the percent of checkpoint rewards that goes to ODAO members
func GetTrustedNodeOperatorRewardsPercent(rp *rocketpool.RocketPool, opts *bind.CallOpts) (*big.Int, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return nil, err
	}
	perc := new(*big.Int)
	if err := rocketRewardsPool.Call(opts, perc, "getClaimingContractPerc", "rocketClaimTrustedNode"); err != nil {
		return nil, fmt.Errorf("Could not get trusted node operator rewards percent: %w", err)
	}
	return *perc, nil
}

// Get the amount of RPL rewards that will be provided to node operators
func GetPendingRPLRewards(rp *rocketpool.RocketPool, opts *bind.CallOpts) (*big.Int, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return nil, err
	}
	rewards := new(*big.Int)
	if err := rocketRewardsPool.Call(opts, rewards, "getPendingRPLRewards"); err != nil {
		return nil, fmt.Errorf("Could not get pending RPL rewards: %w", err)
	}
	return *rewards, nil
}

// Get the amount of ETH rewards that will be provided to node operators
func GetPendingETHRewards(rp *rocketpool.RocketPool, opts *bind.CallOpts) (*big.Int, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return nil, err
	}
	rewards := new(*big.Int)
	if err := rocketRewardsPool.Call(opts, rewards, "getPendingETHRewards"); err != nil {
		return nil, fmt.Errorf("Could not get pending ETH rewards: %w", err)
	}
	return *rewards, nil
}

// Estimate the gas for submiting a Merkle Tree-based snapshot for a rewards interval
func EstimateSubmitRewardSnapshotGas(rp *rocketpool.RocketPool, index *big.Int, block *big.Int, rewardsPerNetworkRPL []*big.Int, rewardsPerNetworkETH []*big.Int, merkleRoot common.Hash, merkleTreeCID string, intervalsPassed *big.Int, opts *bind.TransactOpts) (rocketpool.GasInfo, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return rocketpool.GasInfo{}, err
	}
	return rocketRewardsPool.GetTransactionGasInfo(opts, "submitRewardSnapshot", index, block, rewardsPerNetworkRPL, rewardsPerNetworkETH, merkleRoot, merkleTreeCID, intervalsPassed)
}

// Submit a Merkle Tree-based snapshot for a rewards interval
func SubmitRewardSnapshot(rp *rocketpool.RocketPool, index *big.Int, block *big.Int, rewardsPerNetworkRPL []*big.Int, rewardsPerNetworkETH []*big.Int, merkleRoot common.Hash, merkleTreeCID string, intervalsPassed *big.Int, opts *bind.TransactOpts) (common.Hash, error) {
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return common.Hash{}, err
	}
	hash, err := rocketRewardsPool.Transact(opts, "submitRewardSnapshot", index, block, rewardsPerNetworkRPL, rewardsPerNetworkETH, merkleRoot, merkleTreeCID, intervalsPassed)
	if err != nil {
		return common.Hash{}, fmt.Errorf("Could not submit rewards snapshot: %w", err)
	}
	return hash, nil
}

// Get the event info for a rewards snapshot
func GetRewardSnapshotEvent(rp *rocketpool.RocketPool, index uint64, intervalSize *big.Int, startBlock *big.Int) (RewardsEvent, error) {
	// Get contracts
	rocketRewardsPool, err := getRocketRewardsPool(rp)
	if err != nil {
		return RewardsEvent{}, err
	}

	// Construct a filter query for relevant logs
	indexBig := big.NewInt(0).SetUint64(index)
	indexBytes := [32]byte{}
	indexBig.FillBytes(indexBytes[:])
	addressFilter := []common.Address{*rocketRewardsPool.Address}
	topicFilter := [][]common.Hash{{rocketRewardsPool.ABI.Events["RewardSnapshot"].ID}, {indexBytes}}

	// Get the event logs
	logs, err := eth.GetLogs(rp, addressFilter, topicFilter, intervalSize, startBlock, nil, nil)
	if err != nil {
		return RewardsEvent{}, err
	}

	// Get the log info
	values := make(map[string]interface{})
	if len(logs) == 0 {
		return RewardsEvent{}, fmt.Errorf("reward snapshot for interval %d not found", index)
	}
	if rocketRewardsPool.ABI.Events["RewardSnapshot"].Inputs.UnpackIntoMap(values, logs[0].Data) != nil {
		return RewardsEvent{}, err
	}

	// Get the decoded data
	eventBlock := values["block"].(*big.Int)
	eventRpl := values["rewardsPerNetworkRPL"].([]*big.Int)
	eventEth := values["rewardsPerNetworkETH"].([]*big.Int)
	eventMerkleRoot := values["merkleRoot"].([32]byte)
	eventMerkleTreeCid := values["merkleTreeCID"].(string)
	eventIntervalStartTime := values["intervalStartTime"].(*big.Int)
	eventIntervalEndTime := values["intervalEndTime"].(*big.Int)
	eventData := RewardsEvent{
		Index:                indexBig,
		Block:                eventBlock,
		RewardsPerNetworkRPL: eventRpl,
		RewardsPerNetworkETH: eventEth,
		MerkleRoot:           common.BytesToHash(eventMerkleRoot[:]),
		MerkleTreeCID:        eventMerkleTreeCid,
		IntervalStartTime:    time.Unix(eventIntervalStartTime.Int64(), 0),
		IntervalEndTime:      time.Unix(eventIntervalEndTime.Int64(), 0),
	}

	return eventData, nil

}

// Get contracts
var rocketRewardsPoolLock sync.Mutex

func getRocketRewardsPool(rp *rocketpool.RocketPool) (*rocketpool.Contract, error) {
	rocketRewardsPoolLock.Lock()
	defer rocketRewardsPoolLock.Unlock()
	return rp.GetContract("rocketRewardsPool")
}
