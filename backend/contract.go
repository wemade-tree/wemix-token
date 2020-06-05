package backend

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

//Contract struct holds data before compilation and information after compilation.
type Contract struct {
	File              string
	Name              string
	Backend           *backends.SimulatedBackend
	OwnerKey          *ecdsa.PrivateKey
	Owner             common.Address
	Info              *compiler.ContractInfo
	ConstructorInputs []interface{}
	Abi               *abi.ABI
	Code              []byte
	Address           common.Address
	BlockDeployed     *big.Int
}

//NewContract is to create simulatied backend and compile solidity code
func NewContract(file, name string) (*Contract, error) {

	ownerKey, _ := crypto.GenerateKey()

	r := &Contract{
		File: file,
		Name: name,
		//creates a new binding backend using a simulated blockchain
		Backend: backends.NewSimulatedBackend(
			nil,
			10000000,
		),
		OwnerKey: ownerKey,
		Owner:    crypto.PubkeyToAddress(ownerKey.PublicKey),
	}
	//compile
	if err := r.compile(); err != nil {
		return nil, err
	}

	return r, nil
}

func (p *Contract) compile() error {
	contracts, err := compiler.CompileSolidity("", p.File)
	if err != nil {
		return err
	}

	//Get the contract to test from the compiled contracts.
	contract, ok := contracts[fmt.Sprintf("%s:%s", p.File, p.Name)]
	if ok == false {
		fmt.Errorf("%s contract is not here", p.Name)
	}
	//make abi.ABI instance
	abiBytes, err := json.Marshal(contract.Info.AbiDefinition)
	if err != nil {
		return err
	}
	abi, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return err
	}
	p.Info = &contract.Info
	p.Abi = &abi
	p.Code = common.FromHex(contract.Code)
	return nil
}

//Deploy makes creation contract tx and receives the result by receit.
func (p *Contract) Deploy(args ...interface{}) error {
	input, err := p.Abi.Pack("", args...) //constructor's inputs
	if err != nil {
		return err
	}

	p.ConstructorInputs = args // Save for later checkout

	//make tx for contract creation
	tx := types.NewContractCreation(0, big.NewInt(0), 3000000, big.NewInt(0), append(p.Code, input...))
	//signing
	tx, _ = types.SignTx(tx, types.HomesteadSigner{}, p.OwnerKey)
	//sned tx to simulated backend
	if err := p.Backend.SendTransaction(context.Background(), tx); err != nil {
		return err
	}
	//make block
	p.Backend.Commit()

	//get contract address through receipt
	receipt, err := p.Backend.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return fmt.Errorf("status of deploy tx receipt: %v", receipt.Status)
	}
	//get contract's address and block deployed from the receipt
	p.Address = receipt.ContractAddress
	p.BlockDeployed = receipt.BlockNumber
	return nil
}

// Call is Invokes a view method with args and then receive the result unpacked.
func (p *Contract) Call(result interface{}, method string, args ...interface{}) error {
	if input, err := p.Abi.Pack(method, args...); err != nil {
		return err
	} else {
		msg := ethereum.CallMsg{From: common.Address{}, To: &p.Address, Data: input}

		out := result
		if output, err := p.Backend.CallContract(context.TODO(), msg, nil); err != nil {
			return err
		} else if err := p.Abi.Unpack(out, method, output); err != nil {
			return err
		}
	}
	return nil
}

//LowCall returns method's output in a different way than Call.
func (p *Contract) LowCall(method string, args ...interface{}) ([]interface{}, error) {
	if input, err := p.Abi.Pack(method, args...); err != nil {
		return nil, err
	} else {
		msg := ethereum.CallMsg{From: common.Address{}, To: &p.Address, Data: input}
		if out, err := p.Backend.CallContract(context.TODO(), msg, nil); err != nil {
			return []interface{}{}, err
		} else {
			if ret, err := p.Abi.Methods[method].Outputs.UnpackValues(out); err != nil {
				return []interface{}{}, err
			} else {
				return ret, nil
			}
		}
	}
}

//Execute executes the contract's method. For that, take tx with singer's key, method and inputs,
//and then send it to the simulated backend, and return the receipt.
func (p *Contract) Execute(key *ecdsa.PrivateKey, method string, args ...interface{}) (*types.Receipt, error) {
	if key == nil {
		key = p.OwnerKey
	}

	data, err := p.Abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}

	nonce, err := p.Backend.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(key.PublicKey))
	if err != nil {
		return nil, err
	}

	tx, err := types.SignTx(types.NewTransaction(nonce, p.Address, new(big.Int), uint64(10000000), big.NewInt(0), data),
		types.HomesteadSigner{}, key)
	if err != nil {
		return nil, err
	}
	if err := p.Backend.SendTransaction(context.Background(), tx); err != nil {
		return nil, err
	}
	p.Backend.Commit()

	receipt, err := p.Backend.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return nil, err
	}

	return receipt, nil
}
