package transaction

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/danil-lashin/tendermint/libs/common"
	"math/big"
)

type CreateMultisigData struct {
	Threshold uint
	Weights   []uint
	Addresses []types.Address
}

func (data CreateMultisigData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Threshold uint            `json:"threshold"`
		Weights   []uint          `json:"weights"`
		Addresses []types.Address `json:"addresses"`
	}{
		Weights:   data.Weights,
		Threshold: data.Threshold,
		Addresses: data.Addresses,
	})
}

func (data CreateMultisigData) String() string {
	return fmt.Sprintf("CREATE MULTISIG")
}

func (data CreateMultisigData) Gas() int64 {
	return commissions.CreateMultisig
}

func (data CreateMultisigData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock int64) Response {
	if !context.CoinExists(tx.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if len(data.Weights) > 32 {
		return Response{
			Code: code.TooLargeOwnersList,
			Log:  fmt.Sprintf("Owners list is limited to 32 items")}
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := context.GetStateCoin(tx.GasCoin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)

		if commission == nil {
			return Response{
				Code: 999,
				Log:  "Unknown error"}
		}
	}

	if context.GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, tx.GasCoin)}
	}

	if len(data.Addresses) != len(data.Weights) {
		return Response{
			Code: code.IncorrectWeights,
			Log:  fmt.Sprintf("Incorrect multisig weights")}
	}

	msigAddress := (&state.Multisig{
		Weights:   data.Weights,
		Threshold: data.Threshold,
		Addresses: data.Addresses,
	}).Address()

	if context.AccountExists(msigAddress) {
		return Response{
			Code: code.MultisigExists,
			Log:  fmt.Sprintf("Multisig %s already exists", msigAddress.String())}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		if !tx.GasCoin.IsBaseCoin() {
			context.SubCoinVolume(tx.GasCoin, commission)
			context.SubCoinReserve(tx.GasCoin, commissionInBaseCoin)
		}

		context.SubBalance(sender, tx.GasCoin, commission)
		context.SetNonce(sender, tx.Nonce)

		context.CreateMultisig(data.Weights, data.Addresses, data.Threshold)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeCreateMultisig}},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.created_multisig"), Value: []byte(hex.EncodeToString(msigAddress[:]))},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
