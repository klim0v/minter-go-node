package validators

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/MinterTeam/minter-go-node/coreV2/dao"
	"github.com/MinterTeam/minter-go-node/coreV2/developers"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/candidates"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/upgrades"
	"github.com/cosmos/iavl"

	"math/big"
)

const (
	mainPrefix        = byte('v')
	totalStakePrefix  = byte('s')
	accumRewardPrefix = byte('r')
)

const (
	ValidatorMaxAbsentWindow = 24
	validatorMaxAbsentTimes  = 12
)

// Validators struct is a store of Validators state
type Validators struct {
	list    []*Validator
	removed map[types.Pubkey]struct{}
	loaded  bool

	db   atomic.Value
	bus  *bus.Bus
	lock sync.RWMutex
}

// RValidators interface represents Validator state
type RValidators interface {
	GetValidators() []*Validator
	Export(state *types.AppState)
	GetByPublicKey(pubKey types.Pubkey) *Validator
	LoadValidators()
	GetByTmAddress(address types.TmAddress) *Validator
}

// NewValidators returns newly created Validators state with a given bus and iavl
func NewValidators(bus *bus.Bus, db *iavl.ImmutableTree) *Validators {
	immutableTree := atomic.Value{}
	loaded := false
	if db != nil {
		immutableTree.Store(db)
	} else {
		loaded = true
	}
	validators := &Validators{db: immutableTree, bus: bus, loaded: loaded}
	validators.bus.SetValidators(NewBus(validators))
	return validators
}

func (v *Validators) immutableTree() *iavl.ImmutableTree {
	db := v.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (v *Validators) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	if v.immutableTree() == nil && v.loaded {
		v.loaded = false
	}
	v.db.Store(immutableTree)
}

// Commit writes changes to iavl, may return an error
func (v *Validators) Commit(db *iavl.MutableTree, version int64) error {
	if v.hasDirtyValidators() { // todo move check lost to range
		v.lock.RLock()
		data, err := rlp.EncodeToBytes(v.list)
		v.lock.RUnlock()
		if err != nil {
			return fmt.Errorf("can't encode validators: %v", err)
		}

		path := []byte{mainPrefix}
		db.Set(path, data)
	}

	v.lock.Lock()
	defer v.lock.Unlock()

	for _, val := range v.list {
		if v.IsDirtyOrDirtyTotalStake(val) {
			path := []byte{mainPrefix}

			val.lock.Lock()
			path = append(path, val.PubKey.Bytes()...)
			val.lock.Unlock()

			path = append(path, totalStakePrefix)
			db.Set(path, val.GetTotalBipStake().Bytes())
		}

		if v.IsDirtyOrDirtyAccumReward(val) {
			path := []byte{mainPrefix}

			val.lock.Lock()
			path = append(path, val.PubKey.Bytes()...)
			val.lock.Unlock()

			path = append(path, accumRewardPrefix)
			db.Set(path, val.GetAccumReward().Bytes())
		}
	}

	for _, pubkey := range v.getOrderedRemoved() {
		path := append([]byte{mainPrefix}, pubkey.Bytes()...)
		db.Remove(append(path, totalStakePrefix))
		db.Remove(append(path, accumRewardPrefix))
	}
	v.removed = map[types.Pubkey]struct{}{}

	v.uncheckDirtyValidators()

	return nil
}

func (v *Validators) Count() int {
	v.lock.Lock()
	defer v.lock.Unlock()

	return len(v.list)
}

func (v *Validators) TotalStakes() *big.Int {
	vals := v.GetValidators()

	v.lock.Lock()
	defer v.lock.Unlock()

	totalStakes := big.NewInt(0)
	for _, validator := range vals {
		totalStakes.Add(totalStakes, validator.GetTotalBipStake())
	}
	return totalStakes
}

