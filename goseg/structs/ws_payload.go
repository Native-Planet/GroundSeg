package structs

import "context"

type WsType struct {
	Payload struct {
		Type string `json:"type"`
	} `json:"payload"`
}

type C2CPayload struct {
	Type     string `json:"type"`
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

type WsPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload interface{}   `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type GallsegPayload struct {
	Payload struct {
		Type string `json:"type"`
	} `json:"payload"`
}

type WsUrbitPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsUrbitAction `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsDevPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsDevAction   `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsPenpaiPayload struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload WsPenpaiAction `json:"payload"`
	Token   WsTokenStruct  `json:"token"`
}

type WsPenpaiAction struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Model  string `json:"model"`
	Cores  int    `json:"cores"`
}

type WsUrbitAction struct {
	Type         string `json:"type"`
	Action       string `json:"action"`
	Patp         string `json:"patp"`
	Value        int    `json:"value"`
	Domain       string `json:"domain"`
	Frequency    int    `json:"frequency"`
	IntervalType string `json:"intervalType"`
	Time         string `json:"time"`
	Day          string `json:"day"`
	Date         int    `json:"date"`
	Remind       bool   `json:"remind"`
	Service      string `json:"service"`
	Remote       bool   `json:"remote"`
	Timestamp    int    `json:"timestamp"`
	MD5          string `json:"md5"`
	BackupTime   string `json:"backupTime"`
	BakType      string `json:"bakType"`
}

type WsDevAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Patp     string `json:"patp"`
	Remote   bool   `json:"remote"`
	Reminded bool   `json:"reminded"`
}

type WsNewShipPayload struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload WsNewShipAction `json:"payload"`
	Token   WsTokenStruct   `json:"token"`
}

type WsNewShipAction struct {
	Type          string `json:"type"`
	Action        string `json:"action"`
	Patp          string `json:"patp"`
	Key           string `json:"key"`
	KeyType       string `json:"keyType"`
	Remote        bool   `json:"remote"`
	Command       string `json:"command"`
	SelectedDrive string `json:"selectedDrive"`
}

type WsUploadPayload struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload WsUploadAction `json:"payload"`
	Token   WsTokenStruct  `json:"token"`
}

type WsUploadAction struct {
	Type          string `json:"type"`
	Action        string `json:"action"`
	Endpoint      string `json:"endpoint"`
	Remote        bool   `json:"remote"`
	Fix           bool   `json:"fix"`
	SelectedDrive string `json:"selectedDrive"`
}

type WsLogsPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsLogsAction  `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsLogsAction struct {
	Action      bool   `json:"action"`
	ContainerID string `json:"container_id"`
}

type WsSystemPayload struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload WsSystemAction `json:"payload"`
	Token   WsTokenStruct  `json:"token"`
}

type WsSystemAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Command  string `json:"command"`
	Value    int    `json:"value"`
	Update   string `json:"update"`
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

type WsTokenStruct struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type WsLoginPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsLoginAction `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsLoginAction struct {
	Type     string `json:"type"`
	Password string `json:"password"`
}

type WsPwPayload struct {
	ID      string        `json:"id"`
	Payload WsPwAction    `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsPwAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Old      string `json:"old"`
	Password string `json:"password"`
}

type WsSwapPayload struct {
	ID      string        `json:"id"`
	Payload WsSwapAction  `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsSwapAction struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Value  int    `json:"value"`
}

type WsLogoutPayload struct {
	ID    string        `json:"id"`
	Token WsTokenStruct `json:"token"`
}

type WsResponsePayload struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Response string        `json:"response"`
	Error    string        `json:"error"`
	Token    WsTokenStruct `json:"token"`
}

type WsStartramPayload struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Payload WsStartramAction `json:"payload"`
	Token   WsTokenStruct    `json:"token"`
}

type WsStartramAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Key      string `json:"key"`
	Region   string `json:"region"`
	Endpoint string `json:"endpoint"`
	Reset    bool   `json:"reset"`
	Remind   bool   `json:"remind"`
	Backup   int    `json:"backup"`
	Patp     string `json:"patp"`
	Target   string `json:"target"`
	Password string `json:"password"`
}

type WsLogMessage struct {
	Log struct {
		ContainerID string `json:"container_id"`
		Line        string `json:"line"`
	} `json:"log"`
	Type string `json:"type"`
}

type WsSetupPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsSetupAction `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsSetupAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	Password string `json:"password"`
	Key      string `json:"key"`
	Region   string `json:"region"`
}

type WsSupportPayload struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload WsSupportAction `json:"payload"`
	Token   WsTokenStruct   `json:"token"`
}

type WsSupportAction struct {
	Type        string   `json:"type"`
	Action      string   `json:"action"`
	Contact     string   `json:"contact"`
	Description string   `json:"description"`
	Ships       []string `json:"ships"`
	CPUProfile  bool     `json:"cpu_profile"`
	Penpai      bool     `json:"penpai"`
}

type WsC2CPayload struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"` // "c2c"
	Payload WsC2CAction `json:"payload"`
}

type WsC2CAction struct {
	Type     string `json:"type"`
	Action   string `json:"action"`
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

type CtxWithCancel struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}
