package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/structs"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Native-Planet/perigee/libprg"
	"github.com/Native-Planet/perigee/roller"
	perigeeTypes "github.com/Native-Planet/perigee/types"
	"go.uber.org/zap"
)

const keysRequestTimeout = 30 * time.Second

type keysBaseRequest struct {
	Token  structs.WsTokenStruct `json:"token"`
	Roller string                `json:"roller,omitempty"`
}

type keysPointRequest struct {
	keysBaseRequest
	Ship    string `json:"ship"`
	Address string `json:"address,omitempty"`
	Hash    string `json:"hash,omitempty"`
}

type keysMaterialRequest struct {
	keysBaseRequest
	Ship       string `json:"ship"`
	Ticket     string `json:"ticket"`
	Passphrase string `json:"passphrase"`
	Life       int    `json:"life"`
	Step       int    `json:"step"`
}

type keysOperationRequest struct {
	keysBaseRequest
	Operation      string `json:"operation"`
	Ship           string `json:"ship"`
	CredentialType string `json:"credentialType"`
	Ticket         string `json:"ticket"`
	PrivateKey     string `json:"privateKey"`
	Passphrase     string `json:"passphrase"`
	Seed           string `json:"seed"`
	Sponsor        string `json:"sponsor"`
	Adoptee        string `json:"adoptee"`
	NewOwner       string `json:"newOwner"`
	Proxy          string `json:"proxy"`
	Reset          bool   `json:"reset"`
}

type keysWalletPrepareRequest struct {
	keysOperationRequest
	Address string `json:"address"`
}

type keysWalletSubmitRequest struct {
	keysWalletPrepareRequest
	Signature string `json:"signature"`
}

type keysErrorResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type keysPendingSummary struct {
	Operation    string `json:"operation,omitempty"`
	Ship         string `json:"ship,omitempty"`
	Hash         string `json:"hash,omitempty"`
	Signature    string `json:"signature,omitempty"`
	Status       string `json:"status,omitempty"`
	SubmittedAt  int64  `json:"submittedAt,omitempty"`
	NextPollAt   int64  `json:"nextPollAt,omitempty"`
	PollInterval int    `json:"pollInterval,omitempty"`
}

type keysPointResponse struct {
	OK        bool                     `json:"ok"`
	Ship      string                   `json:"ship"`
	Point     any                      `json:"point,omitempty"`
	Pending   []perigeeTypes.PendingTx `json:"pending,omitempty"`
	Batch     *perigeeTypes.BatchInfo  `json:"batch,omitempty"`
	Status    string                   `json:"status,omitempty"`
	PendingTx any                      `json:"pendingTx,omitempty"`
}

type keysOperationResponse struct {
	OK              bool                `json:"ok"`
	Ship            string              `json:"ship"`
	Operation       string              `json:"operation"`
	Transaction     any                 `json:"transaction,omitempty"`
	Pending         *keysPendingSummary `json:"pending,omitempty"`
	ExportSuggested bool                `json:"exportSuggested,omitempty"`
	Message         string              `json:"message,omitempty"`
}

type keysPrepareResponse struct {
	OK             bool           `json:"ok"`
	Ship           string         `json:"ship"`
	Operation      string         `json:"operation"`
	Address        string         `json:"address"`
	Seed           string         `json:"seed,omitempty"`
	SigningPayload string         `json:"signingPayload"`
	SignMethod     string         `json:"signMethod"`
	Method         string         `json:"method"`
	From           map[string]any `json:"from"`
	Data           any            `json:"data"`
	Nonce          int            `json:"nonce"`
}

type keysRPCRequest struct {
	Version string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	ID      string `json:"id"`
}

type keysRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func KeysPointHandler(w http.ResponseWriter, r *http.Request) {
	withKeysRequest(w, r, func() {
		var req keysPointRequest
		if !decodeKeysRequest(w, r, &req) || !authorizeKeysRequest(w, r, req.Token) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), keysRequestTimeout)
		defer cancel()
		endpoint := normalizeRollerEndpoint(req.Roller)
		client := rollerClient(endpoint)
		resp := keysPointResponse{OK: true}
		if strings.TrimSpace(req.Ship) != "" {
			ship, err := normalizeShip(req.Ship)
			if err != nil {
				writeKeysError(w, http.StatusBadRequest, err)
				return
			}
			point, err := client.GetPoint(ctx, ship)
			if err != nil {
				writeKeysError(w, http.StatusBadGateway, err)
				return
			}
			if err := point.ResolveSponsorPatp(); err != nil {
				zap.L().Debug("Could not resolve point sponsor", zap.String("ship", ship), zap.Error(err))
			}
			resp.Ship = ship
			resp.Point = point
			if strings.TrimSpace(point.Ownership.Owner.Address) != "" {
				pending, err := client.GetPendingByAddress(ctx, point.Ownership.Owner.Address)
				if err == nil {
					resp.Pending = pending
				}
			}
		} else if strings.TrimSpace(req.Address) != "" {
			address, err := normalizeAddress(req.Address)
			if err != nil {
				writeKeysError(w, http.StatusBadRequest, err)
				return
			}
			pending, err := client.GetPendingByAddress(ctx, address)
			if err != nil {
				writeKeysError(w, http.StatusBadGateway, err)
				return
			}
			resp.Pending = pending
		}
		batch, err := client.WhenNextBatch(ctx)
		if err == nil {
			resp.Batch = batch
		}
		if strings.TrimSpace(req.Hash) != "" {
			status, err := rollerRPC(ctx, endpoint, "getTransactionStatus", map[string]any{"hash": strings.TrimSpace(req.Hash)})
			if err == nil {
				_ = json.Unmarshal(status, &resp.Status)
			}
			pendingTx, err := rollerRPC(ctx, endpoint, "getPendingTx", map[string]any{"hash": strings.TrimSpace(req.Hash)})
			if err == nil {
				var raw any
				if json.Unmarshal(pendingTx, &raw) == nil {
					resp.PendingTx = raw
				}
			}
		}
		writeKeysJSON(w, http.StatusOK, resp)
	})
}

func KeysKeyfileHandler(w http.ResponseWriter, r *http.Request) {
	withKeysRequest(w, r, func() {
		var req keysMaterialRequest
		if !decodeKeysRequest(w, r, &req) || !authorizeKeysRequest(w, r, req.Token) {
			return
		}
		ship, err := normalizeShip(req.Ship)
		if err != nil {
			writeKeysError(w, http.StatusBadRequest, err)
			return
		}
		keyfile, err := libprg.Keyfile(ship, normalizeTicket(req.Ticket), req.Passphrase, req.Life)
		if err != nil {
			writeKeysError(w, http.StatusBadGateway, err)
			return
		}
		writeKeysJSON(w, http.StatusOK, map[string]any{
			"ok":       true,
			"ship":     ship,
			"filename": strings.TrimPrefix(ship, "~") + ".key",
			"keyfile":  keyfile,
		})
	})
}

func KeysCodeHandler(w http.ResponseWriter, r *http.Request) {
	withKeysRequest(w, r, func() {
		var req keysMaterialRequest
		if !decodeKeysRequest(w, r, &req) || !authorizeKeysRequest(w, r, req.Token) {
			return
		}
		ship, err := normalizeShip(req.Ship)
		if err != nil {
			writeKeysError(w, http.StatusBadRequest, err)
			return
		}
		code, err := libprg.GenerateCode(ship, normalizeTicket(req.Ticket), req.Passphrase, req.Life, req.Step)
		if err != nil {
			writeKeysError(w, http.StatusBadGateway, err)
			return
		}
		writeKeysJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"ship": ship,
			"code": code,
		})
	})
}

func KeysOperationHandler(w http.ResponseWriter, r *http.Request) {
	withKeysRequest(w, r, func() {
		var req keysOperationRequest
		if !decodeKeysRequest(w, r, &req) || !authorizeKeysRequest(w, r, req.Token) {
			return
		}
		tx, ship, err := runSoftwareKeysOperation(req)
		if err != nil {
			writeKeysError(w, http.StatusBadRequest, err)
			return
		}
		resp := keysOperationResponse{
			OK:              true,
			Ship:            ship,
			Operation:       normalizeOperation(req.Operation),
			Transaction:     tx,
			Pending:         summarizePending(ship, normalizeOperation(req.Operation), tx),
			ExportSuggested: normalizeOperation(req.Operation) == "breach",
		}
		if resp.ExportSuggested {
			resp.Message = "Export this ship before booting from the new keys. A breach changes continuity."
		}
		writeKeysJSON(w, http.StatusOK, resp)
	})
}