func (v *Validators) getOrderedRemoved() []types.Pubkey {
	keys := make([]types.Pubkey, 0, len(v.removed))
	for k := range v.removed {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}

func (v *Validators) IsDirtyOrDirtyAccumReward(val *Validator) bool {
	val.lock.RLock()
	defer val.lock.RUnlock()

	return val.isDirty || val.isAccumRewardDirty
}

func (v *Validators) IsDirtyOrDirtyTotalStake(val *Validator) bool {
	val.lock.RLock()
	defer val.lock.RUnlock()

	return val.isDirty || val.isTotalStakeDirty
}

// SetValidatorPresent marks validator as present at current height
func (v *Validators) SetValidatorPresent(height uint64, address types.TmAddress) {
	validator := v.GetByTmAddress(address)
	if validator == nil {
		return
	}
	validator.SetPresent(height)
}

// SetValidatorAbsent marks validator as absent at current height
// if validator misses signs of more than validatorMaxAbsentTimes, it will receive penalty and will be swithed off
func (v *Validators) SetValidatorAbsent(height uint64, address types.TmAddress, grace *upgrades.Grace) {
	validator := v.GetByTmAddress(address)
	if validator == nil {
		return
	}

	validator.SetAbsent(height)

	if validator.CountAbsentTimes() > validatorMaxAbsentTimes {
		if !grace.IsGraceBlock(height) {
			v.punishValidator(height, address)
		}

		v.turnValidatorOff(address)
	}
}

// GetValidators returns list of validators
func (v *Validators) GetValidators() []*Validator {
	v.lock.RLock()
	defer v.lock.RUnlock()

	return v.list
}

// SetNewValidators updated validators list with new candidates
func (v *Validators) SetNewValidators(candidates []*candidates.Candidate) {
	old := v.GetValidators()

	oldValidatorsForRemove := map[types.Pubkey]struct{}{}
	for _, oldVal := range old {
		oldVal.lock.RLock()
		oldValidatorsForRemove[oldVal.PubKey] = struct{}{}
		oldVal.lock.RUnlock()
	}

	var newVals []*Validator
	for _, candidate := range candidates {
		accumReward := big.NewInt(0)
		absentTimes := types.NewBitArray(ValidatorMaxAbsentWindow)

		for _, oldVal := range old {
			if oldVal.GetAddress() == candidate.GetTmAddress() {
				oldVal.lock.RLock()
				accumReward = oldVal.accumReward
				absentTimes = oldVal.AbsentTimes
				delete(oldValidatorsForRemove, oldVal.PubKey)
				oldVal.lock.RUnlock()
			}
		}

		newVals = append(newVals, &Validator{
			PubKey:             candidate.PubKey,
			AbsentTimes:        absentTimes,
			totalStake:         candidate.GetTotalBipStake(),
			accumReward:        accumReward,
			isDirty:            true,
			isTotalStakeDirty:  true,
			isAccumRewardDirty: true,
			tmAddress:          candidate.GetTmAddress(),
			bus:                v.bus,
		})
	}

	v.lock.Lock()
	v.removed = oldValidatorsForRemove
	v.lock.Unlock()

	v.SetValidators(newVals)
}

// PunishByzantineValidator find validator with given tmAddress and punishes it:
// 1. Set total stake 0
// 2. Drop validator
func (v *Validators) PunishByzantineValidator(tmAddress [20]byte) {
	validator := v.GetByTmAddress(tmAddress)
	if validator != nil {
		validator.SetTotalBipStake(big.NewInt(0))

		validator.lock.Lock()
		validator.toDrop = true
		validator.isDirty = true
		validator.lock.Unlock()
	}
}

// Create creates a new validator with given params and adds it to state
func (v *Validators) Create(pubkey types.Pubkey, stake *big.Int) {
	val := &Validator{
		PubKey:             pubkey,
		AbsentTimes:        types.NewBitArray(ValidatorMaxAbsentWindow),
		totalStake:         big.NewInt(0).Set(stake),
		accumReward:        big.NewInt(0),
		isDirty:            true,
		isTotalStakeDirty:  true,
		isAccumRewardDirty: true,
	}

	val.setTmAddress()

	v.lock.RLock()
	defer v.lock.RUnlock()
	v.list = append(v.list, val)
}

// PayRewardsV3
// Deprecated
func (v *Validators) PayRewardsV3(height uint64, period int64) (moreRewards *big.Int) {
	moreRewards = big.NewInt(0)

	vals := v.GetValidators()

	calcReward, safeReward := v.bus.App().Reward()
	for _, validator := range vals {
		candidate := v.bus.Candidates().GetCandidate(validator.PubKey)

		totalReward := big.NewInt(0).Set(validator.GetAccumReward())
		remainder := big.NewInt(0).Set(validator.GetAccumReward())

		// pay commission to DAO
		DAOReward := big.NewInt(0).Set(totalReward)
		DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
		DAOReward.Div(DAOReward, big.NewInt(100))

		candidate.AddUpdate(types.GetBaseCoinID(), DAOReward, DAOReward, dao.Address)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), DAOReward)

		remainder.Sub(remainder, DAOReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleDAO.String(),
			Address:         dao.Address,
			Amount:          DAOReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		// pay commission to Developers
		DevelopersReward := big.NewInt(0).Set(totalReward)
		DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
		DevelopersReward.Div(DevelopersReward, big.NewInt(100))

		candidate.AddUpdate(types.GetBaseCoinID(), DevelopersReward, DevelopersReward, developers.Address)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), DevelopersReward)

		remainder.Sub(remainder, DevelopersReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleDevelopers.String(),
			Address:         developers.Address,
			Amount:          DevelopersReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		totalReward.Sub(totalReward, DevelopersReward)
		totalReward.Sub(totalReward, DAOReward)

		// pay commission to validator
		validatorReward := big.NewInt(0).Set(totalReward)
		validatorReward.Mul(validatorReward, big.NewInt(int64(candidate.Commission)))
		validatorReward.Div(validatorReward, big.NewInt(100))
		totalReward.Sub(totalReward, validatorReward)

		candidate.AddUpdate(types.GetBaseCoinID(), validatorReward, validatorReward, candidate.RewardAddress)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), validatorReward)

		remainder.Sub(remainder, validatorReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleValidator.String(),
			Address:         candidate.RewardAddress,
			Amount:          validatorReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		stakes := v.bus.Candidates().GetStakes(validator.PubKey)
		for _, stake := range stakes {
			if stake.BipValue.Sign() == 0 {
				continue
			}

			reward := big.NewInt(0).Set(totalReward)
			reward.Mul(reward, stake.BipValue)

			reward.Div(reward, validator.GetTotalBipStake())

			remainder.Sub(remainder, reward)

			safeRewardVariable := big.NewInt(0).Set(reward)
			if validator.bus.Accounts().IsX3Mining(stake.Owner, height) {
				safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
				safeRewards.Mul(safeRewards, stake.BipValue)
				safeRewards.Div(safeRewards, validator.GetTotalBipStake())
				safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission+dao.Commission))), big.NewInt(100)))
				safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))
				safeRewards.Mul(safeRewards, big.NewInt(3))

				calcRewards := big.NewInt(0).Mul(calcReward, big.NewInt(period))
				calcRewards.Mul(calcRewards, stake.BipValue)
				calcRewards.Div(calcRewards, validator.GetTotalBipStake())
				calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(developers.Commission+dao.Commission))), big.NewInt(100)))
				calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))

				feeRewards := big.NewInt(0).Sub(reward, calcRewards)

				safeRewardVariable.Set(big.NewInt(0).Add(safeRewards, feeRewards))
				if safeRewardVariable.Sign() < 1 {
					continue
				}

				moreRewards.Add(moreRewards, new(big.Int).Sub(safeRewardVariable, reward))
			}

			if safeRewardVariable.Sign() < 1 {
				continue
			}

			candidate.AddUpdate(types.GetBaseCoinID(), safeRewardVariable, safeRewardVariable, stake.Owner)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), safeRewardVariable)

			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDelegator.String(),
				Address:         stake.Owner,
				Amount:          safeRewardVariable.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         uint64(stake.Coin),
			})
		}

		validator.SetAccumReward(big.NewInt(0))

		if remainder.Sign() != -1 {
			v.bus.App().AddTotalSlashed(remainder)
		} else {
			panic(fmt.Sprintf("Negative remainder: %s", remainder.String()))
		}
	}

	return moreRewards
}

