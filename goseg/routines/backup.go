package routines

import (
	"crypto/sha1"
	"fmt"
	"groundseg/config"
	"math/big"
	"time"

	"go.uber.org/zap"
)

func TlonBackupLocal() {
	for {
		// check backups
		zap.L().Debug(fmt.Sprintf("%+v", config.StartramConfig))
		time.Sleep(15 * time.Second)
	}
}

func TlonBackupRemote() {
	for {
		time.Sleep(1 * time.Hour) // temp
	}
}
func generateTimeOfDay(input string) time.Time {
	// Hash the input string to produce a deterministic but unique value
	hash := sha1.New()
	hash.Write([]byte(input))
	hashBytes := hash.Sum(nil)

	// Convert the hash to a big integer
	bigInt := new(big.Int).SetBytes(hashBytes)

	// Use modulo to generate hour, minute, and second values within the appropriate ranges
	hour := bigInt.Mod(bigInt, big.NewInt(24)).Int64()   // Hours: 0-23
	minute := bigInt.Mod(bigInt, big.NewInt(60)).Int64() // Minutes: 0-59
	second := bigInt.Mod(bigInt, big.NewInt(60)).Int64() // Seconds: 0-59

	// Construct a time.Time object with the generated hour, minute, and second
	generatedTime := time.Date(0, time.January, 1, int(hour), int(minute), int(second), 0, time.UTC)

	return generatedTime
}
