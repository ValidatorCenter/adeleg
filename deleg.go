package main

import (
	"bytes"
	"encoding/json"
	"os"
	"time"

	//"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"encoding/hex"

	tr "github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"

	"gopkg.in/ini.v1"
)

var (
	MnAddress     string
	MnPublicKey   string
	AccAddress    string
	AccPrivateKey string
	TimeOut       int64 // Время в мин. обновления статуса
	MinAmnt       int
)

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

	return cnvStr2Float_18(data.Result.Balance["MNT"])
}

// делегирование
func delegate() {
	valueBuy := getBalance(AccAddress)
	validatorKey := types.Hex2Bytes(strings.TrimLeft(MnPublicKey, "Mp"))

	fmt.Println("valueBuy=", valueBuy)
	
	// 1MNT на прозапас
	if valueBuy < float32(MinAmnt+1) {
		fmt.Printf("Меньше %dMNT+1",MinAmnt)
		return
	}

	privKey, err := crypto.HexToECDSA(AccPrivateKey)
	if err != nil {
		panic(err)
	}

	var mntV types.CoinSymbol
	copy(mntV[:], []byte("MNT"))

	mng18 := big.NewInt(1000000000000000)         // убрал 000 (3-нуля)
	mng000 := big.NewFloat(1000)                  // вот тут 000 (3-нуля)
	amnt := big.NewFloat(float64(valueBuy - 1.0)) // 1MNT на комиссию
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

	resp, err := http.Post("https://minter-node-1.testnet.minter.network/api/sendTransaction", "application/json", bytes.NewBuffer(bytesRepresentation))

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
}

func main() {
	var err error
	ConfFileName := "adlg.ini"

	// проверяем есть ли входной параметр/аргумент
	if len(os.Args) == 2 {
		ConfFileName = os.Args[1]
	}
	fmt.Printf("INI=%s\n", ConfFileName)

	// INI
	cfg, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, ConfFileName)
	if err != nil {
		fmt.Println("Ошибка загрузки INI файла:", err.Error())
		return
	} else {
		fmt.Println("...данные с INI файла = загружены!")
	}

	secMN := cfg.Section("masternode")
	MnAddress = secMN.Key("ADDRESS").String()
	MnPublicKey = secMN.Key("PUBLICKEY").String()

	accMN := cfg.Section("account")
	AccAddress = accMN.Key("ADDRESS").String()       // Адрес аккаунта
	AccPrivateKey = accMN.Key("PRIVATEKEY").String() // приватный ключ аккаунта

	othMN := cfg.Section("other")
	_TgTimeUpdate, err := strconv.Atoi(othMN.Key("TIMEOUT").String())
	if err != nil {
		fmt.Println(err)
		TimeOut = 11
	}
	TimeOut = int64(_TgTimeUpdate)
	MinAmnt, err = strconv.Atoi(othMN.Key("MINAMOUNT").String())
	if err != nil {
		fmt.Println(err)
		MinAmnt = 100
	}

	for { // бесконечный цикл
		delegate()
		fmt.Printf("Пауза %dмин.... в этот момент лучше прерывать\n", TimeOut)
		time.Sleep(time.Minute * time.Duration(TimeOut)) // пауза ~TimeOut~ мин
	}
}