// PayRewardsV5Fix2 distributes accumulated rewards between validator, delegators, DAO and developers addresses
func (v *Validators) PayRewardsV5Fix2(height uint64, period int64) (moreRewards *big.Int) {
	moreRewards = big.NewInt(0)

	vals := v.GetValidators()

	calcReward, safeReward := v.bus.App().Reward()
	var totalAccumRewards = big.NewInt(0)
	for _, validator := range vals {
		totalAccumRewards = totalAccumRewards.Add(totalAccumRewards, validator.GetAccumReward())
	}

	var totalStakes = big.NewInt(0)
	if totalAccumRewards.Sign() != 1 {
		for _, validator := range vals {
			totalStakes = totalStakes.Add(totalStakes, validator.GetTotalBipStake())
		}
	}

	for _, validator := range vals {
		candidate := v.bus.Candidates().GetCandidate(validator.PubKey)

		totalReward := big.NewInt(0).Set(validator.GetAccumReward())
		remainder := big.NewInt(0).Set(validator.GetAccumReward())

		// pay commission to DAO

		DAOReward := big.NewInt(0).Set(totalReward)
		DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
		DAOReward.Div(DAOReward, big.NewInt(100))

		// pay commission to Developers

		DevelopersReward := big.NewInt(0).Set(totalReward)
		DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
		DevelopersReward.Div(DevelopersReward, big.NewInt(100))

		totalReward.Sub(totalReward, DevelopersReward)
		totalReward.Sub(totalReward, DAOReward)
		remainder.Sub(remainder, DAOReward)
		remainder.Sub(remainder, DevelopersReward)

		// pay commission to validator
		validatorReward := big.NewInt(0).Set(totalReward)
		validatorReward.Mul(validatorReward, big.NewInt(int64(candidate.Commission)))
		validatorReward.Div(validatorReward, big.NewInt(100))
		totalReward.Sub(totalReward, validatorReward)

		candidate.AddUpdate(types.GetBaseCoinID(), validatorReward, validatorReward, candidate.RewardAddress)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), validatorReward)

		remainder.Sub(remainder, validatorReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleValidator.String(),
			Address:         candidate.RewardAddress,
			Amount:          validatorReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		stakes := v.bus.Candidates().GetStakes(validator.PubKey)
		for _, stake := range stakes {
			if stake.BipValue.Sign() == 0 {
				continue
			}

			reward := big.NewInt(0).Set(totalReward)
			reward.Mul(reward, stake.BipValue)

			reward.Div(reward, validator.GetTotalBipStake())

			remainder.Sub(remainder, reward)

			safeRewardVariable := big.NewInt(0).Set(reward)
			if validator.bus.Accounts().IsX3Mining(stake.Owner, height) {
				if totalAccumRewards.Sign() == 1 && validator.GetAccumReward().Sign() == 1 {
					safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
					safeRewards.Mul(safeRewards, stake.BipValue)
					safeRewards.Mul(safeRewards, big.NewInt(3))
					safeRewards.Mul(safeRewards, validator.GetAccumReward())
					safeRewards.Div(safeRewards, validator.GetTotalBipStake())
					safeRewards.Div(safeRewards, totalAccumRewards)

					taxDAOx3 := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEVx3 := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					safeRewards.Sub(safeRewards, taxDAOx3)
					safeRewards.Sub(safeRewards, taxDEVx3)
					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))

					calcRewards := big.NewInt(0).Mul(calcReward, big.NewInt(period))
					calcRewards.Mul(calcRewards, stake.BipValue)
					calcRewards.Mul(calcRewards, validator.GetAccumReward())
					calcRewards.Div(calcRewards, validator.GetTotalBipStake())
					calcRewards.Div(calcRewards, totalAccumRewards)

					taxDAO := big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEV := big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					calcRewards.Sub(calcRewards, taxDAO)
					calcRewards.Sub(calcRewards, taxDEV)
					calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))

					diffDAO := big.NewInt(0).Sub(taxDAOx3, taxDAO)
					diffDEV := big.NewInt(0).Sub(taxDAOx3, taxDEV)
					DAOReward.Add(DAOReward, diffDAO)
					DevelopersReward.Add(DevelopersReward, diffDEV)

					moreRewards.Add(moreRewards, diffDAO)
					moreRewards.Add(moreRewards, diffDEV)

					feeRewards := big.NewInt(0).Sub(reward, calcRewards)
					safeRewardVariable.Set(big.NewInt(0).Add(safeRewards, feeRewards))
				} else if totalAccumRewards.Sign() != 1 && validator.GetAccumReward().Sign() != 1 {
					safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
					safeRewards.Mul(safeRewards, stake.BipValue)
					safeRewards.Mul(safeRewards, big.NewInt(3))
					safeRewards.Div(safeRewards, totalStakes)

					taxDAO := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEV := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					DAOReward.Add(DAOReward, taxDAO)
					DevelopersReward.Add(DevelopersReward, taxDEV)
					moreRewards.Add(moreRewards, taxDAO)
					moreRewards.Add(moreRewards, taxDEV)

					safeRewards.Sub(safeRewards, taxDAO)
					safeRewards.Sub(safeRewards, taxDEV)

					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))

					safeRewardVariable.Set(safeRewards)
				}

				if safeRewardVariable.Sign() < 1 {
					continue
				}

				moreRewards.Add(moreRewards, new(big.Int).Sub(safeRewardVariable, reward))
			}

			if safeRewardVariable.Sign() < 1 {
				continue
			}

			candidate.AddUpdate(types.GetBaseCoinID(), safeRewardVariable, safeRewardVariable, stake.Owner)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), safeRewardVariable)

			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDelegator.String(),
				Address:         stake.Owner,
				Amount:          safeRewardVariable.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         uint64(stake.Coin),
			})
		}

		{
			candidate.AddUpdate(types.GetBaseCoinID(), DAOReward, DAOReward, dao.Address)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), DAOReward)
			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDAO.String(),
				Address:         dao.Address,
				Amount:          DAOReward.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         0,
			})
		}

		{
			candidate.AddUpdate(types.GetBaseCoinID(), DevelopersReward, DevelopersReward, developers.Address)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), DevelopersReward)
			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDevelopers.String(),
				Address:         developers.Address,
				Amount:          DevelopersReward.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         0,
			})
		}

		validator.SetAccumReward(big.NewInt(0))

		if remainder.Sign() != -1 {
			v.bus.App().AddTotalSlashed(remainder)
		} else {
			panic(fmt.Sprintf("Negative remainder: %s", remainder.String()))
		}
	}

	return moreRewards
}

