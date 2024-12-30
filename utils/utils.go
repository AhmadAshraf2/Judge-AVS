package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/BitDSM/BitDSM-Node/btcComms"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcutil/psbt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
)

func InitConfigFile() {
	viper.AddConfigPath("./configs")
	viper.SetConfigName("config") // Register config file name (no extension)
	viper.SetConfigType("json")   // Look for specific type
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Error reading config file: ", err)
	}
}

func getBitcoinRpcClient() *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         viper.GetString("btc_node_host"),
		User:         viper.GetString("btc_node_user"),
		Pass:         viper.GetString("btc_node_pass"),
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		fmt.Println("Failed to connect to the Bitcoin client : ", err)
	}

	return client
}

func getBitcoinRpcClientwithWallet(walletName string) *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         fmt.Sprintf("%s/wallet/%s", viper.GetString("btc_node_host"), walletName),
		User:         viper.GetString("btc_node_user"),
		Pass:         viper.GetString("btc_node_pass"),
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		fmt.Println("Failed to connect to the Bitcoin client: ", err)
	}

	return client
}

func LoadBtcWallet(walletName string) {
	client := getBitcoinRpcClient()
	defer client.Shutdown()
	_, err := client.LoadWallet(walletName)
	if err != nil {
		fmt.Println("Failed to load wallet : ", err)
	}
}

func CreateTxFromHex(txHex string) (*wire.MsgTx, error) {
	// Decode the transaction hex string
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %v", err)
	}

	// Create a new transaction object
	tx := wire.NewMsgTx(wire.TxVersion)

	// Deserialize the transaction bytes
	err = tx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	return tx, nil
}

func GetRawTransaction(txid string) (*btcjson.TxRawResult, error) {

	client := getBitcoinRpcClient()
	defer client.Shutdown()

	// Convert txid to chainhash.Hash
	txHash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		fmt.Println("Failed to create tx hash: ", err)
		return nil, err
	}
	tx, err := client.GetRawTransactionVerbose(txHash)
	if err != nil {
		fmt.Println("Failed to get raw transaction: ", err)
		return nil, err
	}
	return tx, nil
}

func GetFeeFromBtcNode(tx *wire.MsgTx) (int64, error) {
	walletName := viper.GetString("wallet_name")
	feeRateAdjustment := viper.GetInt64("fee_rate_adjustment")
	result, err := btcComms.GetEstimateFee(walletName)
	if err != nil {
		fmt.Println("Error getting fee rate : ", err)
		return 0, err
	}

	feeRateInBtc := result.Result.Feerate

	fmt.Printf("Estimated fee per kilobyte for a transaction to be confirmed within 2 blocks: %f BTC\n", feeRateInBtc)
	feeRate := BtcToSats(feeRateInBtc) + feeRateAdjustment
	fmt.Printf("Estimated fee per kilobyte for a transaction to be confirmed within 2 blocks: %d Sats\n", feeRate)
	baseSize := tx.SerializeSizeStripped()
	totalSize := tx.SerializeSize()
	weight := (baseSize * 3) + totalSize
	vsize := (weight + 3) / 4
	fmt.Println("tx size in bytes : ", vsize)
	fee := float64(vsize) * float64(feeRate) / float64(1024)
	fmt.Println("fee for this tx : ", fee)
	minFee, err := GetMinRelayTxFee()
	if err != nil {
		return int64(fee), nil
	}
	minFeeInSats := BtcToSats(minFee)
	if fee < float64(minFeeInSats) {
		fmt.Println("Fee is less than min relay fee. Setting fee to min relay fee")
		fee = float64(minFeeInSats + 100)
	}
	return int64(fee), nil
}

func BtcToSats(btc float64) int64 {
	return int64(btc * 1e8)
}

func SatsToBtc(sats int64) float64 {
	return float64(sats) / 100000000.0
}

func IsValidBtcPubKey(pubKeyStr string) bool {
	// Decode the hex string into bytes
	pubKeyBytes, err := hex.DecodeString(pubKeyStr)
	if err != nil {
		return false
	}

	// Parse the public key using btcec
	_, err = btcec.ParsePubKey(pubKeyBytes, btcec.S256())
	return err == nil
}

func IsValidEthAddress(address string) bool {
	return common.IsHexAddress(address)
}

func IsValidPsbt(psbtStr string) bool {
	// Decode the base58 string into bytes
	psbtBytes := base58.Decode(psbtStr)
	if len(psbtBytes) == 0 {
		return false
	}

	// Convert psbtBytes to an io.Reader
	psbtReader := bytes.NewReader(psbtBytes)

	// Parse the PSBT bytes
	_, err := psbt.NewFromRawBytes(psbtReader, false)
	return err == nil
}