func KeysPrepareWalletHandler(w http.ResponseWriter, r *http.Request) {
	withKeysRequest(w, r, func() {
		var req keysWalletPrepareRequest
		if !decodeKeysRequest(w, r, &req) || !authorizeKeysRequest(w, r, req.Token) {
			return
		}
		prepared, err := prepareWalletOperation(r.Context(), req)
		if err != nil {
			writeKeysError(w, http.StatusBadRequest, err)
			return
		}
		writeKeysJSON(w, http.StatusOK, prepared)
	})
}

func KeysSubmitWalletHandler(w http.ResponseWriter, r *http.Request) {
	withKeysRequest(w, r, func() {
		var req keysWalletSubmitRequest
		if !decodeKeysRequest(w, r, &req) || !authorizeKeysRequest(w, r, req.Token) {
			return
		}
		prepared, err := prepareWalletOperation(r.Context(), req.keysWalletPrepareRequest)
		if err != nil {
			writeKeysError(w, http.StatusBadRequest, err)
			return
		}
		signature := strings.TrimSpace(req.Signature)
		if signature == "" {
			writeKeysError(w, http.StatusBadRequest, fmt.Errorf("signature is required"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), keysRequestTimeout)
		defer cancel()
		endpoint := normalizeRollerEndpoint(req.Roller)
		params := map[string]any{
			"address": prepared.Address,
			"from":    prepared.From,
			"data":    prepared.Data,
			"sig":     signature,
			"force":   false,
		}
		result, err := rollerRPC(ctx, endpoint, prepared.Method, params)
		if err != nil {
			writeKeysError(w, http.StatusBadGateway, err)
			return
		}
		var txHash string
		if err := json.Unmarshal(result, &txHash); err != nil {
			writeKeysError(w, http.StatusBadGateway, fmt.Errorf("unexpected roller response: %v", err))
			return
		}
		tx := perigeeTypes.Transaction{
			Signature: signature,
			Hash:      txHash,
			Type:      prepared.Method,
		}
		operation := normalizeOperation(req.Operation)
		writeKeysJSON(w, http.StatusOK, keysOperationResponse{
			OK:              true,
			Ship:            prepared.Ship,
			Operation:       operation,
			Transaction:     tx,
			Pending:         summarizePending(prepared.Ship, operation, tx),
			ExportSuggested: operation == "breach",
			Message:         walletOperationMessage(operation),
		})
	})
}

func withKeysRequest(w http.ResponseWriter, r *http.Request, next func()) {
	setKeysCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeKeysError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}
	next()
}

func setKeysCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func decodeKeysRequest(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeKeysError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %v", err))
		return false
	}
	return true
}

func authorizeKeysRequest(w http.ResponseWriter, r *http.Request, token structs.WsTokenStruct) bool {
	_, valid, authed := auth.CheckStreamToken(token, r)
	if !valid || !authed {
		writeKeysError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return false
	}
	return true
}

func writeKeysJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		zap.L().Error(fmt.Sprintf("failed to write keys response: %v", err))
	}
}

func writeKeysError(w http.ResponseWriter, status int, err error) {
	zap.L().Warn(fmt.Sprintf("keys request failed: %v", err))
	writeKeysJSON(w, status, keysErrorResponse{OK: false, Error: err.Error()})
}

