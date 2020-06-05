package test

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/wemade-tree/contract-test/backend"
)

type typeKeyMap map[common.Address]*ecdsa.PrivateKey

//Converts the given data into a byte slice and returns it.
func toBytes(t *testing.T, data interface{}) []byte {
	var buf bytes.Buffer
	assert.NoError(t, gob.NewEncoder(&buf).Encode(data))
	return buf.Bytes()
}

//Converts the given number string based decimal number into big.Int type and returns it.
func toBig(t *testing.T, value10 string) *big.Int {
	ret, b := new(big.Int).SetString(value10, 10)
	assert.True(t, b)
	return ret
}

//checkVariable compares value stored in the blockchain with a given expected value
func checkVariable(t *testing.T, contract *backend.Contract, method string, expected interface{}) {
	ret, err := contract.LowCall(method)
	assert.NoError(t, err)
	assert.Equal(t, toBytes(t, ret[0]), toBytes(t, expected))

	switch expected.(type) {
	case common.Address:
		t.Log(method, ret[0].(common.Address).Hex())
	default:
		t.Log(method, ret[0])
	}

}

//executeChangeMethod executes the method with the "change_" prefix in the contract,
//and then compares whether the given arg argument is applied well.
//The argument of "change_" prefixed method is assumed to be one.
func executeChangeMethod(t *testing.T, contract *backend.Contract, methodExceptCall string, arg interface{}) {
	changeMethod := "change_" + methodExceptCall

	r, err := contract.Execute(nil, changeMethod, arg)
	assert.NoError(t, err)
	assert.True(t, r.Status == 1)

	changed, err := contract.LowCall(methodExceptCall)
	assert.NoError(t, err)
	assert.Equal(t, toBytes(t, changed[0]), toBytes(t, arg))
}

//causes contract execution to fail.
func expecedFail(t *testing.T, contract *backend.Contract, key *ecdsa.PrivateKey, method string, arg ...interface{}) {
	r, err := contract.Execute(key, method, arg...)
	assert.NoError(t, err)
	assert.True(t, r.Status == 0)
}

//checks if the contract execution is successful..
func expecedSuccess(t *testing.T, contract *backend.Contract, key *ecdsa.PrivateKey, method string, arg ...interface{}) {
	r, err := contract.Execute(key, method, arg...)
	assert.NoError(t, err)
	assert.True(t, r.Status == 1)
}