func IsValidBtcAddress(address string) bool {
	_, err := btcutil.DecodeAddress(address, &chaincfg.MainNetParams)
	if err == nil {
		return true
	}

	_, err = btcutil.DecodeAddress(address, &chaincfg.TestNet3Params)
	if err == nil {
		return true
	}

	_, err = btcutil.DecodeAddress(address, &chaincfg.SigNetParams)
	if err == nil {
		return true
	}

	_, err = btcutil.DecodeAddress(address, &chaincfg.RegressionNetParams)
	if err == nil {
		return true
	}
	return false
}
func ListUnspentBtcUtxos(address string) ([]btcjson.ListUnspentResult, error) {
	walletName := viper.GetString("wallet_name")
	client := getBitcoinRpcClientwithWallet(walletName)
	defer client.Shutdown()

	addr, err := btcutil.DecodeAddress(address, &chaincfg.RegressionNetParams)
	if err != nil {
		log.Fatalf("Invalid Bitcoin address: %v", err)
	}

	unspent, err := client.ListUnspentMinMaxAddresses(1, 9999999, []btcutil.Address{addr})
	if err != nil {
		fmt.Println("Error listing unspent outputs: ", err)
		return nil, err
	}
	return unspent, nil
}

func GetMinRelayTxFee() (float64, error) {
	client := getBitcoinRpcClient()
	defer client.Shutdown()

	// Get mempool info
	mempoolInfo, err := client.GetNetworkInfo()
	if err != nil {
		return 0, err
	}
	return mempoolInfo.RelayFee, nil
}

func HexToBech32(hexAddr string, network *chaincfg.Params) (string, error) {
	// Decode hex string to bytes
	decoded, err := hex.DecodeString(hexAddr)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex string: %v", err)
	}

	// Create appropriate address based on the length of the decoded bytes
	var addr btcutil.Address
	switch len(decoded) {
	case 20: // P2WPKH (20 bytes)
		addr, err = btcutil.NewAddressWitnessPubKeyHash(decoded, network)
	case 32: // P2WSH (32 bytes)
		addr, err = btcutil.NewAddressWitnessScriptHash(decoded, network)
	default:
		return "", fmt.Errorf("invalid decoded length: got %d, want 20 or 32", len(decoded))
	}

	if err != nil {
		return "", fmt.Errorf("failed to create bech32 address: %v", err)
	}

	return addr.String(), nil
}

func Base64ToHex(base64String string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func HexToBase64(hexString string) (string, error) {
	data, err := hex.DecodeString(hexString)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func CleanXpubKey(input string) string {
	// Find the position of "tpub"
	pubIndex := strings.Index(input, "tpub")
	if pubIndex == -1 {
		pubIndex = strings.Index(input, "xpub")
		if pubIndex == -1 {
			return input // Return original if neither "tpub" nor "xpub" is found
		}
	}

	// Extract from "tpub" to the end
	cleaned := input[pubIndex:]

	// Remove any trailing derivation path (e.g., /0/0)
	if slashIndex := strings.Index(cleaned, "/"); slashIndex != -1 {
		cleaned = cleaned[:slashIndex]
	}

	return cleaned
}

func DerivePublicKey(xpub string, index uint32) (string, error) {
	xpub = CleanXpubKey(xpub)
	masterKey, err := hdkeychain.NewKeyFromString(xpub)
	if err != nil {
		fmt.Println("failed to parse tpub: ", err)
		return "", err
	}
	child0, err := masterKey.Derive(0)
	if err != nil {
		fmt.Println("failed to derive first child: ", err)
		return "", err
	}
	childIndex, err := child0.Derive(index)
	if err != nil {
		fmt.Println("failed to derive second child: ", err)
		return "", err
	}
	pubKey, err := childIndex.ECPubKey()
	if err != nil {
		fmt.Println("failed to get public key: ", err)
		return "", err
	}
	pubKeyBytes := pubKey.SerializeCompressed()
	pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)

	return pubKeyHex, nil
}

func hexToScript(hexStr string) (string, error) {
	// Decode hex string to bytes
	scriptBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex: %v", err)
	}

	// Disassemble the script using btcsuite
	disbuf, err := txscript.DisasmString(scriptBytes)
	if err != nil {
		return "", fmt.Errorf("failed to disassemble script: %v", err)
	}

	return disbuf, nil
}
