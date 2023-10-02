package click

import (
	"fmt"
	"strings"
)

func joinGap(hoon []string) string {
	return strings.Join(hoon, "  ") // gap
}

func storageAction(key, value string) string {
	hoon := joinGap([]string{
		";<",
		"~",
		"bind:m",
		fmt.Sprintf("(poke [our %%storage] %%storage-action !>([%s '%s']))", key, value),
	})
	return hoon
}
