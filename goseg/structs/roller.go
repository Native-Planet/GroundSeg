package structs

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"reflect"
	"strings"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/deelawn/urbit-gob/co"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	CryptoSuiteVersion = 1
)

type Point struct {
	Dominion  string `json:"dominion"`
	Ownership struct {
		Owner struct {
			Address string `json:"address"`
			Nonce   int    `json:"nonce"`
		} `json:"owner"`
		ManagementProxy struct {
			Address string `json:"address"`
			Nonce   int    `json:"nonce"`
		} `json:"managementProxy"`
		SpawnProxy struct {
			Address string `json:"address"`
			Nonce   int    `json:"nonce"`
		} `json:"spawnProxy"`
		TransferProxy struct {
			Address string `json:"address"`
			Nonce   int    `json:"nonce"`
		} `json:"transferProxy"`
	} `json:"ownership"`
	Network struct {
		Keys struct {
			Life  string `json:"life"`
			Suite string `json:"suite"`
			Auth  string `json:"auth"`
			Crypt string `json:"crypt"`
		} `json:"keys"`
		Sponsor struct {
			Has bool `json:"has"`
			Who int  `json:"who"`
		} `json:"sponsor"`
		Rift string `json:"rift"`
	} `json:"network"`
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type JsonRPCRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      string      `json:"id"`
}

type JsonRPCRequestParamSlice struct {
	Version string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      string        `json:"id"`
}

type JsonRPCResponse struct {
	Version string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	ID string `json:"id"`
}

type TransferRequest struct {
	Address string   `json:"address"`
	From    FromData `json:"from"`
	Data    struct {
		Reset   bool   `json:"reset"`
		Address string `json:"address"`
	} `json:"data"`
	Signature string `json:"sig,omitempty"`
}

type ConfigureKeysParams struct {
	Encrypt     string `json:"encrypt"`
	Auth        string `json:"auth"`
	CryptoSuite string `json:"cryptoSuite"`
	Breach      bool   `json:"breach"`
}

type SpawnRequest struct {
	Address string   `json:"address"`
	From    FromData `json:"from"`
	Data    struct {
		Ship    string `json:"ship"`
		Address string `json:"address"`
	} `json:"data"`
	Signature string `json:"sig,omitempty"`
}

type ProxyRequest struct {
	Address string   `json:"address"`
	From    FromData `json:"from"`
	Data    struct {
		Address string `json:"address"`
	} `json:"data"`
	Signature string `json:"sig,omitempty"`
}

type RollerConfig struct {
	BatchSize     int    `json:"batchSize"`
	BatchInterval int    `json:"batchInterval"`
	ChainID       int    `json:"chainId"`
	ContractData  string `json:"contractData"`
}

type BatchInfo struct {
	TimeUntilNext int `json:"timeUntilNext"`
	BatchSize     int `json:"batchSize"`
}

type ShipInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type RawTx struct {
	Tx struct {
		Data struct {
			Breach      bool    `json:"breach"`
			CryptoSuite int     `json:"cryptoSuite"`
			Auth        big.Int `json:"auth,string"`
			Encrypt     big.Int `json:"encrypt,string"`
		} `json:"data"`
		Type string `json:"type"`
	} `json:"tx"`
	From struct {
		Ship  string `json:"ship"`
		Proxy string `json:"proxy"`
	} `json:"from"`
	Sig string `json:"sig"`
}

type PendingTx struct {
	RawTx struct {
		Tx struct {
			Data struct {
				Breach      bool    `json:"breach"`
				CryptoSuite int     `json:"cryptoSuite"`
				Auth        big.Int `json:"auth,string"`
				Encrypt     big.Int `json:"encrypt,string"`
			} `json:"data"`
			Type string `json:"type"`
		} `json:"tx"`
		From struct {
			Ship  string `json:"ship"`
			Proxy string `json:"proxy"`
		} `json:"from"`
		Sig string `json:"sig"`
	} `json:"rawTx"`
	Address string `json:"address"`
	Force   bool   `json:"force"`
	Time    int64  `json:"time"`
}

