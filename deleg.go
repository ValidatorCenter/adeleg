package main

import (
	"fmt"
	"math"
	"os"
	"strings"
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
	Coin   string
}

func getMinString(bigStr string) string {
	return fmt.Sprintf("%s...%s", bigStr[:6], bigStr[len(bigStr)-4:len(bigStr)])
}

// делегирование
func delegate() {
	var err error
	for iS, _ := range sdk {
		var valueBuy map[string]float32
		valueBuy, _, err = sdk[iS].GetAddress(sdk[iS].AccAddress)
		if err != nil {
			fmt.Println("ERROR:", err.Error())
			continue
		}

		valueBuy_f32 := valueBuy[conf.CoinNet]
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
			if nodes[i].Coin == "" || nodes[i].Coin == conf.CoinNet {
				// Страндартная монета BIP(MNT)
				amnt_f64 := fullDelegCoin * float64(nodes[i].Prc) / 100 // в процентном соотношение

				delegDt := m.TxDelegateData{
					Coin:     conf.CoinNet,
					PubKey:   nodes[i].PubKey,
					Stake:    float32(amnt_f64),
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
			} else {
				// Кастомная
				amnt_f64 := fullDelegCoin * float64(nodes[i].Prc) / 100 // в процентном соотношение на какую сумму берём кастомных монет
				amnt_i64 := math.Floor(amnt_f64)                        // в меньшую сторону
				if amnt_i64 <= 0 {
					fmt.Println("ERROR: Value to Sell =0")
					continue // переходим к другой записи мастернод
				}

				sellDt := m.TxSellCoinData{
					CoinToBuy:   nodes[i].Coin,
					CoinToSell:  conf.CoinNet,
					ValueToSell: float32(amnt_i64),
					GasCoin:     conf.CoinNet,
					GasPrice:    1,
				}
				fmt.Println("TX: ", getMinString(sdk[iS].AccAddress), fmt.Sprintf("%d%s", int64(amnt_f64), conf.CoinNet), "=>", nodes[i].Coin)
				resHash, err := sdk[iS].TxSellCoin(&sellDt)
				if err != nil {
					fmt.Println("ERROR:", err.Error())
					continue // переходим к другой записи мастернод
				} else {
					fmt.Println("HASH TX:", resHash)
				}

				var valDeleg2 map[string]float32
				valDeleg2, _, err = sdk[iS].GetAddress(sdk[iS].AccAddress)
				if err != nil {
					fmt.Println("ERROR:", err.Error())
					continue
				}

				valDeleg2_f32 := valDeleg2[nodes[i].Coin]
				valDeleg2_i64 := math.Floor(float64(valDeleg2_f32)) // в меньшую сторону
				if valDeleg2_i64 <= 0 {
					fmt.Println("ERROR: Delegate =0")
					continue // переходим к другой записи мастернод
				}

				delegDt := m.TxDelegateData{
					Coin:     nodes[i].Coin,
					PubKey:   nodes[i].PubKey,
					Stake:    float32(valDeleg2_i64),
					GasCoin:  conf.CoinNet,
					GasPrice: 1,
				}

				fmt.Println("TX: ", getMinString(sdk[iS].AccAddress), fmt.Sprintf("%d%%", nodes[i].Prc), "=>", getMinString(nodes[i].PubKey), "=", valDeleg2_i64, nodes[i].Coin)

				resHash2, err := sdk[iS].TxDelegate(&delegDt)
				if err != nil {
					fmt.Println("ERROR:", err.Error())
				} else {
					fmt.Println("HASH TX:", resHash2)
				}
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
		coinX := ""
		ok := true

		if str0, ok = d[0].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[0], "not a masternode public key")
			return
		}
		if str1, ok = d[1].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[1], "not a number")
			return
		}

		if len(d) == 3 {
			if coinX, ok = d[2].(string); !ok {
				fmt.Println("ERROR: loading toml file:", d[2], "not a coin")
				return
			}
			coinX = strings.ToUpper(coinX)
		}

		int1, err := strconv.Atoi(str1)
		if err != nil {
			fmt.Println("ERROR: loading toml file:", str1, "not a number")
			return
		}

		n1 := NodeData{
			PubKey: str0,
			Prc:    int1,
			Coin:   coinX,
		}
		nodes = append(nodes, n1)

		//fmt.Printf("%#v\n", n1)
	}

	for { // бесконечный цикл
		delegate()
		fmt.Printf("Pause %dmin .... at this moment it is better to interrupt\n", conf.Timeout)
		time.Sleep(time.Minute * time.Duration(conf.Timeout)) // пауза ~TimeOut~ мин
	}
}
