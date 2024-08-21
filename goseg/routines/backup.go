package routines

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"groundseg/config"
	"math/big"
	"time"

	"go.uber.org/zap"
)

func TlonBackupLocal() {
	for {
		// check local backup time
		// create backup
		zap.L().Debug("temporary backup print for local backups")
		time.Sleep(15 * time.Second) // temp
	}
}

type BackupTime struct {
	IsSet bool
	Time  time.Time
}

func TlonBackupRemote() {
	backupTime := BackupTime{IsSet: false}
	for {
		if !backupTime.IsSet {
			backupTime = BackupTime{IsSet: true, Time: generateTimeOfDay(config.StartramConfig.UrlID)}
		}
		zap.L().Debug(fmt.Sprintf("remote backup time: %+v", backupTime.Time))
		time.Sleep(15 * time.Second) // temp
	}
}
func generateTimeOfDay(input string) time.Time {
	// modulos
	mod24 := big.NewInt(24)
	mod60 := big.NewInt(60)
	// time maker
	makeTime := func(text string, mod *big.Int) int64 {
		// get hash
		hashed := sha256.Sum256([]byte(text))
		hex := hex.EncodeToString(hashed[:])
		// to big int
		bigInt := new(big.Int)
		bigInt.SetString(hex, 16)
		// mod and convert to int64
		return new(big.Int).Mod(bigInt, mod).Int64()
	}
	hour := makeTime(input+"hour", mod24)
	minute := makeTime(input+"minute", mod60)
	second := makeTime(input+"second", mod60)
	// Construct a time.Time object with the generated hour, minute, and second
	generatedTime := time.Date(0, time.January, 1, int(hour), int(minute), int(second), 0, time.UTC)
	return generatedTime
}
