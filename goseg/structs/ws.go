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

type WsTokenStruct struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type WsLoginPayload struct {
	Type     string        `json:"type"`
	Password string        `json:"password"`
	Token    WsTokenStruct `json:"token"`
}

type WsResponsePayload struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Response string        `json:"response"`
	Error    string        `json:"error"`
	Token    WsTokenStruct `json:"token"`
}