func runSoftwareKeysOperation(req keysOperationRequest) (any, string, error) {
	ship, err := normalizeShip(req.Ship)
	if err != nil {
		return nil, "", err
	}
	credential, err := operationCredential(req)
	if err != nil {
		return nil, "", err
	}
	operation := normalizeOperation(req.Operation)
	switch operation {
	case "breach":
		if req.CredentialType == "private-key" {
			tx, err := breachWithPrivateKey(ship, credential, req.Passphrase, req.Seed, req.Roller)
			return tx, ship, err
		}
		tx, err := libprg.Breach(ship, credential, req.Passphrase, req.Seed)
		return tx, ship, err
	case "escape":
		sponsor, err := normalizeShip(req.Sponsor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid sponsor: %v", err)
		}
		tx, err := libprg.Escape(ship, credential, req.Passphrase, sponsor)
		return tx, ship, err
	case "cancel-escape":
		sponsor, err := normalizeShip(req.Sponsor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid sponsor: %v", err)
		}
		tx, err := libprg.CancelEscape(ship, credential, req.Passphrase, sponsor)
		return tx, ship, err
	case "adopt":
		adoptee, err := normalizeShip(req.Adoptee)
		if err != nil {
			return nil, "", fmt.Errorf("invalid adoptee: %v", err)
		}
		tx, err := libprg.Adopt(ship, credential, req.Passphrase, adoptee)
		return tx, ship, err
	case "transfer":
		newOwner, err := normalizeAddress(req.NewOwner)
		if err != nil {
			return nil, "", fmt.Errorf("invalid new owner: %v", err)
		}
		tx, err := libprg.TransferOwnership(ship, credential, req.Passphrase, newOwner, req.Reset)
		return tx, ship, err
	case "set-management-proxy":
		proxy, err := normalizeAddress(req.Proxy)
		if err != nil {
			return nil, "", fmt.Errorf("invalid management proxy: %v", err)
		}
		tx, err := libprg.SetManagementProxy(ship, credential, req.Passphrase, proxy)
		return tx, ship, err
	case "set-spawn-proxy":
		proxy, err := normalizeAddress(req.Proxy)
		if err != nil {
			return nil, "", fmt.Errorf("invalid spawn proxy: %v", err)
		}
		tx, err := libprg.SetSpawnProxy(ship, credential, req.Passphrase, proxy)
		return tx, ship, err
	case "set-transfer-proxy":
		proxy, err := normalizeAddress(req.Proxy)
		if err != nil {
			return nil, "", fmt.Errorf("invalid transfer proxy: %v", err)
		}
		tx, err := libprg.SetTransferProxy(ship, credential, req.Passphrase, proxy)
		return tx, ship, err
	default:
		return nil, "", fmt.Errorf("unsupported operation: %s", req.Operation)
	}
}

func prepareWalletOperation(parentCtx context.Context, req keysWalletPrepareRequest) (keysPrepareResponse, error) {
	ship, err := normalizeShip(req.Ship)
	if err != nil {
		return keysPrepareResponse{}, err
	}
	address, err := normalizeAddress(req.Address)
	if err != nil {
		return keysPrepareResponse{}, err
	}
	operation := normalizeOperation(req.Operation)
	ctx, cancel := context.WithTimeout(parentCtx, keysRequestTimeout)
	defer cancel()
	endpoint := normalizeRollerEndpoint(req.Roller)
	rClient := rollerClient(endpoint)
	method, data, proxyKind, seed, err := walletOperationData(operation, req.keysOperationRequest)
	if err != nil {
		return keysPrepareResponse{}, err
	}
	point, err := rClient.GetPoint(ctx, ship)
	if err != nil {
		return keysPrepareResponse{}, fmt.Errorf("getting point: %v", err)
	}
	proxy, err := proxyTypeForAddress(point, address, proxyKind)
	if err != nil {
		return keysPrepareResponse{}, err
	}
	from := map[string]any{"ship": ship, "proxy": proxy}
	client := perigeeTypes.Client{Endpoint: endpoint, HttpClient: http.DefaultClient}
	nonce, err := client.GetNonce(ctx, map[string]any{"from": from})
	if err != nil {
		return keysPrepareResponse{}, fmt.Errorf("getting nonce: %v", err)
	}
	signingPayloadRaw, err := rollerRPC(ctx, endpoint, "prepareForSigning", map[string]any{
		"nonce": nonce,
		"from":  from,
		"tx":    method,
		"data":  data,
	})
	if err != nil {
		return keysPrepareResponse{}, fmt.Errorf("preparing transaction for signing: %v", err)
	}
	var signingPayload string
	if err := json.Unmarshal(signingPayloadRaw, &signingPayload); err != nil {
		return keysPrepareResponse{}, fmt.Errorf("unexpected signing payload: %v", err)
	}
	return keysPrepareResponse{
		OK:             true,
		Ship:           ship,
		Operation:      operation,
		Address:        address,
		Seed:           seed,
		SigningPayload: signingPayload,
		SignMethod:     "personal_sign",
		Method:         method,
		From:           from,
		Data:           data,
		Nonce:          nonce,
	}, nil
}

