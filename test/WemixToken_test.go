package test

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wemade-tree/contract-test/backend"
)

type (
	//Structure to store block partner information
	typePartner struct {
		Serial                 *big.Int
		Partner                common.Address
		Payer                  common.Address
		BlockStaking           *big.Int
		BlockWaitingWithdrawal *big.Int
		BalanceStaking         *big.Int
	}
	typePartnerSlice []*typePartner
)

//Print all block partner information in log.
func (p *typePartner) log(serial *big.Int, t *testing.T) {
	t.Logf("Partner:%s serial:%v", p.Partner.Hex(), serial)
	t.Logf(" -Payer:%v", p.Payer.Hex())
	t.Logf(" -BalanceStaking:%v", p.BalanceStaking)
	t.Logf(" -BlockStaking:%v", p.BlockStaking)
	t.Logf(" -BlockWaitingWithdrawal:%v", p.BlockWaitingWithdrawal)
}

//Retrieve and store all block partner information from the blockchain,
func (p *typePartnerSlice) loadAllStake(t *testing.T, contract *backend.Contract) {
	partnersNumber := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&partnersNumber, "partnersNumber"))

	for i := int64(0); i < partnersNumber.Int64(); i++ {
		s := typePartner{}
		assert.NoError(t, contract.Call(&s, "partnerByIndex", new(big.Int).SetInt64(i)))
		*p = append(*p, &s)
	}
	t.Logf("ok > loadAllStake, partners number: %d", partnersNumber)
}

//After compiling and distributing the contract, return the Contract pointer object.
func depolyWemix(t *testing.T) *backend.Contract {
	contract, err := backend.NewContract("../contracts/WemixToken.sol", "WemixToken")
	assert.NoError(t, err)

	ecoFundKey, _ := crypto.GenerateKey()
	wemixKey, _ := crypto.GenerateKey()

	//deploy contract
	args := []interface{}{
		crypto.PubkeyToAddress(ecoFundKey.PublicKey), //ecoFund address
		crypto.PubkeyToAddress(wemixKey.PublicKey),   //wemix address
	}
	if err := contract.Deploy(args...); err != nil {
		assert.NoError(t, err)
	}
	return contract
}

type Contract struct {
	Code  string       `json:"code"`
	RCode string       `json:"runtime-code"`
	Info  ContractInfo `json:"info"`
}

type ContractInfo struct {
	Source          string      `json:"source"`
	Language        string      `json:"language"`
	LanguageVersion string      `json:"languageVersion"`
	CompilerVersion string      `json:"compilerVersion"`
	CompilerOptions string      `json:"compilerOptions"`
	AbiDefinition   interface{} `json:"abiDefinition"`
	UserDoc         interface{} `json:"userDoc"`
	DeveloperDoc    interface{} `json:"developerDoc"`
	Metadata        string      `json:"metadata"`
}

//Test to compile and deploy the contract
func TestWemixDeploy(t *testing.T) {
	contract := depolyWemix(t)

	// jsonContract := &Contract{
	// 	Code:  hexutil.Encode(contract.Code),
	// 	RCode: hexutil.Encode(contract.Code),
	// 	Info: ContractInfo{
	// 		Source:          contract.Info.Source,
	// 		Language:        contract.Info.Language,
	// 		LanguageVersion: contract.Info.LanguageVersion,
	// 		CompilerVersion: contract.Info.CompilerVersion,
	// 		CompilerOptions: contract.Info.CompilerOptions,
	// 		AbiDefinition:   contract.Info.AbiDefinition,
	// 		UserDoc:         contract.Info.UserDoc,
	// 		DeveloperDoc:    contract.Info.DeveloperDoc,
	// 		Metadata:        contract.Info.Metadata,
	// 	},
	// }

	// b, err := json.Marshal(jsonContract)
	// assert.NoError(t, err)

	// var buff bytes.Buffer
	// gz := gzip.NewWriter(&buff)
	// _, err = gz.Write(b)
	// assert.NoError(t, err)

	// assert.NoError(t, gz.Close())
	// t.Log(hexutil.Encode(b))

	// wemix := (*backend.Contract)(nil)
	// if err := json.Unmarshal(b, &wemix); err != nil {
	// 	assert.NoError(t, err)
	// }
	// contract = wemix

	t.Log("contract source file:", contract.File)
	t.Log("contract name:", contract.Name)
	t.Log("contract Language:", contract.Info.Language)
	t.Log("contract LanguageVersion", contract.Info.LanguageVersion)
	t.Log("contract CompilerVersion", contract.Info.CompilerVersion)
	t.Log("contract bytecode size:", len(contract.Code))

	t.Log("ok > contract address deployed", contract.Address.Hex())

	// t.Log("contract bytecode:", hexutil.Encode(contract.Code))
	// abiBytes, _ := json.Marshal(contract.Info.AbiDefinition)
	// t.Log("contract abi:", string(abiBytes))

}

