package api

import (
	"groundseg/handler/auth"
	"groundseg/shipworkflow"
	"groundseg/structs"
)

var (
	MaxFailedLogins = auth.MaxFailedLogins
	LockoutDuration = auth.LockoutDuration
)

var ErrInvalidLoginCredentials = auth.ErrInvalidLoginCredentials

func NewShipHandler(msg []byte) error {
	return shipworkflow.HandleNewShip(msg, handleNewShipBoot, handleNewShipCancel, handleNewShipReset)
}

func LoginHandler(conn *structs.MuConn, msg []byte) (map[string]string, error) {
	return auth.LoginHandler(conn, msg)
}

func LogoutHandler(msg []byte) error {
	return auth.LogoutHandler(msg)
}

func UnauthHandler() ([]byte, error) {
	return auth.UnauthHandler()
}

func C2CHandler(msg []byte) ([]byte, error) {
	return auth.C2CHandler(msg)
}

func PwHandler(msg []byte, urbitMode bool) error {
	return auth.PwHandler(msg, urbitMode)
}