// PayRewardsV5Fix
// Deprecated
func (v *Validators) PayRewardsV5Fix(height uint64, period int64) (moreRewards *big.Int) {
	moreRewards = big.NewInt(0)

	vals := v.GetValidators()

	calcReward, safeReward := v.bus.App().Reward()
	var totalAccumRewards = big.NewInt(0)
	for _, validator := range vals {
		totalAccumRewards = totalAccumRewards.Add(totalAccumRewards, validator.GetAccumReward())
	}

	var totalStakes = big.NewInt(0)
	if totalAccumRewards.Sign() != 1 {
		for _, validator := range vals {
			totalStakes = totalStakes.Add(totalStakes, validator.GetTotalBipStake())
		}
	}

	for _, validator := range vals {
		candidate := v.bus.Candidates().GetCandidate(validator.PubKey)

		totalReward := big.NewInt(0).Set(validator.GetAccumReward())
		remainder := big.NewInt(0).Set(validator.GetAccumReward())

		// pay commission to DAO

		DAOReward := big.NewInt(0).Set(totalReward)
		DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
		DAOReward.Div(DAOReward, big.NewInt(100))

		// pay commission to Developers

		DevelopersReward := big.NewInt(0).Set(totalReward)
		DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
		DevelopersReward.Div(DevelopersReward, big.NewInt(100))

		totalReward.Sub(totalReward, DevelopersReward)
		totalReward.Sub(totalReward, DAOReward)
		remainder.Sub(remainder, DAOReward)
		remainder.Sub(remainder, DevelopersReward)

		// pay commission to validator
		validatorReward := big.NewInt(0).Set(totalReward)
		validatorReward.Mul(validatorReward, big.NewInt(int64(candidate.Commission)))
		validatorReward.Div(validatorReward, big.NewInt(100))
		totalReward.Sub(totalReward, validatorReward)

		candidate.AddUpdate(types.GetBaseCoinID(), validatorReward, validatorReward, candidate.RewardAddress)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), validatorReward)

		remainder.Sub(remainder, validatorReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleValidator.String(),
			Address:         candidate.RewardAddress,
			Amount:          validatorReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		stakes := v.bus.Candidates().GetStakes(validator.PubKey)
		for _, stake := range stakes {
			if stake.BipValue.Sign() == 0 {
				continue
			}

			reward := big.NewInt(0).Set(totalReward)
			reward.Mul(reward, stake.BipValue)

			reward.Div(reward, validator.GetTotalBipStake())

			remainder.Sub(remainder, reward)

			safeRewardVariable := big.NewInt(0).Set(reward)
			if validator.bus.Accounts().IsX3Mining(stake.Owner, height) {
				if totalAccumRewards.Sign() == 1 && validator.GetAccumReward().Sign() == 1 {
					safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
					safeRewards.Mul(safeRewards, stake.BipValue)
					safeRewards.Mul(safeRewards, big.NewInt(3))
					safeRewards.Mul(safeRewards, validator.GetAccumReward())
					safeRewards.Div(safeRewards, validator.GetTotalBipStake())
					safeRewards.Div(safeRewards, totalAccumRewards)

					taxDAOx3 := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEVx3 := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					safeRewards.Sub(safeRewards, taxDAOx3)
					safeRewards.Sub(safeRewards, taxDEVx3)
					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))

					calcRewards := big.NewInt(0).Mul(calcReward, big.NewInt(period))
					calcRewards.Mul(calcRewards, stake.BipValue)
					calcRewards.Mul(calcRewards, validator.GetAccumReward())
					calcRewards.Div(calcRewards, validator.GetTotalBipStake())
					calcRewards.Div(calcRewards, totalAccumRewards)

					taxDAO := big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEV := big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					calcRewards.Sub(calcRewards, taxDAO)
					calcRewards.Sub(calcRewards, taxDEV)

					{
						// backward compatibility
						calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(developers.Commission+dao.Commission))), big.NewInt(100)))
					}

					calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))

					diffDAO := big.NewInt(0).Sub(taxDAOx3, taxDAO)
					diffDEV := big.NewInt(0).Sub(taxDAOx3, taxDEV)
					DAOReward.Add(DAOReward, diffDAO)
					DevelopersReward.Add(DevelopersReward, diffDEV)

					moreRewards.Add(moreRewards, diffDAO)
					moreRewards.Add(moreRewards, diffDEV)

					feeRewards := big.NewInt(0).Sub(reward, calcRewards)
					safeRewardVariable.Set(big.NewInt(0).Add(safeRewards, feeRewards))
				} else if totalAccumRewards.Sign() != 1 && validator.GetAccumReward().Sign() != 1 {
					safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
					safeRewards.Mul(safeRewards, stake.BipValue)
					safeRewards.Mul(safeRewards, big.NewInt(3))
					safeRewards.Div(safeRewards, totalStakes)

					taxDAO := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEV := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					DAOReward.Add(DAOReward, taxDAO)
					DevelopersReward.Add(DevelopersReward, taxDEV)
					moreRewards.Add(moreRewards, taxDAO)
					moreRewards.Add(moreRewards, taxDEV)

					safeRewards.Sub(safeRewards, taxDAO)
					safeRewards.Sub(safeRewards, taxDEV)

					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))

					safeRewardVariable.Set(safeRewards)
				}

				if safeRewardVariable.Sign() < 1 {
					continue
				}

				moreRewards.Add(moreRewards, new(big.Int).Sub(safeRewardVariable, reward))
			}

			if safeRewardVariable.Sign() < 1 {
				continue
			}

			candidate.AddUpdate(types.GetBaseCoinID(), safeRewardVariable, safeRewardVariable, stake.Owner)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), safeRewardVariable)

			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDelegator.String(),
				Address:         stake.Owner,
				Amount:          safeRewardVariable.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         uint64(stake.Coin),
			})
		}

		{
			candidate.AddUpdate(types.GetBaseCoinID(), DAOReward, DAOReward, dao.Address)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), DAOReward)
			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDAO.String(),
				Address:         dao.Address,
				Amount:          DAOReward.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         0,
			})
		}

		{
			candidate.AddUpdate(types.GetBaseCoinID(), DevelopersReward, DevelopersReward, developers.Address)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), DevelopersReward)
			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDevelopers.String(),
				Address:         developers.Address,
				Amount:          DevelopersReward.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         0,
			})
		}

		validator.SetAccumReward(big.NewInt(0))

		if remainder.Sign() != -1 {
			v.bus.App().AddTotalSlashed(remainder)
		} else {
			panic(fmt.Sprintf("Negative remainder: %s", remainder.String()))
		}
	}

	return moreRewards
}