//Test to verify the variables of the deployed contract.
//Fatal if the expected value and the actual contract value differ.
func TestWemixVariable(t *testing.T) {
	contract := depolyWemix(t)
	block := contract.Backend.Blockchain().CurrentBlock().Header().Number

	checkVariable(t, contract, "name", "WEMIX TOKEN")
	checkVariable(t, contract, "symbol", "WEMIX")
	checkVariable(t, contract, "decimals", uint8(18))
	checkVariable(t, contract, "totalSupply", toBig(t, "1000000000000000000000000000"))
	checkVariable(t, contract, "unitStaking", toBig(t, "2000000000000000000000000"))
	checkVariable(t, contract, "minBlockWaitingWithdrawal", new(big.Int).SetUint64(7776000))
	checkVariable(t, contract, "blockUnitForMint", new(big.Int).SetUint64(60))
	checkVariable(t, contract, "ecoFund", contract.ConstructorInputs[0].(common.Address))
	checkVariable(t, contract, "wemix", contract.ConstructorInputs[1].(common.Address))
	checkVariable(t, contract, "nextPartnerToMint", new(big.Int))
	checkVariable(t, contract, "mintToPartner", new(big.Int).SetUint64(500000000000000000))
	checkVariable(t, contract, "mintToEcoFund", new(big.Int).SetUint64(250000000000000000))
	checkVariable(t, contract, "mintToWemix", new(big.Int).SetUint64(250000000000000000))
	checkVariable(t, contract, "blockToMint", new(big.Int).Add(block, new(big.Int).SetUint64(60)))
}

//Test to execute onlyOwner modifier method.
func TestWemixExecute(t *testing.T) {
	contract := depolyWemix(t)

	executeChangeMethod(t, contract, "unitStaking", big.NewInt(1))
	executeChangeMethod(t, contract, "minBlockWaitingWithdrawal", big.NewInt(1))
	executeChangeMethod(t, contract, "ecoFund", common.HexToAddress("0x0000000000000000000000000000000000000001"))
	executeChangeMethod(t, contract, "wemix", common.HexToAddress("0x0000000000000000000000000000000000000001"))
	executeChangeMethod(t, contract, "mintToPartner", big.NewInt(1))
	executeChangeMethod(t, contract, "mintToEcoFund", big.NewInt(1))
	executeChangeMethod(t, contract, "mintToWemix", big.NewInt(1))
}

//test to run onlyOwner modifier method under non-owner account.
func TestWemixOwner(t *testing.T) {
	contract := depolyWemix(t)

	key, _ := crypto.GenerateKey()

	expecedFail(t, contract, key, "change_unitStaking", big.NewInt(1))
	expecedFail(t, contract, key, "change_minBlockWaitingWithdrawal", big.NewInt(1))
	expecedFail(t, contract, key, "change_ecoFund", common.HexToAddress("0x0000000000000000000000000000000000000001"))
	expecedFail(t, contract, key, "change_wemix", common.HexToAddress("0x0000000000000000000000000000000000000002"))
	expecedFail(t, contract, key, "change_mintToPartner", big.NewInt(1))
	expecedFail(t, contract, key, "change_mintToWemix", big.NewInt(1))
	expecedFail(t, contract, key, "transferOwnership", func() common.Address {
		k, _ := crypto.GenerateKey()
		return crypto.PubkeyToAddress(k.PublicKey)
	}())

	newOwnerKey, _ := crypto.GenerateKey()
	expecedSuccess(t, contract, nil, "transferOwnership", crypto.PubkeyToAddress(newOwnerKey.PublicKey))
	expecedSuccess(t, contract, newOwnerKey, "transferOwnership", contract.Owner)
}

