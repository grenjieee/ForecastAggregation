package chain

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const usdcDecimals = 6

// Escrow releaseFunds 最小 ABI
const escrowReleaseFundsABI = `[
	{"name":"releaseFunds","type":"function","inputs":[
		{"name":"betId","type":"bytes32"},
		{"name":"to","type":"address"},
		{"name":"amount","type":"uint256"}
	],"outputs":[]}
]`

// ReleaseFunds 调用 Escrow.releaseFunds(betId, to, amount)。Executor 私钥对应地址需在 Escrow 上具备 EXECUTOR_ROLE；Gas 由该账户支付。
// betIdHex 为 contract_order_id 十六进制（可带或不带 0x 前缀，不足 32 字节时左补零）。
func ReleaseFunds(ctx context.Context, rpcURL, escrowAddr, executorPrivateKeyHex string, betIdHex string, toAddr common.Address, amount *big.Int) (txHash string, err error) {
	if rpcURL == "" || escrowAddr == "" || executorPrivateKeyHex == "" {
		return "", fmt.Errorf("rpc_url, escrow_address, executor_private_key 必填")
	}
	if amount == nil || amount.Sign() <= 0 {
		return "", fmt.Errorf("amount 必须大于 0")
	}

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return "", fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	hexStr := strings.TrimPrefix(strings.TrimSpace(betIdHex), "0x")
	for _, c := range hexStr {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return "", fmt.Errorf("contract_order_id 含有非十六进制字符，请使用入金后获得的完整 64 位 hex（勿截断或混入其它字符）")
	}
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	buf, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("decode betId hex: %w", err)
	}
	if len(buf) > 32 {
		return "", fmt.Errorf("betId 超过 32 字节")
	}
	// contract_order_id 必须与 lockFunds 时使用的 betId 完全一致（通常为 64 位十六进制）。不足 32 字节时左补零会得到不同的 bytes32，导致 lockedAmount[betId]=0、解冻 revert
	if len(hexStr) != 64 {
		return "", fmt.Errorf("contract_order_id 须为 64 位十六进制（与入金 lockFunds 的 betId 一致），当前 %d 位会导致链上 betId 不匹配、lockedAmount 为 0 且解冻失败", len(hexStr))
	}
	var betId [32]byte
	copy(betId[32-len(buf):], buf)

	parsed, err := abi.JSON(strings.NewReader(escrowReleaseFundsABI))
	if err != nil {
		return "", err
	}
	data, err := parsed.Pack("releaseFunds", betId, toAddr, amount)
	if err != nil {
		return "", fmt.Errorf("pack releaseFunds: %w", err)
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

	toAddrContract := common.HexToAddress(escrowAddr)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonceU64,
		GasPrice: gasPrice,
		Gas:      150000,
		To:       &toAddrContract,
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
	txHashHex := signed.Hash().Hex()
	// 等待交易上链并确认是否执行成功，避免链上 revert 但后端仍标记为已解冻
	for i := 0; i < 30; i++ {
		receipt, err := client.TransactionReceipt(ctx, signed.Hash())
		if err != nil {
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("等待交易确认: %w", ctx.Err())
			case <-time.After(2 * time.Second):
				continue
			}
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return "", fmt.Errorf("解冻交易已上链但执行失败(revert)，请检查 contract_order_id 是否为完整 64 位 hex、Executor 是否有 EXECUTOR_ROLE、该 betId 是否仍有锁定金额，tx: %s", txHashHex)
		}
		return txHashHex, nil
	}
	return "", fmt.Errorf("等待交易确认超时，请稍后在区块浏览器查看 tx: %s", txHashHex)
}

// FloatToUSDCAmount 将 USDC 金额（如 10.5）转为链上 6 位精度 *big.Int
func FloatToUSDCAmount(amount float64) *big.Int {
	if amount <= 0 {
		return big.NewInt(0)
	}
	// 简单做法：先乘 1e6 再取整
	div := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(usdcDecimals), nil))
	a := new(big.Float).SetFloat64(amount)
	a.Mul(a, div)
	i, _ := a.Int(nil)
	return i
}
