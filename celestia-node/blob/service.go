package blob

import (
	"errors"
)

var (
	ErrBlobNotFound = errors.New("blob: not found")
	ErrInvalidProof = errors.New("blob: invalid proof")
)

// GasPrice represents the amount to be paid per gas unit. Fee is set by
// multiplying GasPrice by GasLimit, which is determined by the blob sizes.
type GasPrice float64

// DefaultGasPrice returns the default gas price, letting node automatically
// determine the Fee based on the passed blob sizes.
func DefaultGasPrice() GasPrice {
	return -1.0
}