//Test to run addAllowedStaker method.
func TestWemixAllowedPartner(t *testing.T) {
	contract := depolyWemix(t)

	//make an error occur
	r, err := contract.Execute(nil, "stake", new(big.Int))
	assert.NoError(t, err)
	assert.True(t, r.Status == 0)

	//make partner
	partnerKey, _ := crypto.GenerateKey()
	partner := crypto.PubkeyToAddress(partnerKey.PublicKey)

	//addAllowedPartner
	r, err = contract.Execute(nil, "addAllowedPartner", partner)
	assert.NoError(t, err)
	assert.True(t, r.Status == 1)

	topics := []common.Hash{}
	r, err = contract.Execute(nil, "stakeDelegated", partner, new(big.Int))
	assert.NoError(t, err)
	assert.True(t, r.Status == 1)
	for _, g := range r.Logs {
		if g.Topics[0] == contract.Abi.Events["Staked"].Id() {
			topics = append(topics, g.Topics...)
		}
	}
	assert.Equal(t, common.BytesToAddress(topics[1].Bytes()), partner)
	assert.Equal(t, common.BytesToAddress(topics[2].Bytes()), contract.Owner)

	t.Log("ok > test addAllowedPartner")
}

//test staking
func TestWemixStake(t *testing.T) {
	contract := depolyWemix(t)

	testStake(t, contract, false)
}

func testStake(t *testing.T, contract *backend.Contract, showStakeInfo bool) typeKeyMap {
	unitStaking := func() *big.Int {
		ret := (*big.Int)(nil)
		assert.NoError(t, contract.Call(&ret, "unitStaking"))
		return ret
	}()

	minBlockWaitingWithdrawal := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&minBlockWaitingWithdrawal, "minBlockWaitingWithdrawal"))

	countExecuteStake := int64(0)
	partnerKeyMap := typeKeyMap{}

	_stake := func(delegation bool, partner common.Address, payerKey *ecdsa.PrivateKey, waitBlock *big.Int) *typePartner {
		var r *types.Receipt
		var err error
		//addAllowedPartner
		r, err = contract.Execute(nil, "addAllowedPartner", partner)
		assert.NoError(t, err)
		assert.True(t, r.Status == 1)

		if delegation == true {
			r, err = contract.Execute(payerKey, "stakeDelegated", partner, waitBlock)
		} else {
			r, err = contract.Execute(payerKey, "stake", waitBlock)
		}
		assert.NoError(t, err)
		assert.True(t, r.Status == 1)

		serial := (*big.Int)(nil)
		for _, g := range r.Logs {
			if g.Topics[0] == contract.Abi.Events["Staked"].Id() {
				serial = g.Topics[3].Big()
			}
		}
		assert.NotNil(t, serial)
		countExecuteStake++

		result := typePartner{}
		assert.NoError(t, contract.Call(&result, "partnerBySerial", serial))
		if showStakeInfo == true {
			result.log(serial, t)
		}
		return &result
	}

	makePartner := func() (common.Address, *ecdsa.PrivateKey) {
		partnerKey, _ := crypto.GenerateKey()
		partner := crypto.PubkeyToAddress(partnerKey.PublicKey)
		partnerKeyMap[partner] = partnerKey
		return partner, partnerKey
	}

	//stake
	for i := 0; i < 3; i++ {
		partner, partnerKey := makePartner()

		//send wemix for testing,
		amount := new(big.Int).Mul(unitStaking, new(big.Int).SetInt64(int64(i+1)))
		r, err := contract.Execute(nil, "transfer", partner, amount)
		assert.NoError(t, err)
		assert.True(t, r.Status == 1)
		waitBlock := new(big.Int).Mul(minBlockWaitingWithdrawal, new(big.Int).SetInt64(int64(i+1)))

		for {
			result := _stake(false, partner, partnerKey, waitBlock)

			assert.Equal(t, result.Payer, result.Partner)
			assert.Equal(t, partner, result.Payer)

			//when all tokens received have been exhausted, terminate staking.
			balance := (*big.Int)(nil)
			assert.NoError(t, contract.Call(&balance, "balanceOf", partner))
			if balance.Sign() == 0 {
				break
			}
		}
	}

	//delegated stake
	for i := 0; i < 5; i++ {
		partner, _ := makePartner()

		amount := new(big.Int).Mul(unitStaking, new(big.Int).SetInt64(int64(i+1)))

		waitBlock := new(big.Int).Mul(minBlockWaitingWithdrawal, new(big.Int).SetInt64(int64(i+1)))

		for {
			result := _stake(true, partner, contract.OwnerKey, waitBlock)

			assert.NotEqual(t, result.Payer, result.Partner)
			assert.Equal(t, contract.Owner, result.Payer)

			amount = new(big.Int).Sub(amount, result.BalanceStaking)
			if amount.Sign() == 0 {
				break
			}
		}
	}

	//Compare the number of block partners registered with the number registered on the blockchain.
	partnersNumber := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&partnersNumber, "partnersNumber"))
	assert.True(t, partnersNumber.Cmp(new(big.Int).SetInt64(countExecuteStake)) == 0)

	return partnerKeyMap
}

