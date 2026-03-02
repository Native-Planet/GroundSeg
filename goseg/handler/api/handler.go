package api

import (
	"groundseg/handler/authsvc"
	"groundseg/shipworkflow"
	"groundseg/structs"
)

var (
	MaxFailedLogins = authsvc.MaxFailedLogins
	LockoutDuration = authsvc.LockoutDuration
)

var ErrInvalidLoginCredentials = authsvc.ErrInvalidLoginCredentials

func NewShipHandler(msg []byte) error {
	return shipworkflow.HandleNewShip(msg, handleNewShipBoot, handleNewShipCancel, handleNewShipReset)
}

func LoginHandler(conn *structs.MuConn, msg []byte) (map[string]string, error) {
	return authsvc.LoginHandler(conn, msg)
}

func LogoutHandler(msg []byte) error {
	return authsvc.LogoutHandler(msg)
}

func UnauthHandler() ([]byte, error) {
	return authsvc.UnauthHandler()
}

func C2CHandler(msg []byte) ([]byte, error) {
	return authsvc.C2CHandler(msg)
}

func PwHandler(msg []byte, urbitMode bool) error {
	return authsvc.PwHandler(msg, urbitMode)
}
