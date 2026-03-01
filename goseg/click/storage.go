package click

import (
	"groundseg/structs"
)

var executeClickCommandForStorage = executeClickCommand

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