//test to withdraw
func TestWemixWithdraw(t *testing.T) {
	contract := depolyWemix(t)

	//change withdrawalWaitingMinBlockd short for testing.
	r, err := contract.Execute(nil, "change_minBlockWaitingWithdrawal", new(big.Int).SetUint64(1000))
	assert.NoError(t, err)
	assert.True(t, r.Status == 1)

	partnerKeyMap := testStake(t, contract, false)

	stakes := typePartnerSlice{}
	stakes.loadAllStake(t, contract)

	contractBalance := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&contractBalance, "balanceOf", contract.Address))

	totalStakeBalance := new(big.Int)
	for i := 0; i < len(stakes); i++ {
		totalStakeBalance.Add(totalStakeBalance, stakes[i].BalanceStaking)
	}

	assert.True(t, totalStakeBalance.Cmp(contractBalance) == 0)
	t.Logf("ok > contract's balance: %v, total stake balance:%v", contractBalance, totalStakeBalance)

	for {
		for i, s := range stakes {
			key := (*ecdsa.PrivateKey)(nil)
			if s.Partner == s.Payer {
				key = partnerKeyMap[s.Payer]
			}

			r, err := contract.Execute(key, "withdraw", s.Serial)
			assert.NoError(t, err)

			block := contract.Backend.Blockchain().CurrentBlock().Header().Number
			blockWithdrawable := new(big.Int).Add(s.BlockStaking, s.BlockWaitingWithdrawal)
			if r.Status == 1 {
				assert.True(t, block.Cmp(blockWithdrawable) >= 0)
				t.Logf("ok > withdrawal : %v", s.Serial)
				stakes[i] = stakes[len(stakes)-1]
				stakes = stakes[:len(stakes)-1]
				break
			} else {
				assert.True(t, block.Cmp(blockWithdrawable) < 0)
			}
		}

		if len(stakes) == 0 {
			break
		}

		//make block
		contract.Backend.Commit()
	}

	for staker, key := range partnerKeyMap {
		balance := (*big.Int)(nil)
		assert.NoError(t, contract.Call(&balance, "balanceOf", staker))
		if balance.Sign() > 0 {
			r, err := contract.Execute(key, "transfer", contract.Owner, balance)
			assert.NoError(t, err)
			assert.True(t, r.Status == 1)
			t.Log("ok > return token to owner")
		}
	}
}

