package telegram

import (
	"fmt"
	"strconv"
	"strings"
)

type callbackAction string

const (
	callbackOpenMenu callbackAction = "menu:open"

	callbackListNew       callbackAction = "list:new"
	callbackListScheduled callbackAction = "list:scheduled"
)

func encodeCallback(action callbackAction, id int64) string {
	return fmt.Sprintf("%s:%d", action, id)
}

func parseCallbackID(wantAction callbackAction, data string) (int64, error) {
	action, idStr, ok := strings.Cut(data, ":")
	if !ok || action != string(wantAction) {
		return 0, fmt.Errorf("callback: expected action %q got %q", wantAction, action)
	}
	return strconv.ParseInt(idStr, 10, 64)
}
