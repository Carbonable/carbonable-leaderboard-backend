package starknet_test

import (
	"testing"

	"github.com/carbonable/leaderboard/internal/starknet"
	"gotest.tools/assert"
)

func TestCallSlotUri(t *testing.T) {
	goerli := starknet.GoerliJsonRpcStarknetClient()

	slotUri, err := starknet.GetSlotUri(goerli, "0x04b9f63c40668305ff651677f97424921bcd1b781aafa66d1b4948a87f056d0d", uint64(1))
	if err != nil {
		t.Errorf("error while testing GetSlotUri : %s", err)
	}

	assert.Equal(t, slotUri.Name, "Banegas Farm")
}

func TestCallSlotOf(t *testing.T) {
	goerli := starknet.GoerliJsonRpcStarknetClient()

	slot, err := starknet.GetSlotOf(goerli, "0x04b9f63c40668305ff651677f97424921bcd1b781aafa66d1b4948a87f056d0d", uint64(1))
	if err != nil {
		t.Errorf("error while testing GetSlotOf : %s", err)
	}

	// Token ID 1 is in slot 1
	assert.Equal(t, slot, uint64(1))
}

func TestGetRemainingValue(t *testing.T) {
	mainnet := starknet.MainnetJsonRpcStarknetClient()

	rv, err := starknet.GetRemainingValue(mainnet, "0x07336c28e621dce9940603fb85136c57a3c46ce22e4ec862eeb0bdb0cd5cc9d9")
	if err != nil {
		t.Errorf("error while testing GetSlotOf : %s", err)
	}

	// Token ID 1 is in slot 1
	assert.NilError(t, err)
	assert.Equal(t, nil != rv, true)
}