func testMint(t *testing.T, contract *backend.Contract) {
	stakes := typePartnerSlice{}
	stakes.loadAllStake(t, contract)

	balancePartners := func() map[common.Address]*big.Int {
		m := make(map[common.Address]*big.Int)
		for _, s := range stakes {
			if _, ok := m[s.Partner]; ok == true {
				continue
			}
			b := (*big.Int)(nil)
			assert.NoError(t, contract.Call(&b, "balanceOf", s.Partner))
			m[s.Partner] = b
		}
		return m
	}()

	wemix := common.Address{}
	assert.NoError(t, contract.Call(&wemix, "wemix"))

	balanceWemix := func() *big.Int {
		b := (*big.Int)(nil)
		assert.NoError(t, contract.Call(&b, "balanceOf", wemix))
		return b
	}()

	ecoFund := common.Address{}
	assert.NoError(t, contract.Call(&ecoFund, "ecoFund"))

	balanceEcoFund := func() *big.Int {
		b := (*big.Int)(nil)
		assert.NoError(t, contract.Call(&b, "balanceOf", ecoFund))
		return b
	}()

	mintToPartner := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&mintToPartner, "mintToPartner"))

	mintToWemix := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&mintToWemix, "mintToWemix"))

	mintToEcoFund := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&mintToEcoFund, "mintToEcoFund"))

	nextPartnerToMint := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&nextPartnerToMint, "nextPartnerToMint"))

	blockUnitForMint := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&blockUnitForMint, "blockUnitForMint"))

	indexNext := int(nextPartnerToMint.Uint64())
	totalMinted := new(big.Int)
	countExpectedBalance := func() {
		if len(stakes) == 0 {
			return
		}

		if indexNext >= len(stakes) {
			indexNext = 0
		}

		s := stakes[indexNext]
		balancePartners[s.Partner].Add(balancePartners[s.Partner], new(big.Int).Mul(mintToPartner, blockUnitForMint))
		totalMinted.Add(totalMinted, new(big.Int).Mul(mintToPartner, blockUnitForMint))
		indexNext++

		balanceWemix.Add(balanceWemix, new(big.Int).Mul(mintToWemix, blockUnitForMint))
		totalMinted.Add(totalMinted, new(big.Int).Mul(mintToWemix, blockUnitForMint))
		balanceEcoFund.Add(balanceEcoFund, new(big.Int).Mul(mintToEcoFund, blockUnitForMint))
		totalMinted.Add(totalMinted, new(big.Int).Mul(mintToEcoFund, blockUnitForMint))
	}

	initialTotalSupply := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&initialTotalSupply, "totalSupply"))

	startBlock := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&startBlock, "blockToMint"))

	timesMinting := uint64(210)

	for i := uint64(0); i < timesMinting; i++ {
		blockToMint := (*big.Int)(nil)
		assert.NoError(t, contract.Call(&blockToMint, "blockToMint"))

		currentBlock := contract.Backend.Blockchain().CurrentBlock().Header().Number
		if currentBlock.Cmp(blockToMint) < 0 {
			commitCnt := new(big.Int).Sub(blockToMint, contract.Backend.Blockchain().CurrentBlock().Header().Number)

			for b := uint64(0); b < commitCnt.Uint64(); b++ {
				contract.Backend.Commit() //make block
			}
		}
		key, _ := crypto.GenerateKey()
		r, err := contract.Execute(key, "mint")
		assert.NoError(t, err)
		assert.True(t, r.Status == 1)
		countExpectedBalance()
	}
	endBlock := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&endBlock, "blockToMint"))

	isMintable := bool(false)
	assert.NoError(t, contract.Call(&isMintable, "isMintable"))
	assert.True(t, isMintable == false)

	checkBalance := func(tag string, addr common.Address, expected *big.Int) {
		got := (*big.Int)(nil)
		assert.NoError(t, contract.Call(&got, "balanceOf", addr))
		assert.True(t, expected.Cmp(got) == 0)
		t.Logf("ok > %s(%s) balance  expected:%v, got:%v", tag, addr.Hex(), expected, got)
	}

	for a, expected := range balancePartners {
		checkBalance("partner", a, expected)
	}
	checkBalance("wemix", wemix, balanceWemix)
	checkBalance("ecoFund", ecoFund, balanceEcoFund)

	totalSupply := (*big.Int)(nil)
	assert.NoError(t, contract.Call(&totalSupply, "totalSupply"))
	expected := new(big.Int).Add(initialTotalSupply, totalMinted)
	assert.True(t, totalSupply.Cmp(expected) == 0)
	t.Logf("ok > match totalSupply and expected totalSupply after mint, got :%d, expectd: %d", totalSupply, expected)
}

//After registering block partners, do minting test and check the amount of minting.
func TestWemixMint(t *testing.T) {
	contract := depolyWemix(t)
	testStake(t, contract, true)

	testMint(t, contract)
}

//Test minting without block partner and check minting amount.
func TestWemixMintWithoutPartner(t *testing.T) {
	contract := depolyWemix(t)
	testMint(t, contract)
}