// PayRewardsV5Bug
// Deprecated
func (v *Validators) PayRewardsV5Bug(height uint64, period int64) (moreRewards *big.Int) {
	moreRewards = big.NewInt(0)

	vals := v.GetValidators()

	calcReward, safeReward := v.bus.App().Reward()
	var totalAccumRewards = big.NewInt(0)
	for _, validator := range vals {
		totalAccumRewards = totalAccumRewards.Add(totalAccumRewards, validator.GetAccumReward())
	}

	var totalStakes = big.NewInt(0)
	if totalAccumRewards.Sign() != 1 {
		for _, validator := range vals {
			totalStakes = totalStakes.Add(totalStakes, validator.GetTotalBipStake())
		}
	}

	for _, validator := range vals {
		candidate := v.bus.Candidates().GetCandidate(validator.PubKey)

		totalReward := big.NewInt(0).Set(validator.GetAccumReward())
		remainder := big.NewInt(0).Set(validator.GetAccumReward())

		// pay commission to DAO

		DAOReward := big.NewInt(0).Set(totalReward)
		DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
		DAOReward.Div(DAOReward, big.NewInt(100))

		// pay commission to Developers

		DevelopersReward := big.NewInt(0).Set(totalReward)
		DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
		DevelopersReward.Div(DevelopersReward, big.NewInt(100))

		totalReward.Sub(totalReward, DevelopersReward)
		totalReward.Sub(totalReward, DAOReward)
		remainder.Sub(remainder, DAOReward)
		remainder.Sub(remainder, DevelopersReward)

		// pay commission to validator
		validatorReward := big.NewInt(0).Set(totalReward)
		validatorReward.Mul(validatorReward, big.NewInt(int64(candidate.Commission)))
		validatorReward.Div(validatorReward, big.NewInt(100))
		totalReward.Sub(totalReward, validatorReward)

		candidate.AddUpdate(types.GetBaseCoinID(), validatorReward, validatorReward, candidate.RewardAddress)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), validatorReward)

		remainder.Sub(remainder, validatorReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleValidator.String(),
			Address:         candidate.RewardAddress,
			Amount:          validatorReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		stakes := v.bus.Candidates().GetStakes(validator.PubKey)
		for _, stake := range stakes {
			if stake.BipValue.Sign() == 0 {
				continue
			}

			reward := big.NewInt(0).Set(totalReward)
			reward.Mul(reward, stake.BipValue)

			reward.Div(reward, validator.GetTotalBipStake())

			remainder.Sub(remainder, reward)

			safeRewardVariable := big.NewInt(0).Set(reward)
			if validator.bus.Accounts().IsX3Mining(stake.Owner, height) {
				if totalAccumRewards.Sign() == 1 && validator.GetAccumReward().Sign() == 1 {
					safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
					safeRewards.Mul(safeRewards, stake.BipValue)
					safeRewards.Mul(safeRewards, big.NewInt(3))
					safeRewards.Mul(safeRewards, validator.GetAccumReward())
					safeRewards.Div(safeRewards, validator.GetTotalBipStake())

					taxDAOx3 := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEVx3 := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					safeRewards.Sub(safeRewards, taxDAOx3)
					safeRewards.Sub(safeRewards, taxDEVx3)
					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))
					safeRewards.Div(safeRewards, totalAccumRewards)

					calcRewards := big.NewInt(0).Mul(calcReward, big.NewInt(period))
					calcRewards.Mul(calcRewards, stake.BipValue)
					calcRewards.Mul(calcRewards, validator.GetAccumReward())
					calcRewards.Div(calcRewards, validator.GetTotalBipStake())

					taxDAO := big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEV := big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					calcRewards.Sub(calcRewards, taxDAO)
					calcRewards.Sub(calcRewards, taxDEV)
					calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(developers.Commission+dao.Commission))), big.NewInt(100)))
					calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))
					calcRewards.Div(calcRewards, totalAccumRewards)

					diffDAO := big.NewInt(0).Sub(taxDAOx3, taxDAO)
					diffDEV := big.NewInt(0).Sub(taxDAOx3, taxDEV)
					DAOReward.Add(DAOReward, diffDAO)
					DevelopersReward.Add(DevelopersReward, diffDEV)

					moreRewards.Add(moreRewards, diffDAO)
					moreRewards.Add(moreRewards, diffDEV)

					feeRewards := big.NewInt(0).Sub(reward, calcRewards)
					safeRewardVariable.Set(big.NewInt(0).Add(safeRewards, feeRewards))
				} else if totalAccumRewards.Sign() != 1 && validator.GetAccumReward().Sign() != 1 {
					safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
					safeRewards.Mul(safeRewards, stake.BipValue)
					safeRewards.Mul(safeRewards, big.NewInt(3))

					taxDAO := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission))), big.NewInt(100))
					taxDEV := big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(dao.Commission))), big.NewInt(100))

					DAOReward.Add(DAOReward, taxDAO)
					DevelopersReward.Add(DevelopersReward, taxDEV)
					moreRewards.Add(moreRewards, taxDAO)
					moreRewards.Add(moreRewards, taxDEV)

					safeRewards.Sub(safeRewards, taxDAO)
					safeRewards.Sub(safeRewards, taxDEV)

					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))
					safeRewards.Div(safeRewards, totalStakes)

					safeRewardVariable.Set(safeRewards)
				}

				if safeRewardVariable.Sign() < 1 {
					continue
				}

				moreRewards.Add(moreRewards, new(big.Int).Sub(safeRewardVariable, reward))
			}

			if safeRewardVariable.Sign() < 1 {
				continue
			}

			candidate.AddUpdate(types.GetBaseCoinID(), safeRewardVariable, safeRewardVariable, stake.Owner)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), safeRewardVariable)

			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDelegator.String(),
				Address:         stake.Owner,
				Amount:          safeRewardVariable.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         uint64(stake.Coin),
			})
		}

		{
			candidate.AddUpdate(types.GetBaseCoinID(), DAOReward, DAOReward, dao.Address)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), DAOReward)
			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDAO.String(),
				Address:         dao.Address,
				Amount:          DAOReward.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         0,
			})
		}

		{
			candidate.AddUpdate(types.GetBaseCoinID(), DevelopersReward, DevelopersReward, developers.Address)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), DevelopersReward)
			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDevelopers.String(),
				Address:         developers.Address,
				Amount:          DevelopersReward.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         0,
			})
		}

		validator.SetAccumReward(big.NewInt(0))

		if remainder.Sign() != -1 {
			v.bus.App().AddTotalSlashed(remainder)
		} else {
			panic(fmt.Sprintf("Negative remainder: %s", remainder.String()))
		}
	}

	return moreRewards
}