func walletOperationData(operation string, req keysOperationRequest) (string, any, string, string, error) {
	switch operation {
	case "breach":
		seed := strings.TrimPrefix(strings.TrimSpace(req.Seed), "0x")
		if seed == "" {
			generated, err := defaultNetworkSeed()
			if err != nil {
				return "", nil, "", "", err
			}
			seed = generated
		}
		keys, err := libprg.GenerateNetworkKeysFromSeed(seed)
		if err != nil {
			return "", nil, "", "", err
		}
		return "configureKeys", map[string]any{
			"encrypt":     "0x" + keys.Crypt.Public,
			"auth":        "0x" + keys.Auth.Public,
			"cryptoSuite": "1",
			"breach":      true,
		}, "management", seed, nil
	case "escape":
		sponsor, err := normalizeShip(req.Sponsor)
		if err != nil {
			return "", nil, "", "", fmt.Errorf("invalid sponsor: %v", err)
		}
		return "escape", map[string]any{"ship": sponsor}, "management", "", nil
	case "cancel-escape":
		sponsor, err := normalizeShip(req.Sponsor)
		if err != nil {
			return "", nil, "", "", fmt.Errorf("invalid sponsor: %v", err)
		}
		return "cancel-escape", map[string]any{"ship": sponsor}, "management", "", nil
	case "adopt":
		adoptee, err := normalizeShip(req.Adoptee)
		if err != nil {
			return "", nil, "", "", fmt.Errorf("invalid adoptee: %v", err)
		}
		return "adopt", map[string]any{"ship": adoptee}, "management", "", nil
	case "transfer":
		newOwner, err := normalizeAddress(req.NewOwner)
		if err != nil {
			return "", nil, "", "", fmt.Errorf("invalid new owner: %v", err)
		}
		return "transferPoint", map[string]any{"address": newOwner, "reset": req.Reset}, "transfer", "", nil
	case "set-management-proxy":
		proxy, err := normalizeAddress(req.Proxy)
		if err != nil {
			return "", nil, "", "", fmt.Errorf("invalid management proxy: %v", err)
		}
		return "setManagementProxy", map[string]any{"address": proxy}, "management", "", nil
	case "set-spawn-proxy":
		proxy, err := normalizeAddress(req.Proxy)
		if err != nil {
			return "", nil, "", "", fmt.Errorf("invalid spawn proxy: %v", err)
		}
		return "setSpawnProxy", map[string]any{"address": proxy}, "management", "", nil
	case "set-transfer-proxy":
		proxy, err := normalizeAddress(req.Proxy)
		if err != nil {
			return "", nil, "", "", fmt.Errorf("invalid transfer proxy: %v", err)
		}
		return "setTransferProxy", map[string]any{"address": proxy}, "management", "", nil
	default:
		return "", nil, "", "", fmt.Errorf("unsupported wallet operation: %s", operation)
	}
}

func proxyTypeForAddress(point *perigeeTypes.Point, address, kind string) (string, error) {
	if strings.EqualFold(point.Ownership.Owner.Address, address) {
		return "own", nil
	}
	switch kind {
	case "management":
		if strings.EqualFold(point.Ownership.ManagementProxy.Address, address) {
			return "manage", nil
		}
		return "", fmt.Errorf("connected wallet is neither owner nor management proxy for this ship")
	case "transfer":
		if strings.EqualFold(point.Ownership.TransferProxy.Address, address) {
			return "transfer", nil
		}
		return "", fmt.Errorf("connected wallet is neither owner nor transfer proxy for this ship")
	case "spawn":
		if strings.EqualFold(point.Ownership.SpawnProxy.Address, address) {
			return "spawn", nil
		}
		return "", fmt.Errorf("connected wallet is neither owner nor spawn proxy for this ship")
	default:
		return "", fmt.Errorf("unknown proxy kind: %s", kind)
	}
}

