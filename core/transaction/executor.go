package transaction

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/danil-lashin/tendermint/libs/common"
	"math/big"
)

var (
	CommissionMultiplier = big.NewInt(10e14)
)

const (
	MaxTxLength          = maxPayloadLength + maxServiceDataLength + 1024
	maxPayloadLength     = 1024
	maxServiceDataLength = 128
)

type Response struct {
	Code      uint32          `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Data      []byte          `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Log       string          `protobuf:"bytes,3,opt,name=log,proto3" json:"log,omitempty"`
	Info      string          `protobuf:"bytes,4,opt,name=info,proto3" json:"info,omitempty"`
	GasWanted int64           `protobuf:"varint,5,opt,name=gas_wanted,json=gasWanted,proto3" json:"gas_wanted,omitempty"`
	GasUsed   int64           `protobuf:"varint,6,opt,name=gas_used,json=gasUsed,proto3" json:"gas_used,omitempty"`
	Tags      []common.KVPair `protobuf:"bytes,7,rep,name=tags" json:"tags,omitempty"`
}

func RunTx(context *state.StateDB, isCheck bool, tx *Transaction, rewardPool *big.Int, currentBlock int64) Response {
	if !isCheck {
		log.Info("Deliver tx", "tx", tx.String())
	}

	if len(tx.Payload) > maxPayloadLength {
		return Response{
			Code: code.TxPayloadTooLarge,
			Log:  fmt.Sprintf("TX payload length is over %d bytes", maxPayloadLength)}
	}

	if len(tx.ServiceData) > maxServiceDataLength {
		return Response{
			Code: code.TxServiceDataTooLarge,
			Log:  fmt.Sprintf("TX service data length is over %d bytes", maxServiceDataLength)}
	}

	sender, err := tx.Sender()

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	// check multi-signature
	if tx.SignatureType == SigTypeMulti {
		multisig := context.GetOrNewStateObject(tx.multisig.Multisig)

		if !multisig.IsMultisig() {
			return Response{
				Code: code.MultisigNotExists,
				Log:  "Multisig does not exists"}
		}

		multisigData := multisig.Multisig()

		if len(tx.multisig.Signatures) > 32 || len(multisigData.Weights) < len(tx.multisig.Signatures) {
			return Response{
				Code: code.IncorrectMultiSignature,
				Log:  "Incorrect multi-signature"}
		}

		txHash := tx.Hash()
		var totalWeight uint = 0
		var usedAccounts = map[types.Address]bool{}

		for _, sig := range tx.multisig.Signatures {
			signer, err := RecoverPlain(txHash, sig.R, sig.S, sig.V)

			if err != nil {
				return Response{
					Code: code.IncorrectMultiSignature,
					Log:  "Incorrect multi-signature"}
			}

			if usedAccounts[signer] {
				return Response{
					Code: code.IncorrectMultiSignature,
					Log:  "Incorrect multi-signature"}
			}

			usedAccounts[signer] = true
			totalWeight += multisigData.GetWeight(signer)
		}

		if totalWeight < multisigData.Threshold {
			return Response{
				Code: code.IncorrectMultiSignature,
				Log:  fmt.Sprintf("Not enough multisig votes. Needed %d, has %d", multisigData.Threshold, totalWeight)}
		}
	}

	// TODO: deal with multiple pending transactions from one account
	if expectedNonce := context.GetNonce(sender) + 1; expectedNonce != tx.Nonce {
		return Response{
			Code: code.WrongNonce,
			Log:  fmt.Sprintf("Unexpected nonce. Expected: %d, got %d.", expectedNonce, tx.Nonce)}
	}

	return tx.decodedData.Run(sender, tx, context, isCheck, rewardPool, currentBlock)
}