// PayRewardsV4
// Deprecated
func (v *Validators) PayRewardsV4(height uint64, period int64) (moreRewards *big.Int) {
	moreRewards = big.NewInt(0)

	vals := v.GetValidators()

	calcReward, safeReward := v.bus.App().Reward()
	var totalAccumRewards = big.NewInt(0)
	for _, validator := range vals {
		totalAccumRewards = totalAccumRewards.Add(totalAccumRewards, validator.GetAccumReward())
	}

	var totalStakes = big.NewInt(0)
	if totalAccumRewards.Sign() != 1 {
		for _, validator := range vals {
			totalStakes = totalStakes.Add(totalStakes, validator.GetTotalBipStake())
		}
	}

	for _, validator := range vals {
		candidate := v.bus.Candidates().GetCandidate(validator.PubKey)

		totalReward := big.NewInt(0).Set(validator.GetAccumReward())
		remainder := big.NewInt(0).Set(validator.GetAccumReward())

		// pay commission to DAO
		DAOReward := big.NewInt(0).Set(totalReward)
		DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
		DAOReward.Div(DAOReward, big.NewInt(100))

		candidate.AddUpdate(types.GetBaseCoinID(), DAOReward, DAOReward, dao.Address)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), DAOReward)

		remainder.Sub(remainder, DAOReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleDAO.String(),
			Address:         dao.Address,
			Amount:          DAOReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		// pay commission to Developers
		DevelopersReward := big.NewInt(0).Set(totalReward)
		DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
		DevelopersReward.Div(DevelopersReward, big.NewInt(100))

		candidate.AddUpdate(types.GetBaseCoinID(), DevelopersReward, DevelopersReward, developers.Address)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), DevelopersReward)

		remainder.Sub(remainder, DevelopersReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleDevelopers.String(),
			Address:         developers.Address,
			Amount:          DevelopersReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		totalReward.Sub(totalReward, DevelopersReward)
		totalReward.Sub(totalReward, DAOReward)

		// pay commission to validator
		validatorReward := big.NewInt(0).Set(totalReward)
		validatorReward.Mul(validatorReward, big.NewInt(int64(candidate.Commission)))
		validatorReward.Div(validatorReward, big.NewInt(100))
		totalReward.Sub(totalReward, validatorReward)

		candidate.AddUpdate(types.GetBaseCoinID(), validatorReward, validatorReward, candidate.RewardAddress)
		v.bus.Checker().AddCoin(types.GetBaseCoinID(), validatorReward)

		remainder.Sub(remainder, validatorReward)
		v.bus.Events().AddEvent(&eventsdb.RewardEvent{
			Role:            eventsdb.RoleValidator.String(),
			Address:         candidate.RewardAddress,
			Amount:          validatorReward.String(),
			ValidatorPubKey: validator.PubKey,
			ForCoin:         0,
		})

		stakes := v.bus.Candidates().GetStakes(validator.PubKey)
		for _, stake := range stakes {
			if stake.BipValue.Sign() == 0 {
				continue
			}

			reward := big.NewInt(0).Set(totalReward)
			reward.Mul(reward, stake.BipValue)

			reward.Div(reward, validator.GetTotalBipStake())

			remainder.Sub(remainder, reward)

			safeRewardVariable := big.NewInt(0).Set(reward)
			if validator.bus.Accounts().IsX3Mining(stake.Owner, height) {
				if totalAccumRewards.Sign() == 1 && validator.GetAccumReward().Sign() == 1 {
					safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
					safeRewards.Mul(safeRewards, stake.BipValue)
					safeRewards.Mul(safeRewards, big.NewInt(3))
					safeRewards.Mul(safeRewards, validator.GetAccumReward())
					safeRewards.Div(safeRewards, validator.GetTotalBipStake())
					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission+dao.Commission))), big.NewInt(100)))
					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))
					safeRewards.Div(safeRewards, totalAccumRewards)

					calcRewards := big.NewInt(0).Mul(calcReward, big.NewInt(period))
					calcRewards.Mul(calcRewards, stake.BipValue)
					calcRewards.Mul(calcRewards, validator.GetAccumReward())
					calcRewards.Div(calcRewards, validator.GetTotalBipStake())
					calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(developers.Commission+dao.Commission))), big.NewInt(100)))
					calcRewards.Sub(calcRewards, big.NewInt(0).Div(big.NewInt(0).Mul(calcRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))
					calcRewards.Div(calcRewards, totalAccumRewards)

					feeRewards := big.NewInt(0).Sub(reward, calcRewards)
					safeRewardVariable.Set(big.NewInt(0).Add(safeRewards, feeRewards))
				} else if totalAccumRewards.Sign() != 1 && validator.GetAccumReward().Sign() != 1 {
					safeRewards := big.NewInt(0).Mul(safeReward, big.NewInt(period))
					safeRewards.Mul(safeRewards, stake.BipValue)
					safeRewards.Mul(safeRewards, big.NewInt(3))
					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(developers.Commission+dao.Commission))), big.NewInt(100)))
					safeRewards.Sub(safeRewards, big.NewInt(0).Div(big.NewInt(0).Mul(safeRewards, big.NewInt(int64(candidate.Commission))), big.NewInt(100)))
					safeRewards.Div(safeRewards, totalStakes)

					safeRewardVariable.Set(safeRewards)
				}

				if safeRewardVariable.Sign() < 1 {
					continue
				}

				moreRewards.Add(moreRewards, new(big.Int).Sub(safeRewardVariable, reward))
			}

			if safeRewardVariable.Sign() < 1 {
				continue
			}

			candidate.AddUpdate(types.GetBaseCoinID(), safeRewardVariable, safeRewardVariable, stake.Owner)
			v.bus.Checker().AddCoin(types.GetBaseCoinID(), safeRewardVariable)

			v.bus.Events().AddEvent(&eventsdb.RewardEvent{
				Role:            eventsdb.RoleDelegator.String(),
				Address:         stake.Owner,
				Amount:          safeRewardVariable.String(),
				ValidatorPubKey: validator.PubKey,
				ForCoin:         uint64(stake.Coin),
			})
		}

		validator.SetAccumReward(big.NewInt(0))

		if remainder.Sign() != -1 {
			v.bus.App().AddTotalSlashed(remainder)
		} else {
			panic(fmt.Sprintf("Negative remainder: %s", remainder.String()))
		}
	}

	return moreRewards
}

