package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/AhmadAshraf2/Judge-AVS/comms"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/psbt"
	"github.com/cosmos/btcutil/base58"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

func getBitcoinRpcClient(walletName string) *rpcclient.Client {
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

func LoadBtcWallet(walletName string) {
	client := getBitcoinRpcClient(walletName)
	_, err := client.LoadWallet(walletName)
	if err != nil {
		fmt.Println("Failed to load wallet : ", err)
	}
}

func GenerateEthKeyPair() accounts.Account {
	// Generate a new random private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	password := viper.GetString("eth_keystore_password")
	ks := keystore.NewKeyStore("keystore", keystore.StandardScryptN, keystore.StandardScryptP)
	account, err := ks.ImportECDSA(privateKey, password)
	if err != nil {
		log.Fatalf("Failed to import private key: %v", err)
	}

	return account
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

func GetFeeFromBtcNode(tx *wire.MsgTx) (int64, error) {
	walletName := viper.GetString("wallet_name")
	feeRateAdjustment := viper.GetInt64("fee_rate_adjustment")
	result, err := comms.GetEstimateFee(walletName)
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
	fee := float64(vsize) * float64(feeRate/1024)
	fmt.Println("fee for this sweep : ", fee)
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
