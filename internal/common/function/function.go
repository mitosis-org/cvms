package function

import (
	"encoding/base64"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/api"
	indexermodel "github.com/cosmostation/cvms/internal/common/indexer/model"
	"github.com/cosmostation/cvms/internal/common/types"

	sdkhelper "github.com/cosmostation/cvms/internal/helper/sdk"
	"github.com/pkg/errors"
)

func MakeValidatorInfoList(
	app common.CommonApp,
	chainID string, chainInfoID int64,
	chainName string, isConsumer bool,
	newValidatorAddressMap map[string]bool,
	height ...int64,
) ([]indexermodel.ValidatorInfo, error) {
	switch true {
	case chainName == "bera":
		// berachain doesn't have staking module, so we need to make custom validator info list for bera
		// stakingValidators, err := api.GetValidators(commonClient, chainName)
		newStakingValidatorMap := make(map[string]types.StakingValidatorMetaInfo)
		validators, err := api.GetValidators(app.CommonClient)
		if err != nil {
			return nil, errors.Cause(err)
		}
		// TODO: change logic with staking module contract on bera-geth
		for _, validator := range validators {
			hexAddress := validator.Address
			blsPubkey, _ := sdkhelper.MakeBLSPubkey(validator.Pubkey.Value)
			newStakingValidatorMap[hexAddress] = types.StakingValidatorMetaInfo{
				Moniker:         blsPubkey,
				OperatorAddress: blsPubkey,
			}
		}

		newValidatorInfoList := make([]indexermodel.ValidatorInfo, 0)
		for newHexAddress := range newValidatorAddressMap {
			if _, exist := newStakingValidatorMap[newHexAddress]; !exist {
				return nil, errors.New("unexpected error while collecting validator info data")
			}
			newValidatorInfoList = append(
				newValidatorInfoList,
				indexermodel.ValidatorInfo{
					ChainInfoID:     chainInfoID,
					HexAddress:      newHexAddress,
					OperatorAddress: newStakingValidatorMap[newHexAddress].OperatorAddress,
					Moniker:         newStakingValidatorMap[newHexAddress].Moniker,
				})
		}
		return newValidatorInfoList, nil

	case chainName == "mitosis":
		// mitosis chain uses evmvalidator module instead of staking module
		newStakingValidatorMap := make(map[string]types.StakingValidatorMetaInfo)
		mitosisValidators, err := api.GetMitosisValidators(app.CommonClient)
		if err != nil {
			return nil, errors.Cause(err)
		}

		// get tendermint validators to map consensus addresses
		validators, err := api.GetValidators(app.CommonClient)
		if err != nil {
			return nil, errors.Cause(err)
		}

		// create map of pubkey to hex address from tendermint validators
		pubkeyToHexAddr := make(map[string]string)
		for _, validator := range validators {
			pubkeyToHexAddr[validator.Pubkey.Value] = validator.Address
		}

		for _, mitosisValidator := range mitosisValidators {
			// find the corresponding tendermint validator by matching pubkey
			hexAddress, found := pubkeyToHexAddr[mitosisValidator.Pubkey]
			if !found {
				app.Warnf("mitosis validator %s pubkey not found in tendermint validators", mitosisValidator.Addr)
				continue
			}

			// use ethereum address as moniker and operator address
			newStakingValidatorMap[hexAddress] = types.StakingValidatorMetaInfo{
				Moniker:         mitosisValidator.Addr, // Use Ethereum address as moniker
				OperatorAddress: mitosisValidator.Addr, // Use Ethereum address as operator address
			}
		}

		newValidatorInfoList := make([]indexermodel.ValidatorInfo, 0)
		for newHexAddress := range newValidatorAddressMap {
			if _, exist := newStakingValidatorMap[newHexAddress]; !exist {
				return nil, errors.Errorf("mitosis validator with hex address %s not found in evmvalidator module", newHexAddress)
			}
			newValidatorInfoList = append(
				newValidatorInfoList,
				indexermodel.ValidatorInfo{
					ChainInfoID:     chainInfoID,
					HexAddress:      newHexAddress,
					OperatorAddress: newStakingValidatorMap[newHexAddress].OperatorAddress,
					Moniker:         newStakingValidatorMap[newHexAddress].Moniker,
				})
		}
		return newValidatorInfoList, nil

	// only for babylon checkpoint indexer
	case chainName == "babylon" && len(height) > 0:
		if len(height) == 0 {
			return nil, errors.New("height was not passed")
		}

		// if there is a new validators, first make stakingvalidator map by hexAddress
		newStakingValidatorMap := make(map[string]types.StakingValidatorMetaInfo)
		stakingValidators, err := api.GetStakingValidatorsByHeight(app.CommonClient, chainName, height[0])
		if err != nil {
			return nil, errors.Cause(err)
		}
		for _, validator := range stakingValidators {
			decodedKey, _ := base64.StdEncoding.DecodeString(validator.ConsensusPubkey.Key)
			hexAddress, err := sdkhelper.MakeProposerAddress(validator.ConsensusPubkey.Type, decodedKey)
			if err != nil {
				return nil, errors.Cause(err)
			}
			newStakingValidatorMap[hexAddress] = types.StakingValidatorMetaInfo{
				Moniker:         validator.Description.Moniker,
				OperatorAddress: validator.OperatorAddress,
			}
		}
		newValidatorInfoList := make([]indexermodel.ValidatorInfo, 0)
		for newHexAddress := range newValidatorAddressMap {
			if _, exist := newStakingValidatorMap[newHexAddress]; !exist {
				return nil, errors.New("unexpected error while collecting validator info data")
			}
			newValidatorInfoList = append(
				newValidatorInfoList,
				indexermodel.ValidatorInfo{
					ChainInfoID:     chainInfoID,
					HexAddress:      newHexAddress,
					OperatorAddress: newStakingValidatorMap[newHexAddress].OperatorAddress,
					Moniker:         newStakingValidatorMap[newHexAddress].Moniker,
				})
		}
		return newValidatorInfoList, nil
	case isConsumer:
		providerClient := app.OptionalClient

		var consumerID string
		consumerChains, err := api.GetConsumerChainID(providerClient)
		if err != nil {
			return nil, errors.Cause(err)
		}
		for _, consumerChain := range consumerChains {
			if consumerChain.ChainID == chainID {
				consumerID = consumerChain.ConsumerID
				break
			}
		}
		if consumerID == "" {
			return nil, errors.Errorf("failed to find consumer id, check again your chain-id: %s", chainID)
		}

		newStakingValidatorMap := make(map[string]types.StakingValidatorMetaInfo)
		stakingProviderValidators, err := api.GetProviderValidators(providerClient, consumerID)
		if err != nil {
			return nil, errors.Cause(err)
		}
		for _, validator := range stakingProviderValidators {
			decodedKey, _ := base64.StdEncoding.DecodeString(validator.ConsumerKey.Pubkey)
			hexAddress, err := sdkhelper.MakeProposerAddress(sdkhelper.Ed25519, decodedKey)
			if err != nil {
				return nil, errors.Cause(err)
			}
			newStakingValidatorMap[hexAddress] = types.StakingValidatorMetaInfo{
				Moniker:         validator.Description.Moniker,
				OperatorAddress: validator.ProviderValoperAddress,
			}
		}
		retryCount := 0
		newValidatorInfoList := make([]indexermodel.ValidatorInfo, 0)
		for newHexAddress := range newValidatorAddressMap {
		consumerRetryLoop:
			if _, exist := newStakingValidatorMap[newHexAddress]; !exist {
				app.Warnln("some validators aren't in BOND_STATUS_BONDED, retry after getting validator info from not BOND_STATUS", "Retrying... Attempt", retryCount)
				retryCount++
				switch retryCount {
				case 1:
					GetStakingValidators(app.CommonClient, chainName, newStakingValidatorMap, types.Unbonding)
					goto consumerRetryLoop
				case 2:
					GetStakingValidators(app.CommonClient, chainName, newStakingValidatorMap, types.Unbonded)
					goto consumerRetryLoop
				case 3:
					GetStakingValidators(app.CommonClient, chainName, newStakingValidatorMap, types.Unspecfied)
					goto consumerRetryLoop
				default:
					return nil, errors.New("unexpected error while collecting validator info data")
				}
			}
			newValidatorInfoList = append(
				newValidatorInfoList,
				indexermodel.ValidatorInfo{
					ChainInfoID:     chainInfoID,
					HexAddress:      newHexAddress,
					OperatorAddress: newStakingValidatorMap[newHexAddress].OperatorAddress,
					Moniker:         newStakingValidatorMap[newHexAddress].Moniker,
				})
		}
		return newValidatorInfoList, nil
	default:
		// if there is a new validators, first make stakingvalidator map by hexAddress
		newStakingValidatorMap := make(map[string]types.StakingValidatorMetaInfo)
		stakingValidators, err := api.GetStakingValidators(app.CommonClient, chainName)
		if err != nil {
			return nil, errors.Cause(err)
		}
		for _, validator := range stakingValidators {
			decodedKey, _ := base64.StdEncoding.DecodeString(validator.ConsensusPubkey.Key)
			hexAddress, err := sdkhelper.MakeProposerAddress(validator.ConsensusPubkey.Type, decodedKey)
			if err != nil {
				return nil, errors.Cause(err)
			}
			newStakingValidatorMap[hexAddress] = types.StakingValidatorMetaInfo{
				Moniker:         validator.Description.Moniker,
				OperatorAddress: validator.OperatorAddress,
			}
		}
		retryCount := 0
		newValidatorInfoList := make([]indexermodel.ValidatorInfo, 0)
		for newHexAddress := range newValidatorAddressMap {
		retryLoop:
			if _, exist := newStakingValidatorMap[newHexAddress]; !exist {
				app.Warnln("some validators aren't in BOND_STATUS_BONDED, retry after getting validator info from not BOND_STATUS_BONDED", "Retrying... Attempt", retryCount)
				retryCount++
				switch retryCount {
				case 1:
					GetStakingValidators(app.CommonClient, chainName, newStakingValidatorMap, types.Unbonding)
					goto retryLoop
				case 2:
					GetStakingValidators(app.CommonClient, chainName, newStakingValidatorMap, types.Unbonded)
					goto retryLoop
				case 3:
					GetStakingValidators(app.CommonClient, chainName, newStakingValidatorMap, types.Unspecfied)
					goto retryLoop
				default:
					app.Errorf("%v", newStakingValidatorMap)
					return nil, errors.New("unexpected error while collecting validator info data")
				}
			}
			newValidatorInfoList = append(
				newValidatorInfoList,
				indexermodel.ValidatorInfo{
					ChainInfoID:     chainInfoID,
					HexAddress:      newHexAddress,
					OperatorAddress: newStakingValidatorMap[newHexAddress].OperatorAddress,
					Moniker:         newStakingValidatorMap[newHexAddress].Moniker,
				})
		}
		return newValidatorInfoList, nil
	}
}

