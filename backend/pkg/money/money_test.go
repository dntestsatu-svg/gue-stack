package money

import "testing"

import "github.com/stretchr/testify/require"

func TestPercentageFeeRoundsHalfUp(t *testing.T) {
	fee, err := PercentageFee(3333, 3)
	require.NoError(t, err)
	require.Equal(t, uint64(100), fee)
}

func TestPercentageFeeZeroCases(t *testing.T) {
	fee, err := PercentageFee(0, 3)
	require.NoError(t, err)
	require.Zero(t, fee)

	fee, err = PercentageFee(10000, 0)
	require.NoError(t, err)
	require.Zero(t, fee)
}

func TestAddUint64(t *testing.T) {
	sum, err := AddUint64(100000, 1500)
	require.NoError(t, err)
	require.Equal(t, uint64(101500), sum)
}
