/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"fmt"
	"bufio"
	"encoding/json"
	"io"
	"os"

	"github.com/ontio/auto-transfer/common"
	ocommon "github.com/ontio/ontology/common"
	sdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	"math/big"
)

const BONUS = 10000000000000000

type Result struct {
	PeerPubkey string `json:"peer_pubkey"`
	Address    string `json:"address"`
	Value      uint64 `json:"value"`
}

func main() {
	err := common.DefConfig.Init("./config.json")
	if err != nil {
		fmt.Println("DefConfig.Init error:", err)
		return
	}

	ontSdk := sdk.NewOntologySdk()
	ontSdk.NewRpcClient().SetAddress(common.DefConfig.JsonRpcAddress)
	user, ok := common.GetAccountByPassword(ontSdk, common.DefConfig.WalletFile)
	if !ok {
		fmt.Println("common.GetAccountByPassword error")
		return
	}

	var data []*Result
	peerSum := make(map[string]uint64)
	var sts []*ont.State
	fi, err := os.Open(common.DefConfig.DataFile)
	if err != nil {
		fmt.Println("Error os.Open: ", err)
		return
	}
	defer fi.Close()
	br := bufio.NewReader(fi)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		result := new(Result)
		err := json.Unmarshal([]byte(a), result)
		if err != nil {
			fmt.Println("json.Unmarshal error")
			return
		}
		if result.Value != 0 {
			_, ok := peerSum[result.PeerPubkey]
			if ok {
				peerSum[result.PeerPubkey] = peerSum[result.PeerPubkey] + result.Value
			} else {
				peerSum[result.PeerPubkey] = result.Value
			}
			data = append(data, result)
		}
	}

	var sum uint64 = 0
	for _, record := range data {
		share := new(big.Int).SetUint64(record.Value)
		bonus := new(big.Int).SetUint64(uint64(BONUS))
		total := new(big.Int).SetUint64(peerSum[record.PeerPubkey])
		amount := new(big.Int).Div(new(big.Int).Mul(share, bonus),total)
		address, err := ocommon.AddressFromBase58(record.Address)
		if err != nil {
			fmt.Println("ocommon.AddressFromBase58 error")
			return
		}
		sts = append(sts, &ont.State{
			From:  user.Address,
			To:    address,
			Value: amount.Uint64(),
		})
		sum += amount.Uint64()
	}
	if sum > BONUS {
		fmt.Println("error, sum of split is more than total bonus")
		return
	}
	txHash, err := ontSdk.Native.Ong.MultiTransfer(common.DefConfig.GasPrice, common.DefConfig.GasLimit, sts, user)
	if err != nil {
		fmt.Println("invokeNativeContract error :", err)
		return
	}
	fmt.Println("tx success, txHash is :", txHash.ToHexString())
}
