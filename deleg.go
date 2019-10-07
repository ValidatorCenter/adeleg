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

const tagVersion = "Vc"

var (
	version string
	conf    Config
	sdk     []m.SDK
	nodes   []NodeData
	accs    []AccData
	mina    []MinAmntData
)

type Config struct {
	Address   string          `toml:"address"`
	Nodes     [][]interface{} `toml:"nodes"`
	Accounts  [][]interface{} `toml:"accounts"`
	CoinNet   string          `toml:"-"`
	Timeout   int             `toml:"timeout"`
	MinAmount [][]interface{} `toml:"min_amount"`
	ChainNet  string          `toml:"chain"`
	MaxGas    int             `toml:"max_gas"`
}

type NodeData struct {
	PubKey string
	Prc    int
	Coin   string
	Rule   string
}

type AccData struct {
	Rule string
	Mntr m.SDK
}

type MinAmntData struct {
	Rule string
	Amnt int
}

func getMinString(bigStr string) string {
	return fmt.Sprintf("%s...%s", bigStr[:6], bigStr[len(bigStr)-4:len(bigStr)])
}

// делегирование
func delegate() {
	var err error
	for iS, _ := range accs {
		var valueBuy map[string]float32
		valueBuy, _, err = accs[iS].Mntr.GetAddress(accs[iS].Mntr.AccAddress)
		if err != nil {
			fmt.Println("ERROR:", err.Error())
			continue
		}

		minAmnt := 0
		for iM, _ := range mina {
			if accs[iS].Rule == mina[iM].Rule {
				minAmnt = mina[iM].Amnt
			}
		}

		valueBuy_f32 := valueBuy[conf.CoinNet]
		fmt.Println("#################################")
		fmt.Println("DELEGATE: ", valueBuy_f32)

		// 1bip на прозапас
		if valueBuy_f32 < float32(minAmnt+1) {
			fmt.Printf("ERROR: Less than %d%s+1\n", minAmnt, conf.CoinNet)
			continue // переходим к другой учетной записи
		}
		fullDelegCoin := float64(valueBuy_f32 - 1.0) // 1MNT на комиссию

		// Цикл делегирования
		for i, _ := range nodes {
			if accs[iS].Rule == nodes[i].Rule {
				if nodes[i].Coin == "" || nodes[i].Coin == conf.CoinNet {
					// Страндартная монета BIP(MNT)
					amnt_f64 := fullDelegCoin * float64(nodes[i].Prc) / 100 // в процентном соотношение

					Gas, _ := accs[iS].Mntr.GetMinGas()
					if Gas > int64(conf.MaxGas) {
						// Если комиссия дофига, то ничего делать не будем
						fmt.Println("Comission GAS >", conf.MaxGas)
						continue
					}

					delegDt := m.TxDelegateData{
						Coin:     conf.CoinNet,
						PubKey:   nodes[i].PubKey,
						Stake:    float32(amnt_f64),
						Payload:  tagVersion,
						GasCoin:  conf.CoinNet,
						GasPrice: Gas,
					}

					fmt.Println("TX: ", getMinString(accs[iS].Mntr.AccAddress), fmt.Sprintf("%d%%", nodes[i].Prc), "=>", getMinString(nodes[i].PubKey), "=", int64(amnt_f64), conf.CoinNet)

					resHash, err := accs[iS].Mntr.TxDelegate(&delegDt)
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

					Gas, _ := accs[iS].Mntr.GetMinGas()
					if Gas > int64(conf.MaxGas) {
						// Если комиссия дофига, то ничего делать не будем
						fmt.Println("Comission GAS >", conf.MaxGas)
						continue
					}

					sellDt := m.TxSellCoinData{
						CoinToBuy:   nodes[i].Coin,
						CoinToSell:  conf.CoinNet,
						ValueToSell: float32(amnt_i64),
						Payload:     tagVersion,
						GasCoin:     conf.CoinNet,
						GasPrice:    Gas,
					}
					fmt.Println("TX: ", getMinString(accs[iS].Mntr.AccAddress), fmt.Sprintf("%d%s", int64(amnt_f64), conf.CoinNet), "=>", nodes[i].Coin)
					resHash, err := accs[iS].Mntr.TxSellCoin(&sellDt)
					if err != nil {
						fmt.Println("ERROR:", err.Error())
						continue // переходим к другой записи мастернод
					} else {
						fmt.Println("HASH TX:", resHash)
					}

					if nodes[i].PubKey == "" {
						// просто закупка монеты кастомной
						continue // переходим к другой записи мастернод
					}

					// SLEEP!
					time.Sleep(time.Second * 10) // пауза 10сек, Nonce чтобы в блокчейна +1

					var valDeleg2 map[string]float32
					valDeleg2, _, err = accs[iS].Mntr.GetAddress(accs[iS].Mntr.AccAddress)
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

					Gas, _ = accs[iS].Mntr.GetMinGas()
					if Gas > int64(conf.MaxGas) {
						// Если комиссия дофига, то ничего делать не будем
						fmt.Println("Comission GAS >", conf.MaxGas)
						continue
					}

					delegDt := m.TxDelegateData{
						Coin:     nodes[i].Coin,
						PubKey:   nodes[i].PubKey,
						Stake:    float32(valDeleg2_i64),
						Payload:  tagVersion,
						GasCoin:  conf.CoinNet,
						GasPrice: Gas,
					}

					fmt.Println("TX: ", getMinString(accs[iS].Mntr.AccAddress), fmt.Sprintf("%d%%", nodes[i].Prc), "=>", getMinString(nodes[i].PubKey), "=", valDeleg2_i64, nodes[i].Coin)

					resHash2, err := accs[iS].Mntr.TxDelegate(&delegDt)
					if err != nil {
						fmt.Println("ERROR:", err.Error())
					} else {
						fmt.Println("HASH TX:", resHash2)
					}
				}
				// SLEEP!
				time.Sleep(time.Second * 10) // пауза 10сек, Nonce чтобы в блокчейна +1
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

	MainChain := false
	conf.CoinNet = "MNT"
	if conf.ChainNet == "main" {
		MainChain = true
		conf.CoinNet = "BIP"
	}

	for _, d := range conf.Accounts {
		var err error
		acc1 := AccData{}
		priv := ""
		pblck := ""
		ok := true

		if priv, ok = d[0].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[0], "not private wallet key")
			return
		}

		if acc1.Rule, ok = d[1].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[1], "not a number")
			return
		}

		pblck, err = m.GetAddressPrivateKey(priv)
		if err != nil {
			fmt.Println("ERROR: convert PrivKey: ", err.Error())
			return
		}

		acc1.Mntr = m.SDK{
			MnAddress:     conf.Address,
			AccAddress:    pblck,
			AccPrivateKey: priv,
			ChainMainnet:  MainChain,
		}

		accs = append(accs, acc1)
	}

	for _, d := range conf.MinAmount {
		var err error
		min1 := MinAmntData{}
		str1 := ""
		ok := true

		if str1, ok = d[0].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[0], "not a number")
			return
		}

		min1.Amnt, err = strconv.Atoi(str1)
		if err != nil {
			fmt.Println("ERROR: loading toml file:", str1, "not a number")
			return
		}

		if min1.Rule, ok = d[1].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[1], "not a rule")
			return
		}

		mina = append(mina, min1)
	}

	for _, d := range conf.Nodes {
		rul := ""
		pubN := ""
		prcInt := ""
		coinX := ""
		ok := true

		if len(d) == 4 {
			if coinX, ok = d[3].(string); !ok {
				fmt.Println("ERROR: loading toml file:", d[3], "not a coin")
				return
			}
			coinX = strings.ToUpper(coinX)
		}

		if rul, ok = d[0].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[0], "not a rule")
			return
		}

		if pubN, ok = d[1].(string); !ok {
			if coinX == "" {
				// нет пабликея и это не кастомная монета, значит - ошибка
				fmt.Println("ERROR: loading toml file:", d[1], "not a masternode public key")
				return
			}
		}

		if prcInt, ok = d[2].(string); !ok {
			fmt.Println("ERROR: loading toml file:", d[2], "not a number")
			return
		}

		int1, err := strconv.Atoi(prcInt)
		if err != nil {
			fmt.Println("ERROR: loading toml file:", prcInt, "not a number")
			return
		}

		n1 := NodeData{
			Rule:   rul,
			PubKey: pubN,
			Prc:    int1,
			Coin:   coinX,
		}
		nodes = append(nodes, n1)
	}

	for { // бесконечный цикл
		delegate()
		fmt.Printf("Pause %dmin .... at this moment it is better to interrupt\n", conf.Timeout)
		time.Sleep(time.Minute * time.Duration(conf.Timeout)) // пауза ~TimeOut~ мин
	}
}