type TransactionRequest struct {
	Method string      `json:"tx"`
	Nonce  string      `json:"nonce"`
	From   FromData    `json:"from"`
	Data   interface{} `json:"data"`
}

type Transaction struct {
	Signature string `json:"sig"`
	Hash      string `json:"hash"`
	Type      string `json:"type"`
}

type FromData struct {
	Ship  string `json:"ship"`
	Proxy string `json:"proxy"`
}

type RollerClient struct {
	Endpoint   string
	HttpClient *http.Client
	ReqCounter atomic.Int64
}

func (c *RollerClient) Request(method string, params interface{}) (json.RawMessage, error) {
	req := JsonRPCRequest{
		Version: "2.0",
		Method:  method,
		Params:  params,
		ID:      c.NextID(),
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	zap.L().Info("Request", zap.String("body", string(reqBody)))

	httpReq, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	zap.L().Info("Response", zap.String("status", resp.Status), zap.String("body", string(bodyBytes)))

	var rpcResp JsonRPCResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// map of methods that require params to be converted to array format
var arrayParamMethods = map[string]bool{
	"getUnsignedTx": true,
	// add others as needed
}

func (c *RollerClient) paramsToArray(method string, params interface{}) (interface{}, error) {
	if !arrayParamMethods[method] {
		return params, nil
	}
	switch method {
	case "getUnsignedTx":
		var p struct {
			Tx    string      `json:"tx"`
			Nonce string      `json:"nonce"`
			From  FromData    `json:"from"`
			Data  interface{} `json:"data"`
		}
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		if err := json.Unmarshal(b, &p); err != nil {
			return nil, fmt.Errorf("unmarshal params: %w", err)
		}
		return []interface{}{
			p.Tx,
			p.Nonce,
			p.From,
			p.Data,
		}, nil
	}

	return params, nil
}

func (c *RollerClient) IsOwner(ctx context.Context, point, address string) (bool, error) {
	pointInfo, err := c.GetPoint(ctx, point)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(pointInfo.Ownership.Owner.Address, address), nil
}

func (c *RollerClient) NextID() string {
	return fmt.Sprintf("%d", c.ReqCounter.Add(1))
}

func (c *RollerClient) DoRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	finalParams, err := c.paramsToArray(method, params)
	if err != nil {
		return nil, fmt.Errorf("convert params: %w", err)
	}
	req := JsonRPCRequest{
		Version: "2.0",
		Method:  method,
		Params:  finalParams,
		ID:      c.NextID(),
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	zap.L().Info("Request", zap.String("body", string(reqBody)))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	zap.L().Info("Response", zap.String("status", resp.Status), zap.String("body", string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	var rpcResp JsonRPCResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w, body: %s", err, string(bodyBytes))
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: %s, body: %s", rpcResp.Error.Message, string(bodyBytes))
	}

	return rpcResp.Result, nil
}

func (c *RollerClient) ConfigureKeys(ctx context.Context, point, encryptPublic, authPublic string, breach bool, signingAddress string, privateKey *ecdsa.PrivateKey) (*Transaction, error) {
	proxyType, err := c.GetManagementProxyType(ctx, point, signingAddress)
	if err != nil {
		return nil, err
	}

	params := struct {
		Address string   `json:"address"`
		From    FromData `json:"from"`
		Data    struct {
			Encrypt     string `json:"encrypt"`
			Auth        string `json:"auth"`
			CryptoSuite string `json:"cryptoSuite"`
			Breach      bool   `json:"breach"`
		} `json:"data"`
		Signature string `json:"sig,omitempty"`
	}{
		Address: signingAddress,
		From: FromData{
			Ship:  point,
			Proxy: proxyType,
		},
		Data: struct {
			Encrypt     string `json:"encrypt"`
			Auth        string `json:"auth"`
			CryptoSuite string `json:"cryptoSuite"`
			Breach      bool   `json:"breach"`
		}{
			Encrypt:     encryptPublic,
			Auth:        authPublic,
			CryptoSuite: fmt.Sprintf("%d", CryptoSuiteVersion),
			Breach:      breach,
		},
	}
	if err := c.addSignature(ctx, "configureKeys", &params, privateKey); err != nil {
		return nil, fmt.Errorf("add signature: %w", err)
	}
	result, err := c.Request("configureKeys", params)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	var txHash string
	if err := json.Unmarshal(result, &txHash); err != nil {
		return nil, fmt.Errorf("unmarshal transaction hash: %w", err)
	}
	return &Transaction{
		Signature: params.Signature,
		Hash:      txHash,
		Type:      "configureKeys",
	}, nil
}

func (c *RollerClient) GetManagementProxyType(ctx context.Context, point, signingAddress string) (string, error) {
	pointInfo, err := c.GetPoint(ctx, point)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(pointInfo.Ownership.Owner.Address, signingAddress) {
		return "own", nil
	}
	if strings.EqualFold(pointInfo.Ownership.ManagementProxy.Address, signingAddress) {
		return "manage", nil
	}
	return "", errors.New("address is neither owner nor management proxy")
}

func (c *RollerClient) GetNonce(ctx context.Context, params interface{}) (int, error) {
	var fromData struct {
		From FromData `json:"from"`
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return 0, fmt.Errorf("marshal params: %w", err)
	}

	if err := json.Unmarshal(paramsJSON, &fromData); err != nil {
		return 0, fmt.Errorf("unmarshal from data: %w", err)
	}

	nonceParams := struct {
		From FromData `json:"from"`
	}{
		From: fromData.From,
	}

	result, err := c.Request("getNonce", nonceParams)
	if err != nil {
		return 0, fmt.Errorf("get nonce: %w", err)
	}
	var nonce int
	if err := json.Unmarshal(result, &nonce); err != nil {
		return 0, fmt.Errorf("unmarshal nonce: %w", err)
	}

	return nonce, nil
}

func (c *RollerClient) GetUnsignedTx(ctx context.Context, method string, params interface{}) (string, error) {
	nonce, err := c.GetNonce(ctx, params)
	if err != nil {
		return "", err
	}
	var reqParams struct {
		From FromData    `json:"from"`
		Data interface{} `json:"data"`
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("marshal params: %w", err)
	}

	if err := json.Unmarshal(paramsJSON, &reqParams); err != nil {
		return "", fmt.Errorf("unmarshal params: %w", err)
	}
	hashParams := struct {
		Tx    string      `json:"tx"`
		Nonce int         `json:"nonce"`
		From  FromData    `json:"from"`
		Data  interface{} `json:"data"`
	}{
		Tx:    method,
		Nonce: nonce,
		From:  reqParams.From,
		Data:  reqParams.Data,
	}
	result, err := c.Request("getUnsignedTx", hashParams)
	if err != nil {
		return "", fmt.Errorf("get unsigned tx: %w", err)
	}
	var hash string
	if err := json.Unmarshal(result, &hash); err != nil {
		return "", fmt.Errorf("unmarshal hash: %w", err)
	}

	return hash, nil
}

func SignTransactionHash(hash string, privateKey *ecdsa.PrivateKey) (string, error) {
	if !strings.HasPrefix(hash, "0x") {
		return "", fmt.Errorf("hash must start with 0x")
	}
	cleanHash := strings.TrimPrefix(hash, "0x")
	hashBytes, err := hex.DecodeString(cleanHash)
	if err != nil {
		return "", fmt.Errorf("decode hash: %w", err)
	}
	if len(hashBytes) != 32 {
		return "", fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(hashBytes))
	}
	signature, err := crypto.Sign(hashBytes, privateKey)
	if err != nil {
		return "", fmt.Errorf("sign hash: %w", err)
	}
	// convert signature to eth format
	v := signature[64]
	if v < 27 {
		signature[64] = v + 27
	}

	return "0x" + hex.EncodeToString(signature), nil
}

func (c *RollerClient) addSignature(ctx context.Context, method string, params interface{}, privateKey *ecdsa.PrivateKey) error {
	hash, err := c.GetUnsignedTx(ctx, method, params)
	if err != nil {
		return fmt.Errorf("get unsigned tx: %w", err)
	}
	signature, err := SignTransactionHash(hash, privateKey)
	if err != nil {
		return fmt.Errorf("sign transaction: %w", err)
	}
	paramsValue := reflect.ValueOf(params).Elem()
	sigField := paramsValue.FieldByName("Signature")
	if !sigField.IsValid() || !sigField.CanSet() {
		return fmt.Errorf("params struct must have settable Signature field")
	}
	sigField.SetString(signature)
	return nil
}

func ValidatePrivateKey(privateKey *ecdsa.PrivateKey) error {
	if privateKey == nil {
		return fmt.Errorf("private key is nil")
	}
	curveParams := crypto.S256().Params()
	if privateKey.D.Cmp(big.NewInt(0)) <= 0 ||
		privateKey.D.Cmp(new(big.Int).Sub(curveParams.N, big.NewInt(1))) >= 0 {
		return fmt.Errorf("private key out of allowed range")
	}
	return nil
}

func (c *RollerClient) TransferPoint(ctx context.Context, point string, reset bool, newOwnerAddress string, signingAddress string, privateKey *ecdsa.PrivateKey) (*Transaction, error) {
	proxyType, err := c.GetTransferProxyType(ctx, point, signingAddress)
	if err != nil {
		return nil, fmt.Errorf("get transfer proxy type: %w", err)
	}

	params := TransferRequest{
		Address: signingAddress,
		From: FromData{
			Ship:  point,
			Proxy: proxyType,
		},
		Data: struct {
			Reset   bool   `json:"reset"`
			Address string `json:"address"`
		}{
			Reset:   reset,
			Address: newOwnerAddress,
		},
	}

	if err := c.addSignature(ctx, "transferPoint", &params, privateKey); err != nil {
		return nil, fmt.Errorf("add signature: %w", err)
	}
	result, err := c.DoRequest(ctx, "transferPoint", params)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	var txHash string
	if err := json.Unmarshal(result, &txHash); err != nil {
		return nil, fmt.Errorf("unmarshal transaction hash: %w", err)
	}
	return &Transaction{
		Signature: params.Signature,
		Hash:      txHash,
		Type:      "transferPoint",
	}, nil
}

func (c *RollerClient) SetManagementProxy(ctx context.Context, point, proxyAddress, signingAddress string, privateKey *ecdsa.PrivateKey) (*Transaction, error) {
	params := ProxyRequest{
		Address: signingAddress,
		From: FromData{
			Ship:  point,
			Proxy: "own",
		},
		Data: struct {
			Address string `json:"address"`
		}{
			Address: proxyAddress,
		},
	}

	if err := c.addSignature(ctx, "setManagementProxy", &params, privateKey); err != nil {
		return nil, fmt.Errorf("add signature: %w", err)
	}
	result, err := c.DoRequest(ctx, "setManagementProxy", params)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	var txHash string
	if err := json.Unmarshal(result, &txHash); err != nil {
		return nil, fmt.Errorf("unmarshal transaction hash: %w", err)
	}
	return &Transaction{
		Signature: params.Signature,
		Hash:      txHash,
		Type:      "setManagementProxy",
	}, nil
}

func (c *RollerClient) GetSpawnProxyType(ctx context.Context, point, signingAddress string) (string, error) {
	pointInfo, err := c.GetPoint(ctx, point)
	if err != nil {
		return "", fmt.Errorf("get point info: %w", err)
	}
	if strings.EqualFold(pointInfo.Ownership.Owner.Address, signingAddress) {
		return "own", nil
	}
	if strings.EqualFold(pointInfo.Ownership.SpawnProxy.Address, signingAddress) {
		return "spawn", nil
	}
	return "", fmt.Errorf("address is neither owner nor spawn proxy")
}

func (c *RollerClient) GetTransferProxyType(ctx context.Context, point, signingAddress string) (string, error) {
	pointInfo, err := c.GetPoint(ctx, point)
	if err != nil {
		return "", fmt.Errorf("get point info: %w", err)
	}
	if strings.EqualFold(pointInfo.Ownership.Owner.Address, signingAddress) {
		return "own", nil
	}
	if strings.EqualFold(pointInfo.Ownership.TransferProxy.Address, signingAddress) {
		return "transfer", nil
	}
	return "", fmt.Errorf("address is neither owner nor transfer proxy")
}

func (c *RollerClient) CanConfigureKeys(ctx context.Context, point, address string) (bool, error) {
	pointInfo, err := c.GetPoint(ctx, point)
	if err != nil {
		return false, fmt.Errorf("get point info: %w", err)
	}
	return strings.EqualFold(pointInfo.Ownership.Owner.Address, address) ||
		strings.EqualFold(pointInfo.Ownership.ManagementProxy.Address, address), nil
}

func (c *RollerClient) CanTransfer(ctx context.Context, point, address string) (bool, error) {
	pointInfo, err := c.GetPoint(ctx, point)
	if err != nil {
		return false, fmt.Errorf("get point info: %w", err)
	}
	return strings.EqualFold(pointInfo.Ownership.Owner.Address, address) ||
		strings.EqualFold(pointInfo.Ownership.TransferProxy.Address, address), nil
}

func (c *RollerClient) CanSpawn(ctx context.Context, point, address string) (bool, error) {
	pointInfo, err := c.GetPoint(ctx, point)
	if err != nil {
		return false, fmt.Errorf("get point info: %w", err)
	}
	return strings.EqualFold(pointInfo.Ownership.Owner.Address, address) ||
		strings.EqualFold(pointInfo.Ownership.SpawnProxy.Address, address), nil
}

func (c *RollerClient) GetRollerConfig(ctx context.Context) (*RollerConfig, error) {
	result, err := c.DoRequest(ctx, "getRollerConfig", struct{}{})
	if err != nil {
		return nil, fmt.Errorf("get roller config: %w", err)
	}
	var config RollerConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("unmarshal roller config: %w", err)
	}
	return &config, nil
}

func (c *RollerClient) GetAllPending(ctx context.Context) ([]PendingTx, error) {
	result, err := c.DoRequest(ctx, "getAllPending", struct{}{})
	if err != nil {
		return nil, fmt.Errorf("get all pending: %w", err)
	}
	var pending []PendingTx
	if err := json.Unmarshal(result, &pending); err != nil {
		fmt.Printf("Raw JSON: %s\n", string(result))
		return nil, fmt.Errorf("unmarshal pending transactions: %w", err)
	}

	return pending, nil
}

func (c *RollerClient) GetPendingByAddress(ctx context.Context, address string) ([]PendingTx, error) {
	params := struct {
		Address string `json:"address"`
	}{
		Address: address,
	}
	result, err := c.DoRequest(ctx, "getPendingByAddress", params)
	if err != nil {
		return nil, fmt.Errorf("get pending by address: %w", err)
	}

	var pending []PendingTx
	if err := json.Unmarshal(result, &pending); err != nil {
		return nil, fmt.Errorf("unmarshal pending transactions: %w", err)
	}

	return pending, nil
}

func (c *RollerClient) WhenNextBatch(ctx context.Context) (*BatchInfo, error) {
	result, err := c.DoRequest(ctx, "whenNextBatch", struct{}{})
	if err != nil {
		return nil, fmt.Errorf("get next batch info: %w", err)
	}
	var batchInfo BatchInfo
	if err := json.Unmarshal(result, &batchInfo); err != nil {
		return nil, fmt.Errorf("unmarshal batch info: %w", err)
	}
	return &batchInfo, nil
}

// ships owned by an address
func (c *RollerClient) GetShips(ctx context.Context, address string) ([]ShipInfo, error) {
	params := struct {
		Address string `json:"address"`
	}{
		Address: address,
	}
	result, err := c.DoRequest(ctx, "getShips", params)
	if err != nil {
		return nil, fmt.Errorf("get ships: %w", err)
	}
	var ships []ShipInfo
	if err := json.Unmarshal(result, &ships); err != nil {
		return nil, fmt.Errorf("unmarshal ships: %w", err)
	}
	return ships, nil
}

func (c *RollerClient) GetUnspawned(ctx context.Context, point string) ([]ShipInfo, error) {
	params := struct {
		Ship string `json:"ship"`
	}{
		Ship: point,
	}
	result, err := c.DoRequest(ctx, "getUnspawned", params)
	if err != nil {
		return nil, fmt.Errorf("get unspawned: %w", err)
	}
	var unspawned []ShipInfo
	if err := json.Unmarshal(result, &unspawned); err != nil {
		return nil, fmt.Errorf("unmarshal unspawned ships: %w", err)
	}
	return unspawned, nil
}

func (c *RollerClient) SelectDataSource(ctx context.Context, useRoller, useAzimuth bool) (string, error) {
	if useRoller {
		_, err := c.GetRollerConfig(ctx)
		if err != nil {
			return "", fmt.Errorf("roller not available: %w", err)
		}
		return "roller", nil
	}
	if useAzimuth {
		return "azimuth", nil
	}
	_, err := c.GetRollerConfig(ctx)
	if err != nil {
		return "azimuth", nil
	}
	return "roller", nil
}

func ValidateAddress(address string, strip bool) (string, error) {
	if !strings.HasPrefix(address, "0x") {
		return "", &ValidationError{"address", "must start with 0x"}
	}
	cleanAddr := strings.TrimPrefix(address, "0x")
	if len(cleanAddr) != 40 {
		return "", &ValidationError{"address", "must be 40 characters long (excluding 0x)"}
	}
	dst := make([]byte, hex.DecodedLen(len(cleanAddr)))
	if _, err := hex.Decode(dst, []byte(cleanAddr)); err != nil {
		return "", &ValidationError{"address", "contains invalid hex characters"}
	}
	if strip {
		return cleanAddr, nil
	}
	return address, nil
}

func ValidatePoint(point interface{}, strip bool) (*big.Int, error) {
	switch v := point.(type) {
	case string:
		if !regexp.MustCompile(`\d`).MatchString(v) {
			if strings.HasPrefix(v, "~") {
				if co.IsValidPatp(v) {
					return co.Patp2Point(v)
				}
			}
			if strings.HasPrefix(v, "0x") {
				n, ok := new(big.Int).SetString(strings.TrimPrefix(v, "0x"), 16)
				if !ok {
					return nil, &ValidationError{"point", "invalid hex number"}
				}
				return n, nil
			}
			if !strings.HasPrefix(v, "~") {
				if co.IsValidPatp(fmt.Sprintf("~%s", v)) {
					return co.Patp2Point(fmt.Sprintf("~%s", v))
				}
			}
			return nil, &ValidationError{"point", "invalid point string format"}
		} else {
			n, ok := new(big.Int).SetString(v, 10)
			if !ok {
				return nil, &ValidationError{"point", "invalid decimal number"}
			}
			return n, nil
		}
	case int:
		if v < 0 {
			return nil, &ValidationError{"point", "cannot be negative"}
		} else if v > 4294967295 {
			return nil, &ValidationError{"point", "must be less than 2^32"}
		}
		return big.NewInt(int64(v)), nil
	case *big.Int:
		maxUint32 := new(big.Int).SetUint64(4294967295)
		if v.Sign() < 0 {
			return nil, &ValidationError{"point", "cannot be negative"}
		} else if v.Cmp(maxUint32) > 0 {
			return nil, &ValidationError{"point", "must be less than 2^32"}
		}
		return v, nil
	default:
		return nil, &ValidationError{"point", "must be string, number, or *big.Int"}
	}
}

func (c *RollerClient) GetPoint(ctx context.Context, point interface{}) (*Point, error) {
	pointNum, err := ValidatePoint(point, false)
	if err != nil {
		return nil, fmt.Errorf("validate point: %w", err)
	}
	patp, err := co.Point2Patp(pointNum)
	if err != nil {
		return nil, fmt.Errorf("convert to patp: %w", err)
	}
	params := struct {
		Ship string `json:"ship"`
	}{
		Ship: patp,
	}
	result, err := c.DoRequest(ctx, "getPoint", params)
	if err != nil {
		return nil, fmt.Errorf("get point: %w", err)
	}
	var pointInfo Point
	if err := json.Unmarshal(result, &pointInfo); err != nil {
		return nil, fmt.Errorf("unmarshal point info: %w", err)
	}
	return &pointInfo, nil
}

func (c *RollerClient) Spawn(ctx context.Context, parentPoint, spawnPoint interface{}, newOwnerAddress, signingAddress string, privateKey *ecdsa.PrivateKey) (*Transaction, error) {
	parentNum, err := ValidatePoint(parentPoint, false)
	if err != nil {
		return nil, fmt.Errorf("validate parent point: %w", err)
	}
	parentPatp, err := co.Point2Patp(parentNum)
	if err != nil {
		return nil, fmt.Errorf("convert parent to patp: %w", err)
	}
	spawnNum, err := ValidatePoint(spawnPoint, false)
	if err != nil {
		return nil, fmt.Errorf("validate spawn point: %w", err)
	}
	spawnPatp, err := co.Point2Patp(spawnNum)
	if err != nil {
		return nil, fmt.Errorf("convert spawn to patp: %w", err)
	}
	newOwnerAddr, err := ValidateAddress(newOwnerAddress, false)
	if err != nil {
		return nil, fmt.Errorf("validate new owner address: %w", err)
	}
	signingAddr, err := ValidateAddress(signingAddress, false)
	if err != nil {
		return nil, fmt.Errorf("validate signing address: %w", err)
	}
	proxyType, err := c.GetSpawnProxyType(ctx, parentPatp, signingAddr)
	if err != nil {
		return nil, fmt.Errorf("get spawn proxy type: %w", err)
	}
	params := SpawnRequest{
		Address: signingAddr,
		From: FromData{
			Ship:  parentPatp,
			Proxy: proxyType,
		},
		Data: struct {
			Ship    string `json:"ship"`
			Address string `json:"address"`
		}{
			Ship:    spawnPatp,
			Address: newOwnerAddr,
		},
	}
	if err := c.addSignature(ctx, "spawn", &params, privateKey); err != nil {
		return nil, fmt.Errorf("add signature: %w", err)
	}
	result, err := c.DoRequest(ctx, "spawn", params)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	var txHash string
	if err := json.Unmarshal(result, &txHash); err != nil {
		return nil, fmt.Errorf("unmarshal transaction hash: %w", err)
	}
	return &Transaction{
		Signature: params.Signature,
		Hash:      txHash,
		Type:      "spawn",
	}, nil
}

func (c *RollerClient) GetSpawned(ctx context.Context, point interface{}) ([]ShipInfo, error) {
	pointNum, err := ValidatePoint(point, false)
	if err != nil {
		return nil, fmt.Errorf("validate point: %w", err)
	}
	patp, err := co.Point2Patp(pointNum)
	if err != nil {
		return nil, fmt.Errorf("convert to patp: %w", err)
	}
	params := struct {
		Ship string `json:"ship"`
	}{
		Ship: patp,
	}
	result, err := c.DoRequest(ctx, "getSpawned", params)
	if err != nil {
		return nil, fmt.Errorf("get spawned: %w", err)
	}
	var spawned []ShipInfo
	if err := json.Unmarshal(result, &spawned); err != nil {
		return nil, fmt.Errorf("unmarshal spawned ships: %w", err)
	}
	return spawned, nil
}
