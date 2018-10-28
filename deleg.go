package main

import (
	/*"bytes"
	"encoding/json"
	*/

	"fmt"
	"os"
	"time"

	/*"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"encoding/hex"

	tr "github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"

	"gopkg.in/ini.v1"*/

	"github.com/BurntSushi/toml"
	m "github.com/ValidatorCenter/minter-go-sdk"
)

/*const (
	maxAmntMn = 5 // количество мастернод
)*/

var (
	/*
		CoinMinter    string // Основная монета Minter
		MnAddress     string
		MnPublicKey   [maxAmntMn]string
		MnPrc         [maxAmntMn]int
		AccAddress    string
		AccPrivateKey string
		TimeOut       int64 // Время в мин. обновления статуса
		MinAmnt       int
	*/
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

/*
func cnvStr2Float(amntTokenStr string) float32 {
	var fAmntToken float32 = 0.0
	if amntTokenStr != "" {
		fAmntToken64, err := strconv.ParseFloat(amntTokenStr, 64)
		if err != nil {
			panic(err.Error())
		}
		fAmntToken = float32(fAmntToken64)
	}
	return fAmntToken
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

// Результат выполнения транзакции
type send_transaction struct {
	Code   int               `json:"code"`
	Result TransSendResponse `json:"result"`
	Log    string            `json:"log"`
}

// Хэш транзакции
type TransSendResponse struct {
	Hash string `json:"hash"`
}

// Результат выполнения получения номера операции
type count_transaction struct {
	Code   int                `json:"code"`
	Result TransCountResponse `json:"result"`
}
type TransCountResponse struct {
	Count int `json:"count"`
}

// Возвращает количество исходящих транзакций с данной учетной записи. Это нужно использовать для расчета nonce для новой транзакции.
func getNonce(txAddress string) int {
	url := fmt.Sprintf("%s/api/transactionCount/%s", MnAddress, txAddress)
	res, err := http.Get(url)
	if err != nil {
		panic(err.Error())
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err.Error())
	}

	var data count_transaction
	json.Unmarshal(body, &data)
	return data.Result.Count
}

type blnc_usr struct {
	Code   int           `json:"code"`
	Result BlnctResponse `json:"result"`
}
type BlnctResponse struct {
	Balance map[string]string `json:"balance"`
}

// узнаем баланс
func getBalance(usrAddr string) float32 {
	url := fmt.Sprintf("%s/api/balance/%s", MnAddress, usrAddr)
	res, err := http.Get(url)
	if err != nil {
		fmt.Println("ОШИБКА:", err.Error())
		return -1.0
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("ОШИБКА:", err.Error())
		return -1.0
	}

	var data blnc_usr
	json.Unmarshal(body, &data)

	return cnvStr2Float_18(data.Result.Balance[CoinMinter])
}
*/
// делегирование
func delegate() {
	for _, acc1 := range sdk {
		valueBuy := acc1.GetBalance()
		fmt.Println("valueBuy=", valueBuy)
		// 1bip на прозапас
		if valueBuy < float32(conf.MinAmnt+1) {
			fmt.Printf("Меньше %d%s+1", conf.MinAmnt, conf.CoinNet)
			return
		}
		//fullDelegCoin := float64(valueBuy - 1.0) // 1MNT на комиссию

		// Цикл делегирования

	}

	/*for i := 0; i < maxAmntMn; i++ {
		if MnPublicKey[i] == "" || MnPrc[i] <= 0 {
			continue
		}
		fmt.Printf("###########\n#### %d ####\n###########\n", i+1)

		validatorKey := types.Hex2Bytes(strings.TrimLeft(MnPublicKey[i], "Mp"))

		privKey, err := crypto.HexToECDSA(AccPrivateKey)
		if err != nil {
			panic(err)
		}

		var mntV types.CoinSymbol
		copy(mntV[:], []byte(CoinMinter))

		mng18 := big.NewInt(1000000000000000)                         // убрал 000 (3-нуля)
		mng000 := big.NewFloat(1000)                                  // вот тут 000 (3-нуля)
		amnt := big.NewFloat(fullDelegCoin * float64(MnPrc[i]) / 100) // в процентном соотношение
		fmt.Println("amnt=", amnt.String())
		mnFl := big.NewFloat(0).Mul(amnt, mng000)
		fmt.Println("mnFl=", mnFl.String())
		amntInt_000, _ := mnFl.Int64()
		var amntBInt big.Int
		amntBInt1 := amntBInt.Mul(big.NewInt(amntInt_000), mng18)
		fmt.Println("amntBInt1=", amntBInt1.String())

		buyC := tr.DelegateData{}
		buyC.PubKey = validatorKey
		buyC.Coin = mntV
		buyC.Stake = amntBInt1

		trn := tr.Transaction{}
		trn.Type = tr.TypeDelegate // делегирование
		trn.Data, _ = rlp.EncodeToBytes(buyC)
		trn.Nonce = uint64(getNonce(AccAddress) + 1)
		trn.GasCoin = mntV
		trn.SignatureType = tr.SigTypeSingle

		err = trn.Sign(privKey)
		if err != nil {
			panic(err)
		}

		bts, err := trn.Serialize()
		if err != nil {
			panic(err)
		}
		str := hex.EncodeToString(bts)

		message := map[string]interface{}{
			"transaction": str,
		}
		bytesRepresentation, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}

		fmt.Println("TRANSACTION:", bytes.NewBuffer(bytesRepresentation))

		urlTx := fmt.Sprintf("%s/api/sendTransaction", MnAddress)
		resp, err := http.Post(urlTx, "application/json", bytes.NewBuffer(bytesRepresentation))

		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		fmt.Printf("RESP: %#v\n", resp)

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		var data send_transaction
		json.Unmarshal(body, &data)

		if data.Code == 0 {
			fmt.Println("HASH:", data.Result.Hash)
		} else {
			fmt.Println("ERROR:", data.Code, data.Log)
		}
	}*/
}

func main() {
	//var err error
	ConfFileName := "adlg.toml"

	// проверяем есть ли входной параметр/аргумент
	if len(os.Args) == 2 {
		ConfFileName = os.Args[1]
	}
	fmt.Printf("TOML=%s\n", ConfFileName)

	if _, err := toml.DecodeFile(ConfFileName, &conf); err != nil {
		fmt.Println("Ошибка загрузки toml файла:", err.Error())
		return
	} else {
		fmt.Println("...данные с toml файла = загружены!")
	}

	fmt.Printf("%#v", conf)
	for _, d := range conf.Accounts {
		fmt.Println(d)
		sdk1 := m.SDK{
			MnAddress:     conf.Address,
			AccAddress:    d[0],
			AccPrivateKey: d[1],
		}
		sdk = append(sdk, sdk1)
	}

	for _, d := range conf.Nodes {
		fmt.Println(d)
		n1 := NodeData{
			PubKey: d[0],
			Prc:    d[1],
		}
		nodes = append(nodes, n1)
	}

	for { // бесконечный цикл
		//delegate()
		fmt.Printf("Пауза %dмин.... в этот момент лучше прерывать\n", conf.Timeout)
		time.Sleep(time.Minute * time.Duration(conf.Timeout)) // пауза ~TimeOut~ мин
	}
}