func breachWithPrivateKey(ship, privateKey, passphrase, seed, rollerEndpoint string) (*perigeeTypes.Transaction, error) {
	seed = strings.TrimPrefix(strings.TrimSpace(seed), "0x")
	if seed == "" {
		generated, err := defaultNetworkSeed()
		if err != nil {
			return nil, err
		}
		seed = generated
	}
	privKey, derivedPubkey, pointInfo, networkKeys, _, err := libprg.ValidateKey(ship, privateKey, passphrase, seed, true)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", libprg.ErrKeyMaterial, err)
	}
	if pointInfo.Dominion != "l2" {
		return nil, fmt.Errorf("private-key breach through Roller is only available for L2 ships")
	}
	return rollerClient(normalizeRollerEndpoint(rollerEndpoint)).ConfigureKeys(
		context.Background(),
		ship,
		"0x"+networkKeys.Crypt.Public,
		"0x"+networkKeys.Auth.Public,
		true,
		derivedPubkey,
		privKey,
	)
}

func defaultNetworkSeed() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate network seed: %v", err)
	}
	return hex.EncodeToString(buf), nil
}

func operationCredential(req keysOperationRequest) (string, error) {
	switch req.CredentialType {
	case "ticket", "master-ticket", "":
		ticket := normalizeTicket(req.Ticket)
		if strings.TrimSpace(ticket) == "" {
			return "", fmt.Errorf("master ticket is required")
		}
		return ticket, nil
	case "private-key":
		privateKey := strings.TrimPrefix(strings.TrimSpace(req.PrivateKey), "0x")
		if privateKey == "" {
			return "", fmt.Errorf("ethereum private key is required")
		}
		return privateKey, nil
	default:
		return "", fmt.Errorf("unsupported credential type: %s", req.CredentialType)
	}
}

func normalizeShip(ship string) (string, error) {
	patp, _, err := perigeeTypes.ValidateAndNormalizePatp(strings.TrimSpace(ship))
	if err != nil {
		return "", err
	}
	return patp, nil
}

func normalizeAddress(address string) (string, error) {
	addr := strings.TrimSpace(address)
	if addr == "" {
		return "", fmt.Errorf("address is required")
	}
	return roller.ValidateAddress(addr, false)
}

func normalizeTicket(ticket string) string {
	ticket = strings.TrimSpace(ticket)
	if ticket != "" && !strings.HasPrefix(ticket, "~") {
		return "~" + ticket
	}
	return ticket
}

func normalizeOperation(operation string) string {
	return strings.TrimSpace(strings.ToLower(operation))
}

func normalizeRollerEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	if endpoint == "" {
		return roller.RollerURL
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return endpoint
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/v1/roller"
	}
	return parsed.String()
}

func rollerClient(endpoint string) *roller.Roller {
	return roller.New(roller.Config{Endpoint: endpoint, HTTPClient: http.DefaultClient})
}

func summarizePending(ship, operation string, tx any) *keysPendingSummary {
	raw, err := json.Marshal(tx)
	if err != nil {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	sig := stringFromAny(values["sig"])
	if sig == "" {
		sig = stringFromAny(values["Signature"])
	}
	hash := stringFromAny(values["hash"])
	if hash == "" {
		hash = stringFromAny(values["Hash"])
	}
	txType := stringFromAny(values["type"])
	if txType == "" {
		txType = stringFromAny(values["Type"])
	}
	if operation == "" {
		operation = txType
	}
	if sig == "" && hash == "" {
		return nil
	}
	now := time.Now()
	return &keysPendingSummary{
		Operation:    operation,
		Ship:         ship,
		Hash:         hash,
		Signature:    sig,
		Status:       "pending",
		SubmittedAt:  now.UnixMilli(),
		NextPollAt:   now.Add(time.Minute).UnixMilli(),
		PollInterval: 60,
	}
}

func stringFromAny(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func walletOperationMessage(operation string) string {
	if operation == "breach" {
		return "Export this ship before booting from the new keys. A breach changes continuity."
	}
	return ""
}

func rollerRPC(ctx context.Context, endpoint, method string, params any) (json.RawMessage, error) {
	reqPayload := keysRPCRequest{
		Version: "2.0",
		Method:  method,
		Params:  params,
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal roller request: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create roller request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("roller request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read roller response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("roller status %d: %s", resp.StatusCode, string(respBody))
	}
	var rpcResp keysRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("decode roller response: %v", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("roller rpc error: %s", rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}