// GetByTmAddress finds and returns validator with given tendermint-address
func (v *Validators) GetByTmAddress(address types.TmAddress) *Validator {
	v.lock.RLock()
	defer v.lock.RUnlock()

	for _, val := range v.list {
		val.lock.RLock()
		if val.tmAddress == address {
			val.lock.RUnlock()
			return val
		}
		val.lock.RUnlock()
	}

	return nil
}

// GetByPublicKey finds and returns validator
func (v *Validators) GetByPublicKey(pubKey types.Pubkey) *Validator {
	v.lock.RLock()
	defer v.lock.RUnlock()

	for _, val := range v.list {
		val.lock.RLock()
		if val.PubKey == pubKey {
			val.lock.RUnlock()
			return val
		}
		val.lock.RUnlock()
	}

	return nil
}

// LoadValidators loads only list of validators (for read)
func (v *Validators) LoadValidators() {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.loaded {
		return
	}

	v.loaded = true

	path := []byte{mainPrefix}
	_, enc := v.immutableTree().Get(path)
	if len(enc) == 0 {
		v.list = nil
		return
	}

	var validators []*Validator
	if err := rlp.DecodeBytes(enc, &validators); err != nil {
		panic(fmt.Sprintf("failed to decode validators: %s", err))
	}

	v.list = validators
	for _, validator := range validators {
		// load total stake
		path = append([]byte{mainPrefix}, validator.PubKey.Bytes()...)
		path = append(path, totalStakePrefix)
		_, enc = v.immutableTree().Get(path)
		if len(enc) == 0 {
			validator.totalStake = big.NewInt(0)
		} else {
			validator.totalStake = big.NewInt(0).SetBytes(enc)
		}

		// load accum reward
		path = append([]byte{mainPrefix}, validator.PubKey.Bytes()...)
		path = append(path, accumRewardPrefix)
		_, enc = v.immutableTree().Get(path)
		if len(enc) == 0 {
			validator.accumReward = big.NewInt(0)
		} else {
			validator.accumReward = big.NewInt(0).SetBytes(enc)
		}

		validator.setTmAddress()
		validator.bus = v.bus
	}
}

