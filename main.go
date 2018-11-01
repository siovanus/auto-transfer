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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"

	"encoding/hex"
	"github.com/ontio/auto-transfer/common"
	sdk "github.com/ontio/ontology-go-sdk"
	ocommon "github.com/ontio/ontology/common"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
)

var pubkeys = []string{
	"022bf80145bd448d993abffa237f4cd06d9df13eaad37afce5cb71d80c47b03feb",
	"03c8f63775536eb420c96228cdccc9de7d80e87f1b562a6eb93c0838064350aa53",
	"02bcdd278a27e4969d48de95d6b7b086b65b8d1d4ff6509e7a9eab364a76115af7",
	"0251f06bc247b1da94ec7d9fe25f5f913cedaecba8524140353b826cf9b1cbd9f4",
	"0253719ac66d7cafa1fe49a64f73bd864a346da92d908c19577a003a8a4160b7fa",
	"02765d98bb092962734e365bd436bdc80c5b5991dcf22b28dbb02d3b3cf74d6444",
	"022e911fb5a20b4b2e4f917f10eb92f27d17cad16b916bce8fd2dd8c11ac2878c0",
}

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

	f, err := os.Create("record.txt")
	if err != nil {
		fmt.Println("Error os.Create: ", err)
		return
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	var sum uint64 = 0
	for _, record := range data {
		if inList(record.PeerPubkey, pubkeys) {
			_, ok := peerSum[record.PeerPubkey]
			if !ok {
				continue
			}
			share := new(big.Int).SetUint64(record.Value)
			bonus := new(big.Int).SetUint64(common.DefConfig.Bonus)
			total := new(big.Int).SetUint64(peerSum[record.PeerPubkey])
			amount := new(big.Int).Div(new(big.Int).Mul(share, bonus), total)
			if total.Cmp(new(big.Int)) == 0 {
				continue
			}
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
			w.WriteString(address.ToBase58() + "\t" + amount.String())
			w.WriteString("\n")
			sum += amount.Uint64()
		}
	}
	w.Flush()

	if sum > 7*common.DefConfig.Bonus {
		fmt.Println("error, sum of split is more than total bonus")
		return
	}

	n := len(sts) / 500
	for i := 0; i <= n; i++ {
		states := sts[i*500:(i+1)*500]
		if i == n {
			states = sts[i*500:]
		}
		fmt.Println(len(states))
		tx, err := ontSdk.Native.Ong.NewMultiTransferTransaction(common.DefConfig.GasPrice, common.DefConfig.GasLimit, states)
		if err != nil {
			fmt.Println("ontSdk.Native.Ong.NewMultiTransferTransaction error :", err)
			return
		}
		err = ontSdk.SignToTransaction(tx, user)
		if err != nil {
			fmt.Println("ontSdk.SignToTransaction error :", err)
			return
		}
		transaction, err := tx.IntoImmutable()
		if err != nil {
			fmt.Println("tx.IntoImmutable error :", err)
			return
		}
		f2, err := os.Create(fmt.Sprintf("transaction%d.txt", i))
		if err != nil {
			fmt.Println("ontSdk.Native.Ong.NewMultiTransferTransaction error :", err)
			return
		}
		w2 := bufio.NewWriter(f2)
		w2.WriteString(hex.EncodeToString(transaction.ToArray()))
		txHash, err := ontSdk.SendTransaction(tx)
		if err != nil {
			fmt.Println("ontSdk.SendTransaction error :", err)
			return
		}
		fmt.Println("tx success, txHash is :", txHash.ToHexString())
		f2.Close()
	}
}

func inList(item string, list []string) bool {
	result := false
	for _, i := range list {
		if item == i {
			result = true
		}
	}
	return result
}
