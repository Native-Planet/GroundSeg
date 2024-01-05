package click

import (
	"fmt"
	"groundseg/structs"
)

/*
=/  m  (strand ,vase)
;<    our=@p
    bind:m
  get-our
;<    ~
    bind:m
  (poke [our %storage] %storage-action !>([%set-endpoint '{payload['endpoint']}']))
;<    ~
    bind:m
  (poke [our %storage] %storage-action !>([%set-access-key-id '{payload['acc']}']))
;<    ~
    bind:m
  (poke [our %storage] %storage-action !>([%set-secret-access-key '{payload['secret']}']))
;<    ~
    bind:m
  (poke [our %storage] %storage-action !>([%set-current-bucket '{payload['bucket']}']))
(pure:m !>('success'))
*/

func unlinkStorage(patp string) error {
	// <file>.hoon
	file := "unlinkstorage"
	// actual hoon
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "our=@p", "bind:m", "get-our",
		storageAction("%set-endpoint", ""),
		storageAction("%set-access-key-id", ""),
		storageAction("%set-secret-access-key", ""),
		storageAction("%set-current-bucket", ""),
		"(pure:m !>('success'))",
	})
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click unlink storage failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click unlink storage failed to get exec: %v", err)
	}
	_, succeeded, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click unlink storage failed to get exec: %v", err)
	}
	if !succeeded {
		return fmt.Errorf("Click unlink storage failed poke: %s", patp)
	}
	return nil
}
func linkStorage(patp, endpoint string, svcAccount structs.MinIOServiceAccount) error {
	// <file>.hoon
	file := "linkstorage"
	// actual hoon
	hoon := joinGap([]string{
		"=/", "m", "(strand ,vase)",
		";<", "our=@p", "bind:m", "get-our",
		storageAction("%set-endpoint", endpoint),
		storageAction("%set-access-key-id", svcAccount.AccessKey),
		storageAction("%set-secret-access-key", svcAccount.SecretKey),
		storageAction("%set-current-bucket", "bucket"),
		"(pure:m !>('success'))",
	})
	// create hoon file
	if err := createHoon(patp, file, hoon); err != nil {
		return fmt.Errorf("Click link storage failed to create hoon: %v", err)
	}
	// defer hoon file deletion
	defer deleteHoon(patp, file)
	// execute hoon file
	response, err := clickExec(patp, file, "")
	if err != nil {
		return fmt.Errorf("Click link storage failed to get exec: %v", err)
	}
	_, succeeded, err := filterResponse("success", response)
	if err != nil {
		return fmt.Errorf("Click link storage failed to get exec: %v", err)
	}
	if !succeeded {
		return fmt.Errorf("Click link storage failed poke: %s", patp)
	}
	return nil
}
