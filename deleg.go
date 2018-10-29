package main

import (
	"fmt"
	"os"
	"time"

	"strconv"

	"github.com/BurntSushi/toml"
	m "github.com/ValidatorCenter/minter-go-sdk"
)

var (
	conf  Config
	sdk   []m.SDK
	nodes []NodeData
)

type Config struct {
	Address   string          `toml:"address"`
	Nodes     [][]interface{} `toml:"nodes"`
	Accounts  [][]interface{} `toml:"accounts"`
	CoinNet   string          `toml:"coin_net"`
	Timeout   int             `toml:"timeout"`
	MinAmount int             `toml:"min_amount"`
}

type NodeData struct {
	PubKey string
	Prc    int
}

func cnvStr2Float_18(amntTokenStr string) float32 {
	var fAmntToken float32 = 0.0
	if amntTokenStr != "" {
		fAmntToken64, err := strconv.ParseFloat(amntTokenStr, 64)
		if err != nil {
			panic(err.Error())
		}
		fAmntToken = float32(fAmntToken64 / 1000000000000000000)
	}
	return fAmntToken
}

func getMinString(bigStr string) string {
	return fmt.Sprintf("%s...%s", bigStr[:6], bigStr[len(bigStr)-4:len(bigStr)])
}

// делегирование
func delegate() {
	for iS, _ := range sdk {
		var valueBuy map[string]string
		valueBuy = sdk[iS].GetBalance(sdk[iS].AccAddress)
		valueBuy_f32 := cnvStr2Float_18(valueBuy[conf.CoinNet])
		fmt.Println("#################################")
		fmt.Println("DELEGATE: ", valueBuy_f32)
		// 1bip на прозапас
		if valueBuy_f32 < float32(conf.MinAmount+1) {
			fmt.Printf("ERROR: Less than %d%s+1\n", conf.MinAmount, conf.CoinNet)
			continue // переходим к другой учетной записи
		}
		fullDelegCoin := float64(valueBuy_f32 - 1.0) // 1MNT на комиссию

		// Цикл делегирования
		for i, _ := range nodes {

			amnt_f64 := fullDelegCoin * float64(nodes[i].Prc) / 100 // в процентном соотношение

			delegDt := m.TxDelegateData{
				Coin:     conf.CoinNet,
				PubKey:   nodes[i].PubKey,
				Stake:    int64(amnt_f64),
				GasCoin:  conf.CoinNet,
				GasPrice: 1,
			}

			fmt.Println("TX: ", getMinString(sdk[iS].AccAddress), fmt.Sprintf("%d%%", nodes[i].Prc), "=>", getMinString(nodes[i].PubKey), "=", int64(amnt_f64), conf.CoinNet)

			resHash, err := sdk[iS].TxDelegate(&delegDt)
			if err != nil {
				fmt.Println("ERROR:", err.Error())
			} else {
				fmt.Println("HASH TX:", resHash)
			}
		}
	}
}

func main() {
	ConfFileName := "adlg.toml"

	// проверяем есть ли входной параметр/аргумент
	if len(os.Args) == 2 {
		ConfFileName = os.Args[1]
	}
	fmt.Printf("TOML=%s\n", ConfFileName)

	if _, err := toml.DecodeFile(ConfFileName, &conf); err != nil {
		fmt.Println("ERROR: loading toml file:", err.Error())
		return
	} else {
		fmt.Println("...data from toml file = loaded!")
	}

	for _, d := range conf.Accounts {
		str0 := ""
		str1 := ""
		ok := true

		if str0, ok = d[0].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[0], "not wallet address")
			return
		}
		if str1, ok = d[1].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[1], "not private wallet key")
			return
		}

		sdk1 := m.SDK{
			MnAddress:     conf.Address,
			AccAddress:    str0,
			AccPrivateKey: str1,
		}
		sdk = append(sdk, sdk1)
	}

	for _, d := range conf.Nodes {
		str0 := ""
		str1 := ""
		ok := true

		if str0, ok = d[0].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[0], "not a masternode public key")
			return
		}
		if str1, ok = d[1].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[1], "not a number")
			return
		}

		int1, err := strconv.Atoi(str1)
		if err != nil {
			fmt.Println("ERROR: loading toml file:", str1, "not a number")
			return
		}

		n1 := NodeData{
			PubKey: str0,
			Prc:    int1,
		}
		nodes = append(nodes, n1)
	}

	for { // бесконечный цикл
		delegate()
		fmt.Printf("Pause %dmin .... at this moment it is better to interrupt\n", conf.Timeout)
		time.Sleep(time.Minute * time.Duration(conf.Timeout)) // пауза ~TimeOut~ мин
	}
}
