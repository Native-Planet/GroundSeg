package structs

type WsType struct {
	Payload struct {
		Type string `json:"type"`
	} `json:"payload"`
}

type WsPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload interface{}   `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsUrbitPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsUrbitAction `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsSystemPayload struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload WsSystemAction `json:"payload"`
	Token   WsTokenStruct  `json:"token"`
}

type WsUrbitAction struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Patp   string `json:"patp"`
}

type WsSystemAction struct {
	Type    string `json:"type"`
	Action  string `json:"action"`
	Command string `json:"command"`
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

type WsLogoutPayload struct {
	ID      string        `json:"id"`
	Token   WsTokenStruct `json:"token"`
}

type WsResponsePayload struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Response string        `json:"response"`
	Error    string        `json:"error"`
	Token    WsTokenStruct `json:"token"`
}

type WsStartramPayload struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Payload WsStartramAction `json:"payload"`
	Token   WsTokenStruct `json:"token"`
}

type WsStartramAction struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Key    string `json:"key"`
	Region string `json:"region"`
}