package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/token"
	"github.com/Unknwon/goconfig"
	"os"
	"strconv"
	"time"
)

var cfg *goconfig.ConfigFile

type EidosAccount struct  {
	api *eos.API
	from eos.AccountName
	to eos.AccountName
	once int
	tokento eos.AccountName
	tokenvalue int
	minernum int
}

func (s *EidosAccount) Init() (error) {
	url, err := cfg.GetValue("config", "api")
	if (err != nil) {
		return err
	}
	s.api = eos.New(url)

	prikey ,err := cfg.GetValue("config", "prikey")

	keyBag := &eos.KeyBag{}
	err = keyBag.ImportPrivateKey(prikey)
	if err != nil {
		return fmt.Errorf("import private key: %s", err)
	}
	s.api.SetSigner(keyBag)

	from, err := cfg.GetValue("config", "from")
	if (err != nil) {
		return err
	}
	s.from = eos.AccountName(from)

	to, err := cfg.GetValue("config", "to")
	if (err != nil) {
		return err
	}
	s.to = eos.AccountName(to)

	tokento, err := cfg.GetValue("config", "tokento")
	if (err == nil) {
		s.tokento = eos.AccountName(tokento)
	}

	tokenvalue, err := cfg.GetValue("config", "tokenvalue")
	if (err == nil) {
		s.tokenvalue, err = strconv.Atoi(tokenvalue)
	}

	sonce, err := cfg.GetValue("config", "once")
	s.once, err = strconv.Atoi(sonce)

	s.minernum=  0

	return nil
}

func (s *EidosAccount) Send() (error) {
	quantity, err := eos.NewEOSAssetFromString("0.0001 EOS")
	if err != nil {
		return fmt.Errorf("invalid quantity: %s", err)
	}

	txOpts := &eos.TxOptions{}
	if err := txOpts.FillFromChain(s.api); err != nil {
		return fmt.Errorf("filling tx opts: %s", err)
	}

	var memo = ""
	var trs = token.NewTransfer(s.from, s.to, quantity, memo)
	acts := make([]*eos.Action, s.once)
	for i:=0; i < s.once ; i++ {
		acts[i] = trs
	}

	tx := eos.NewTransaction(acts, txOpts)
	signedTx, packedTx, err := s.api.SignTransaction(tx, txOpts.ChainID, eos.CompressionNone)
	if err != nil {
		return fmt.Errorf("sign transaction: %s", err)
	}

	_, err = json.MarshalIndent(signedTx, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshalling transaction: %s", err)
	}

	response, err := s.api.PushTransaction(packedTx)
	if err != nil {
		return fmt.Errorf("push transaction: %s", err)
	}

	fmt.Printf("[%d] Transaction [%s] success.\n", s.minernum, hex.EncodeToString(response.Processed.ID))
	s.minernum++
	return nil
}

func (s *EidosAccount) SendToken() (error) {
	info, err := s.api.GetCurrencyBalance(s.from, "EIDOS", eos.AccountName("eidosonecoin"))
	if (err != nil) {
		return err
	}

	fmt.Println(info)

	if (info[0].Amount < eos.Int64(s.tokenvalue * 10000)) {
		return nil
	}

	txOpts := &eos.TxOptions{}
	if err := txOpts.FillFromChain(s.api); err != nil {
		return fmt.Errorf("filling tx opts: %s", err)
	}

	var quantity = info[0]
	var memo = ""

	tx := eos.NewTransaction([]*eos.Action{
		&eos.Action{
			Account: eos.AN("eidosonecoin"),
			Name: eos.ActN("transfer"),
			Authorization: []eos.PermissionLevel{
				{Actor: s.from, Permission: eos.PN("active")},
			},
			ActionData: eos.NewActionData(token.Transfer{
				From:     s.from,
				To:       s.tokento,
				Quantity: quantity,
				Memo:     memo,
			}),
		},
	}, txOpts)
	signedTx, packedTx, err := s.api.SignTransaction(tx, txOpts.ChainID, eos.CompressionNone)
	if err != nil {
		return fmt.Errorf("sign transaction: %s", err)
	}

	_, err = json.MarshalIndent(signedTx, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshalling transaction: %s", err)
	}

	response, err := s.api.PushTransaction(packedTx)
	if err != nil {
		return fmt.Errorf("push transaction: %s", err)
	}

	fmt.Printf("SendToken Transaction [%s] succesfully.\n", hex.EncodeToString(response.Processed.ID))
	return nil
}

func main() {
	config, err := goconfig.LoadConfigFile("config.ini")    //加载配置文件
	if err != nil {
		fmt.Println("config error ：", err)
		os.Exit(-1)
	}
	cfg = config

	var eidos EidosAccount
	err = eidos.Init()
	if (err != nil) {
		fmt.Println("eidos Init error：", err)
		os.Exit(-1)
	}
	
	sinterval, err := cfg.GetValue("config", "interval")
	interval, err := strconv.Atoi(sinterval)
	fmt.Println("interval: ", interval)

	fmt.Println("once: ", eidos.once)

	for i := 0; i < 1;  {
		err = eidos.Send()
		if (err != nil) {
			fmt.Println(err)
		}
		time.Sleep(time.Millisecond * time.Duration(interval))
		if (eidos.tokento != "" && eidos.minernum%10 == 0) {
			err = eidos.SendToken()
			if (err != nil) {
				fmt.Println(err)
			}
		}
	}
}