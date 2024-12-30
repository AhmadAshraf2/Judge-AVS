package btcComms

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/psbt"
	"github.com/spf13/viper"
)

func SendRPC(method string, data []interface{}, wallet string, signer bool) ([]byte, error) {
	host := viper.GetString("btc_node_host")
	user := viper.GetString("btc_node_user")
	pass := viper.GetString("btc_node_pass")
	protocol := viper.GetString("btc_node_protocol")

	request := JSONRPCRequest{
		ID:      1,
		Method:  method,
		Params:  data,
		Jsonrpc: "1.0",
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error creating JSON: ", err)
		return nil, err
	}

	// Create a HTTP client
	client := &http.Client{}

	// Create a HTTP request
	host = protocol + host + "/wallet/" + wallet
	req, err := http.NewRequest("POST", host, bytes.NewBuffer(requestJSON))
	if err != nil {
		fmt.Println("Error creating request: ", err)
		return nil, err
	}

	// Set the content type to application/json
	req.Header.Set("Content-Type", "application/json")

	// Set the basic authentication header
	req.SetBasicAuth(user, pass)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response: ", err)
		return nil, err
	}
	return body, nil
}

func SendRPCOfflineWallet(method string, data []interface{}, wallet string, signer bool) ([]byte, error) {
	host := viper.GetString("multisig_btc_node")
	user := viper.GetString("multisig_btc_user")
	pass := viper.GetString("multisig_btc_pass")
	protocol := viper.GetString("multisig_btc_protocol")

	request := JSONRPCRequest{
		ID:      1,
		Method:  method,
		Params:  data,
		Jsonrpc: "1.0",
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error creating JSON: ", err)
		return nil, err
	}

	// Create a HTTP client
	client := &http.Client{}

	// Create a HTTP request
	host = protocol + host + "/wallet/" + wallet
	req, err := http.NewRequest("POST", host, bytes.NewBuffer(requestJSON))
	if err != nil {
		fmt.Println("Error creating request: ", err)
		return nil, err
	}

	// Set the content type to application/json
	req.Header.Set("Content-Type", "application/json")

	// Set the basic authentication header
	req.SetBasicAuth(user, pass)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response: ", err)
		return nil, err
	}
	return body, nil
}

func GetDescriptorInfo(dataStr string, wallet string) (DescriptorInfo, error) {

	var response JSONRPCResponseDesc
	data := []interface{}{dataStr}
	result, err := SendRPC("getdescriptorinfo", data, wallet, false)
	if err != nil {
		fmt.Println("error getting descriptor info : ", err)
		return response.Result, err
	}

	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return response.Result, err
	}
	if response.Error != nil {
		fmt.Println("Error in getdescriptorinfo: ", response.Error)
		return DescriptorInfo{}, errors.New("error in getdescriptorinfo")
	}

	return response.Result, nil
}

func ImportDescriptor(desc string, wallet string) error {

	descData := []ImportDescriptorType{
		{
			Desc:      desc,
			Internal:  false,
			Timestamp: "now",
		},
	}
	data := []interface{}{descData}

	_, err := SendRPC("importdescriptors", data, wallet, false)
	if err != nil {
		fmt.Println("error importing descriptor	: ", err)
	}

	return nil
}

func DeriveAddress(wallet string, descriptor string) (string, error) {
	data := []interface{}{descriptor}
	result, _ := SendRPC("deriveaddresses", data, wallet, false)
	fmt.Println("result Derive Address: ", string(result))
	var response JSONRPCArrayResponse
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}
	if response.Error != nil {
		return "", errors.New("error in get new address")
	}
	return response.Result[0], nil
}

func GetNewAddress(wallet string) (string, error) {
	result, _ := SendRPC("getnewaddress", nil, wallet, false)
	fmt.Println("result New Address: ", string(result))
	var response JSONRPCResponse
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}
	if response.Error != nil {
		return "", errors.New("error in get new address")
	}
	return response.Result, nil
}

func DecodePsbt(psbt string, wallet string) (PSBT, error) {
	data := []interface{}{psbt}
	result, _ := SendRPC("decodepsbt", data, wallet, false)
	var response JSONRPCResponsePsbt
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return PSBT{}, err
	}
	if response.Error != nil {
		return PSBT{}, errors.New("error in Decode")
	}
	return response.Result, nil
}

func CreatePsbt(inputs []TxInput, outputs []TxOutput, locktime uint32, wallet string) (string, error) {
	feeRate := make(map[string]float64)
	feeRate["feeRate"] = 0
	data := []interface{}{inputs, outputs, locktime, feeRate}
	result, _ := SendRPC("walletcreatefundedpsbt", data, wallet, false)
	fmt.Println("result Create Psbt: ", string(result))
	var response RPCResponseCreatePsbt
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}
	if response.Error != nil {
		return "", errors.New("error in CreatePSBT")
	}
	return response.Result.Psbt, nil
}

func CreateRawTx(inputs []TxInput, outputs []TxOutput, locktime uint32, wallet string) (string, error) {
	data := []interface{}{inputs, outputs, locktime}
	result, _ := SendRPC("createrawtransaction", data, wallet, false)
	fmt.Println("result in CreateRawTx: ", string(result))
	var response JSONRPCResponse
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}

	if response.Error != nil {
		return "", errors.New("error in CreateRawTx")
	}

	return response.Result, nil
}

