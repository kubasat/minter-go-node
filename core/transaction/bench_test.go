package transaction

import (
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"sync"
	"testing"
)

func BenchmarkSample(b *testing.B) {
	b.ReportAllocs()

	b.StopTimer()
	tx, _ := hex.DecodeString("f88301018a4d4e540000000000000001aae98a4d4e5400000000000000940100000000000000000000000000000000000000888ac7230489e80000808001b845f8431ca01720a1109e25cb13ffbad0ef9ffff6c8292f744cdd81769df1081d916aa654a9a027217d5eca257f5f4970e884a6d27bb94994c80765dc165513b4bc1d30d888f0")
	addr := types.HexToAddress("Mxb15d68fb6b4ca426e35bbba6418dfca4606edc6d")
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		send(tx, addr, b)
	}
}

func send(b []byte, addr types.Address, bt *testing.B) {
	bt.StopTimer()
	cState := getState()
	cState.AddBalance(addr, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(1000000)))
	tx, _ := DecodeFromBytes(b)
	bt.StartTimer()

	wg := sync.WaitGroup{}

	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			_, _ = tx.Sender()
			wg.Done()
		}()
	}

	wg.Wait()

	//response := RunTx(cState, false, b, big.NewInt(0), 0)
	//
	//if response.Code != 0 {
	//	panic("Response code is not 0")
	//}
}
