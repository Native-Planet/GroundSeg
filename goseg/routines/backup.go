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
func TlonBackupLocal() {
	for {
		conf := config.Conf()
		for _, patp := range conf.Piers {
			// get local tz
			location := time.Now().Location()
			// default backup time is midnight
			backupTime := time.Date(0, 1, 1, 0, 0, 0, 0, location)
			// time format
			timeFormat := "1504"
			// retrieve config
			shipConf := config.UrbitConf(patp)
			// parse backup time
			parsedTime, err := time.ParseInLocation(timeFormat, shipConf.BackupTime, location)
			if err == nil {
				backupTime = parsedTime
			}
			zap.L().Debug(fmt.Sprintf("%s: %v", patp, backupTime)) // temp
			// check if backups exists, if exists get latest timestamp
			// check if backup date is before today
			// check if it is currently later or equal backup time
			// create backup
		}
		time.Sleep(1 * time.Minute)
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
