package storage

import (
	"fmt"

	"groundseg/click/internal/runtime"
	"groundseg/click/luscode"
	"groundseg/structs"
)

var (
	executeClickCommandForStorage = runtime.ExecuteCommand
	createHoonForStorage          = runtime.CreateHoon
	deleteHoonForStorage          = runtime.DeleteHoon
	clearLusCode                  = luscode.ClearLusCode
)

func createHoonForStorageCommand(patp, file, hoon string) error {
	if err := createHoonForStorage(patp, file, hoon); err != nil {
		return err
	}
	clearLusCode(patp)
	return nil
}

func UnlinkStorage(patp string) error {
	return unlinkStorage(patp)
}

func LinkStorage(patp, endpoint string, svcAccount structs.MinIOServiceAccount) error {
	return linkStorage(patp, endpoint, svcAccount)
}

func unlinkStorage(patp string) error {
	file := "unlinkstorage"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "our=@p", "bind:m", "get-our",
		storageAction("%set-endpoint", ""),
		storageAction("%set-access-key-id", ""),
		storageAction("%set-secret-access-key", ""),
		storageAction("%set-current-bucket", ""),
		"(pure:m !>('success'))",
	})
	_, err := executeClickCommandForStorage(patp, file, hoon, "", "success", "Click unlink storage")
	return err
}

func linkStorage(patp, endpoint string, svcAccount structs.MinIOServiceAccount) error {
	file := "linkstorage"
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "our=@p", "bind:m", "get-our",
		storageAction("%set-endpoint", endpoint),
		storageAction("%set-access-key-id", svcAccount.AccessKey),
		storageAction("%set-secret-access-key", svcAccount.SecretKey),
		storageAction("%set-current-bucket", "bucket"),
		"(pure:m !>('success'))",
	})
	_, err := executeClickCommandForStorage(patp, file, hoon, "", "success", "Click link storage")
	return err
}

func joinGap(parts []string) string {
	return runtime.JoinGap(parts)
}

func storageAction(key, value string) string {
	return joinGap([]string{
		";<",
		"~",
		"bind:m",
		fmt.Sprintf("(poke [our %%storage] %%storage-action !>([%s '%s']))", key, value),
	})
}
