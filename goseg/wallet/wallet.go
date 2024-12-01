package wallet

import (
	"github.com/deelawn/urbit-gob/co"
	"github.com/nathanlever/keygen"
)

func Wallet(ship, ticket string, revision int) (keygen.Wallet, error) {
	bigPoint, err := co.Patp2Point(ship)
	if err != nil {
		return keygen.Wallet{}, err
	}
	return keygen.GenerateWallet(ticket, uint32(bigPoint.Int64()), "", uint(revision), true), nil
}
