package money

import (
	"fmt"
	"math/big"
	"math/bits"

	"github.com/shopspring/decimal"
)

var hundred = decimal.NewFromInt(100)

// PercentageFee computes a whole-percent fee using half-up rounding to the nearest rupiah.
// This keeps the calculation deterministic and auditable for every transaction.
func PercentageFee(amount uint64, wholePercent uint64) (uint64, error) {
	if amount == 0 || wholePercent == 0 {
		return 0, nil
	}

	amountDec := decimal.NewFromBigInt(new(big.Int).SetUint64(amount), 0)
	percentDec := decimal.NewFromBigInt(new(big.Int).SetUint64(wholePercent), 0)
	fee := amountDec.Mul(percentDec).Div(hundred).Round(0)
	if fee.IsNegative() {
		return 0, fmt.Errorf("computed negative fee")
	}

	feeInt := fee.BigInt()
	if !feeInt.IsUint64() {
		return 0, fmt.Errorf("fee exceeds uint64 range")
	}
	return feeInt.Uint64(), nil
}

func AddUint64(left uint64, right uint64) (uint64, error) {
	sum, carry := bits.Add64(left, right, 0)
	if carry != 0 {
		return 0, fmt.Errorf("uint64 addition overflow")
	}
	return sum, nil
}
