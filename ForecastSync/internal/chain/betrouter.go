package chain

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// BetRouter 用于读 nonce、提交 executeBetIntent 的链上调用
const betRouterABI = `[
	{"name":"nonces","type":"function","inputs":[{"name":"","type":"address"}],"outputs":[{"type":"uint256"}]},
	{"name":"executeBetIntent","type":"function","inputs":[
		{"name":"intent","type":"tuple","components":[
			{"name":"user","type":"address"},
			{"name":"topicId","type":"bytes32"},
			{"name":"amount","type":"uint256"},
			{"name":"nonce","type":"uint256"},
			{"name":"deadline","type":"uint256"}
		]},
		{"name":"signature","type":"bytes"}
	],"outputs":[]}
]`

// GetNonce 从 BetRouter 读取用户当前 nonce
func GetNonce(ctx context.Context, rpcURL, betRouterAddr, userAddr string) (uint64, error) {
	if rpcURL == "" || betRouterAddr == "" || userAddr == "" {
		return 0, fmt.Errorf("rpc_url, bet_router_address, user 必填")
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return 0, fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	parsed, err := abi.JSON(strings.NewReader(betRouterABI))
	if err != nil {
		return 0, err
	}
	data, err := parsed.Pack("nonces", common.HexToAddress(userAddr))
	if err != nil {
		return 0, err
	}
	to := common.HexToAddress(betRouterAddr)
	msg := ethereum.CallMsg{To: &to, Data: data}
	res, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return 0, fmt.Errorf("call nonces: %w", err)
	}
	if len(res) < 32 {
		return 0, fmt.Errorf("nonces result length %d", len(res))
	}
	n := new(big.Int).SetBytes(res)
	return n.Uint64(), nil
}

// ComputeBetId 与合约 BetRouter.computeBetId 一致：keccak256(abi.encode(user, topicId, nonce))
func ComputeBetId(user common.Address, topicId [32]byte, nonce *big.Int) common.Hash {
	return crypto.Keccak256Hash(
		common.LeftPadBytes(user.Bytes(), 32),
		topicId[:],
		common.LeftPadBytes(nonce.Bytes(), 32),
	)
}

// SubmitExecuteBetIntent 使用 Executor 私钥发送 executeBetIntent 交易，返回 betId 十六进制（无 0x 前缀，与 listener 存库一致）
func SubmitExecuteBetIntent(ctx context.Context, rpcURL, betRouterAddr, executorPrivateKeyHex string, user common.Address, topicId [32]byte, amount, nonce, deadline *big.Int, signature []byte) (betIdHex string, err error) {
	if rpcURL == "" || betRouterAddr == "" || executorPrivateKeyHex == "" {
		return "", fmt.Errorf("rpc_url, bet_router_address, executor_private_key 必填")
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return "", fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	parsed, err := abi.JSON(strings.NewReader(betRouterABI))
	if err != nil {
		return "", err
	}
	// intent: (user, topicId, amount, nonce, deadline)
	intent := struct {
		User     common.Address
		TopicId  [32]byte
		Amount   *big.Int
		Nonce    *big.Int
		Deadline *big.Int
	}{user, topicId, amount, nonce, deadline}
	data, err := parsed.Pack("executeBetIntent", intent, signature)
	if err != nil {
		return "", fmt.Errorf("pack executeBetIntent: %w", err)
	}

	keyHex := executorPrivateKeyHex
	if len(keyHex) > 0 && keyHex[:2] == "0x" {
		keyHex = keyHex[2:]
	}
	keyBuf, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", fmt.Errorf("decode executor key: %w", err)
	}
	key, err := crypto.ToECDSA(keyBuf)
	if err != nil {
		return "", fmt.Errorf("to ecdsa: %w", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("chain id: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("gas price: %w", err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonceU64, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("pending nonce: %w", err)
	}

	toAddr := common.HexToAddress(betRouterAddr)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonceU64,
		GasPrice: gasPrice,
		Gas:      300000,
		To:       &toAddr,
		Value:    big.NewInt(0),
		Data:     data,
	})
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return "", fmt.Errorf("sign tx: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("send tx: %w", err)
	}
	betId := ComputeBetId(user, topicId, nonce)
	// 与 chain_subscribe 一致：contract_order_id 存为 hex 无 0x
	return hex.EncodeToString(betId.Bytes()), nil
}
