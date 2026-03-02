package storage

import (
	"fmt"

	"groundseg/click/internal/runtime"
	"groundseg/structs"
)

var (
	executeClickCommandForStorage = runtime.ExecuteCommandWithSuccess
)

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
	_, err := executeClickCommandForStorage(patp, file, hoon, "", "success", "Click unlink storage", nil)
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
	_, err := executeClickCommandForStorage(patp, file, hoon, "", "success", "Click link storage", nil)
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