func SignPsbt(psbtStr string, wallet string, signer bool) (string, string, error) {
	data := []interface{}{psbtStr, true, "ALL|ANYONECANPAY"}

	fmt.Println("data: ", data)

	result, _ := SendRPCOfflineWallet("walletprocesspsbt", data, wallet, signer)
	fmt.Println("result Sign PSBT: ", string(result))
	var response RPCResponseCreatePsbt
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", "", err
	}
	if response.Error != nil {
		return "", "", errors.New("error in SignPsbt")
	}
	p := response.Result.Psbt
	psbt, err := DecodePsbt(p, wallet)

	if len(psbt.Inputs) <= 0 {
		return "", "", errors.New("no inputs in psbt")
	}
	var signatures []string

	for _, input := range psbt.Inputs {
		for _, v := range input.PartialSigs {
			signatures = append(signatures, v)
			break
		}
	}

	fmt.Println("psbt.Tx.Hash: ", psbt.Tx.Hash)
	return psbt.Tx.Hash, p, nil
}

func UtxoUpdatePsbt(psbtStr string, desc string, wallet string) (string, error) {
	data := []interface{}{
		psbtStr,
		[]map[string]interface{}{
			{
				"desc": desc,
			},
		},
	}
	result, _ := SendRPC("utxoupdatepsbt", data, wallet, false)
	fmt.Println("result Update Utxo PSBT: ", string(result))
	var response JSONRPCResponse
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}

	if response.Error != nil {
		return "", errors.New("error in utxoupdatepsbt")
	}

	return response.Result, nil
}

func CreatePsbtV1(utxo TxInput, outputs []TxOutput, unlockHeight uint32, scriptPubKey []byte, amount int64) (*psbt.Packet, error) {
	// Create a new PSBT
	chainParams, err := getChainParams()
	if err != nil {
		return nil, err
	}
	hash, err := chainhash.NewHashFromStr(utxo.Txid)
	if err != nil {
		log.Fatalf("Invalid hash: %v", err)
	}
	TxIn := wire.OutPoint{
		Hash:  *hash,
		Index: uint32(utxo.Vout),
	}
	TxOut := []*wire.TxOut{}
	for _, output := range outputs {
		fmt.Println("output : ", output)
		for addr, amount := range output {
			fmt.Println("amount : ", amount)
			address, err := btcutil.DecodeAddress(addr, chainParams)
			if err != nil {
				return nil, err
			}

			TxOut = append(TxOut, wire.NewTxOut(int64(amount), address.ScriptAddress()))
		}
	}

	fmt.Println("TxOut: ", TxOut)
	fmt.Println("TxIn: ", TxIn)

	packet, err := psbt.New([]*wire.OutPoint{&TxIn}, TxOut, 2, 0, []uint32{unlockHeight})
	if err != nil {
		return nil, err
	}

	packet.Inputs[0].WitnessUtxo = &wire.TxOut{
		PkScript: scriptPubKey,
		Value:    amount,
	}

	return packet, nil
}

func GetAddressInfo(address string, wallet string) (AddressInfo, error) {
	data := []interface{}{address}
	var response JSONRPCResponseAddressInfo
	result, err := SendRPC("getaddressinfo", data, wallet, false)
	if err != nil {
		fmt.Println("error getting descriptor info : ", err)
		return AddressInfo{}, err
	}

	fmt.Println("result Get Address Info: ", string(result))

	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return AddressInfo{}, err
	}
	if response.Error != nil {
		return AddressInfo{}, errors.New("error in getting address info")
	}

	fmt.Println("result Get Address Info after unmarshal: ", string(result))

	return response.Result, nil
}

func GetEstimateFee(wallet string) (EstimateFeeResponse, error) {
	data := []interface{}{3}
	var response EstimateFeeResponse
	result, err := SendRPC("estimatesmartfee", data, wallet, false)
	if err != nil {
		fmt.Println("error getting  estimate fee info : ", err)
		return EstimateFeeResponse{}, err
	}

	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return EstimateFeeResponse{}, err
	}
	if response.Error != nil {
		return EstimateFeeResponse{}, errors.New("error in GetEstimateFee")
	}
	return response, nil
}

func SendToAddress(address string, fee float64, wallet string) (TransactionInfo, error) {
	var response TransactionInfo
	for i := 0; i < 5; i++ {
		data := []interface{}{address, fee + 0.000001}
		result, err := SendRPC("sendtoaddress", data, wallet, false)
		fmt.Println("result Send to Address: ", string(result))
		if err != nil {
			fmt.Println("error creating utxo for fee : ", err)
			return TransactionInfo{}, err
		}
		if strings.Contains(string(result), "Transaction amount too small") {
			continue
		}
		err = json.Unmarshal(result, &response)
		if err != nil {
			fmt.Println("Error unmarshalling JSON: ", err)
			return TransactionInfo{}, err
		}
		if response.Error != nil {
			return TransactionInfo{}, errors.New("error in sending to address")
		}
		return response, nil
	}
	return TransactionInfo{}, errors.New("error in sending to address 5 tries")
}

func SignRawTransaction(tx string, wallet string) (string, error) {
	data := []interface{}{tx, nil, "ALL|ANYONECANPAY"}
	var response SignedTransactionInfo
	result, err := SendRPC("signrawtransactionwithwallet", data, wallet, false)
	if err != nil {
		fmt.Println("error creating utxo for fee : ", err)
		return "", err
	}

	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}
	if response.Error != nil {
		return "", errors.New("error in signing raw transaction")
	}
	return response.Result.Hex, nil
}

func getChainParams() (*chaincfg.Params, error) {
	env := viper.GetString("env")
	if env == "dev" {
		return &chaincfg.SigNetParams, nil
	}
	if env == "prod" {
		return &chaincfg.MainNetParams, nil
	}
	return nil, fmt.Errorf("Invalid environment")
}
