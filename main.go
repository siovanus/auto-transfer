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
	"github.com/ontio/auto-transfer/common"
	sdk "github.com/ontio/ontology-go-sdk"
	"time"
)

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

	for {
		txHash, err := ontSdk.Native.Ont.Transfer(common.DefConfig.GasPrice, common.DefConfig.GasLimit, user, user.Address, 10)
		if err != nil {
			fmt.Println("transfer error:", err)
		}
		fmt.Println("txHash is:", txHash.ToHexString())
		time.Sleep(1*time.Second)
	}
}