func (v *Validators) hasDirtyValidators() bool {
	v.lock.RLock()
	defer v.lock.RUnlock()

	for _, val := range v.list {
		if val.isDirty {
			return true
		}
	}

	return false
}

func (v *Validators) uncheckDirtyValidators() {
	for _, val := range v.list {
		val.lock.Lock()
		val.isDirty = false
		val.isAccumRewardDirty = false
		val.isTotalStakeDirty = false
		val.lock.Unlock()
	}
}

func (v *Validators) punishValidator(height uint64, tmAddress types.TmAddress) {
	v.bus.Candidates().Punish(height, tmAddress)
}

// SetValidators updates validators list
func (v *Validators) SetValidators(vals []*Validator) {
	v.lock.Lock()
	v.list = vals
	v.lock.Unlock()
}

func (v *Validators) IsValidator(pubkey types.Pubkey) bool {
	v.lock.RLock()
	defer v.lock.RUnlock()
	for _, val := range v.GetValidators() {
		if val.PubKey == pubkey {
			return true
		}
	}
	return false
}

// Export exports all data to the given state
func (v *Validators) Export(state *types.AppState) {
	v.LoadValidators()

	for _, val := range v.GetValidators() {
		state.Validators = append(state.Validators, types.Validator{
			TotalBipStake: val.GetTotalBipStake().String(),
			PubKey:        val.PubKey,
			AccumReward:   val.GetAccumReward().String(),
			AbsentTimes:   val.AbsentTimes,
		})
	}
}

// SetToDrop marks given validator as inactive for dropping it in the next block
func (v *Validators) SetToDrop(pubkey types.Pubkey) {
	vals := v.GetValidators()
	for _, val := range vals {
		val.lock.Lock()
		if val.PubKey == pubkey {
			val.toDrop = true
		}
		val.lock.Unlock()
	}
}

func (v *Validators) turnValidatorOff(tmAddress types.TmAddress) {
	validator := v.GetByTmAddress(tmAddress)

	validator.lock.Lock()
	defer validator.lock.Unlock()

	validator.AbsentTimes = types.NewBitArray(ValidatorMaxAbsentWindow)
	validator.toDrop = true
	validator.isDirty = true
	v.bus.Candidates().SetOffline(validator.PubKey)
}