func GetStakingValidators(client common.CommonClient, chainName string, newStakingValidatorMap map[string]types.StakingValidatorMetaInfo, status types.BondStatus) error {
	stakingValidators, err := api.GetStakingValidators(client, chainName, string(status))
	if err != nil {
		return errors.Cause(err)
	}
	for _, validator := range stakingValidators {
		decodedKey, _ := base64.StdEncoding.DecodeString(validator.ConsensusPubkey.Key)
		hexAddress, err := sdkhelper.MakeProposerAddress(validator.ConsensusPubkey.Type, decodedKey)
		if err != nil {
			return errors.Cause(err)
		}
		newStakingValidatorMap[hexAddress] = types.StakingValidatorMetaInfo{
			Moniker:         validator.Description.Moniker,
			OperatorAddress: validator.OperatorAddress,
		}
	}
	return nil
}

func MakeFinalityProviderInfoList(c common.CommonClient, chainID string, chainInfoID int64, newFinalityProviderMap map[string]bool) ([]indexermodel.FinalityProviderInfo, error) {
	fps, err := api.GetBabylonFinalityProviderInfos(c, chainID)
	if err != nil {
		return nil, errors.Cause(err)
	}

	fpInfoList := make([]indexermodel.FinalityProviderInfo, 0)
	for _, fp := range fps {
		// if found new btc pk, it need to add fp list for indexing
		if newFinalityProviderMap[fp.BTCPK] {
			fpInfoList = append(fpInfoList, indexermodel.FinalityProviderInfo{
				ChainInfoID:     chainInfoID,
				Moniker:         fp.Description.Moniker,
				BTCPKs:          fp.BTCPK,
				OperatorAddress: fp.Address,
			})
		}
	}

	if len(fpInfoList) == 0 {
		return nil, errors.New("unexpected err")
	}

	return fpInfoList, nil
}
