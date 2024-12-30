package BtcDepositConfirmer

import (
	"fmt"
	"math/big"
	"time"

	"github.com/BitDSM/BitDSM-Node/db"
	"github.com/BitDSM/BitDSM-Node/ethComms"
	"github.com/BitDSM/BitDSM-Node/utils"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/spf13/viper"
)

func CheckDeposit() {
	fmt.Println("starting check deposit process")
	chainParams, err := getChainParams()
	if err != nil {
		return
	}
	dbconn := db.InitDB()
	defer dbconn.Close()
	for {
		depositRequests := db.QueryDepositRequests(dbconn)

		for _, request := range depositRequests {
			tx, err := utils.GetRawTransaction(request.TransactionID)
			if err != nil {
				continue
			}
			if tx.Confirmations <= 2 {
				continue
			}

			transaction, err := utils.CreateTxFromHex(tx.Hex)
			if err != nil {
				fmt.Println("Failed to create tx from hex: ", err)
				continue
			}

			multisigaddresses := db.QueryMultisigAddresses(dbconn)

			for _, multiSigAddress := range multisigaddresses {
				for _, txOut := range transaction.TxOut {
					_, addresses, _, err := txscript.ExtractPkScriptAddrs(txOut.PkScript, chainParams)
					if err != nil {
						fmt.Println("Failed to extract addresses: ", err)
						continue
					}
					for _, address := range addresses {
						if multiSigAddress.Address == address.EncodeAddress() {
							fmt.Println("found my deposit requests")
							bigInt := new(big.Int)
							_, success := bigInt.SetString(request.Amount, 10)
							if !success {
								fmt.Println("Failed to convert string to big.Int")
								continue
							}
							_, err := ethComms.CallConfirmBtcDeposit(request.PodAddress, request.OperatorAddress, request.TransactionID, *bigInt)
							if err != nil {
								fmt.Println("Failed to call confirm btc deposit: ", err)
								continue
							}
							db.UpdateMultiSigAddressPod(dbconn, multiSigAddress.Address, request.PodAddress)
							db.MarkDepositRequestAsConfirmed(dbconn, request.TransactionID)
						}
					}
				}
			}

		}
		fmt.Println("sleeping for 3 minutes check deposit")
		time.Sleep(3 * time.Minute)

	}

}

func CheckWithdraw() {
	fmt.Println("starting check Withdraw process")
	dbconn := db.InitDB()
	defer dbconn.Close()
	for {
		withdrawRequests := db.QueryWithdrawRequests(dbconn)

		for _, request := range withdrawRequests {
			tx, err := utils.GetRawTransaction(request.TransactionID)
			if err != nil {
				continue
			}
			if tx.Confirmations <= 2 {
				continue
			}

			_, err = ethComms.CallConfirmBtcWithdraw(request.PodAddress, request.OperatorAddress, tx.Hex, request.WithdrawAddr)
			if err != nil {
				fmt.Println("Failed to call confirm btc withdraw: ", err)
				continue
			}

			db.MarkWithdrawRequestAsConfirmed(dbconn, request.TransactionID)

		}
		fmt.Println("sleeping for 3 minutes check withdrawal")
		time.Sleep(3 * time.Minute)

	}

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
