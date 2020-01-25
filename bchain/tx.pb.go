// Code generated by protoc-gen-go. DO NOT EDIT.
// source: tx.proto

/*
Package bchain is a generated protocol buffer package.

It is generated from these files:
	tx.proto

It has these top-level messages:
	ProtoTransaction
*/
package bchain

import proto "github.com/golang/protobuf/proto"
import bech32 "github.com/martinboehm/btcutil/bech32"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ProtoTransaction struct {
	Txid      []byte                       `protobuf:"bytes,1,opt,name=Txid,json=txid,proto3" json:"Txid,omitempty"`
	Hex       []byte                       `protobuf:"bytes,2,opt,name=Hex,json=hex,proto3" json:"Hex,omitempty"`
	Blocktime uint64                       `protobuf:"varint,3,opt,name=Blocktime,json=blocktime" json:"Blocktime,omitempty"`
	Locktime  uint32                       `protobuf:"varint,4,opt,name=Locktime,json=locktime" json:"Locktime,omitempty"`
	Height    uint32                       `protobuf:"varint,5,opt,name=Height,json=height" json:"Height,omitempty"`
	Vin       []*ProtoTransaction_VinType  `protobuf:"bytes,6,rep,name=Vin,json=vin" json:"Vin,omitempty"`
	Vout      []*ProtoTransaction_VoutType `protobuf:"bytes,7,rep,name=Vout,json=vout" json:"Vout,omitempty"`
	Version   int32                        `protobuf:"varint,8,opt,name=Version,json=version" json:"Version,omitempty"`
}

func (m *ProtoTransaction) Reset()                    { *m = ProtoTransaction{} }
func (m *ProtoTransaction) String() string            { return proto.CompactTextString(m) }
func (*ProtoTransaction) ProtoMessage()               {}
func (*ProtoTransaction) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *ProtoTransaction) GetTxid() []byte {
	if m != nil {
		return m.Txid
	}
	return nil
}

func (m *ProtoTransaction) GetHex() []byte {
	if m != nil {
		return m.Hex
	}
	return nil
}

func (m *ProtoTransaction) GetBlocktime() uint64 {
	if m != nil {
		return m.Blocktime
	}
	return 0
}

func (m *ProtoTransaction) GetLocktime() uint32 {
	if m != nil {
		return m.Locktime
	}
	return 0
}

func (m *ProtoTransaction) GetHeight() uint32 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *ProtoTransaction) GetVin() []*ProtoTransaction_VinType {
	if m != nil {
		return m.Vin
	}
	return nil
}

func (m *ProtoTransaction) GetVout() []*ProtoTransaction_VoutType {
	if m != nil {
		return m.Vout
	}
	return nil
}

func (m *ProtoTransaction) GetVersion() int32 {
	if m != nil {
		return m.Version
	}
	return 0
}

type ProtoTransaction_VinType struct {
	Coinbase     string   `protobuf:"bytes,1,opt,name=Coinbase,json=coinbase" json:"Coinbase,omitempty"`
	Txid         []byte   `protobuf:"bytes,2,opt,name=Txid,json=txid,proto3" json:"Txid,omitempty"`
	Vout         uint32   `protobuf:"varint,3,opt,name=Vout,json=vout" json:"Vout,omitempty"`
	ScriptSigHex []byte   `protobuf:"bytes,4,opt,name=ScriptSigHex,json=scriptSigHex,proto3" json:"ScriptSigHex,omitempty"`
	Sequence     uint32   `protobuf:"varint,5,opt,name=Sequence,json=sequence" json:"Sequence,omitempty"`
	Addresses    []string `protobuf:"bytes,6,rep,name=Addresses,json=addresses" json:"Addresses,omitempty"`
}

func (m *ProtoTransaction_VinType) Reset()                    { *m = ProtoTransaction_VinType{} }
func (m *ProtoTransaction_VinType) String() string            { return proto.CompactTextString(m) }
func (*ProtoTransaction_VinType) ProtoMessage()               {}
func (*ProtoTransaction_VinType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

func (m *ProtoTransaction_VinType) GetCoinbase() string {
	if m != nil {
		return m.Coinbase
	}
	return ""
}

func (m *ProtoTransaction_VinType) GetTxid() []byte {
	if m != nil {
		return m.Txid
	}
	return nil
}

func (m *ProtoTransaction_VinType) GetVout() uint32 {
	if m != nil {
		return m.Vout
	}
	return 0
}

func (m *ProtoTransaction_VinType) GetScriptSigHex() []byte {
	if m != nil {
		return m.ScriptSigHex
	}
	return nil
}

func (m *ProtoTransaction_VinType) GetSequence() uint32 {
	if m != nil {
		return m.Sequence
	}
	return 0
}

func (m *ProtoTransaction_VinType) GetAddresses() []string {
	if m != nil {
		return m.Addresses
	}
	return nil
}

type ProtoTransaction_VoutType struct {
	ValueSat        []byte   `protobuf:"bytes,1,opt,name=ValueSat,json=valueSat,proto3" json:"ValueSat,omitempty"`
	N               uint32   `protobuf:"varint,2,opt,name=N,json=n" json:"N,omitempty"`
	ScriptPubKeyHex []byte   `protobuf:"bytes,3,opt,name=ScriptPubKeyHex,json=scriptPubKeyHex,proto3" json:"ScriptPubKeyHex,omitempty"`
	Addresses       []string `protobuf:"bytes,4,rep,name=Addresses,json=addresses" json:"Addresses,omitempty"`
}

func (m *ProtoTransaction_VoutType) Reset()                    { *m = ProtoTransaction_VoutType{} }
func (m *ProtoTransaction_VoutType) String() string            { return proto.CompactTextString(m) }
func (*ProtoTransaction_VoutType) ProtoMessage()               {}
func (*ProtoTransaction_VoutType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 1} }

func (m *ProtoTransaction_VoutType) GetValueSat() []byte {
	if m != nil {
		return m.ValueSat
	}
	return nil
}

func (m *ProtoTransaction_VoutType) GetN() uint32 {
	if m != nil {
		return m.N
	}
	return 0
}

func (m *ProtoTransaction_VoutType) GetScriptPubKeyHex() []byte {
	if m != nil {
		return m.ScriptPubKeyHex
	}
	return nil
}

func (m *ProtoTransaction_VoutType) GetAddresses() []string {
	if m != nil {
		return m.Addresses
	}
	return nil
}

type ProtoTransaction_WitnessAddressType struct {
	Version        uint32 `protobuf:"varint,1,opt,name=Version,json=version" json:"Version,omitempty"`
	WitnessProgram string `protobuf:"bytes,2,opt,name=WitnessProgram,json=witnessProgram" json:"WitnessProgram,omitempty"`
}

func (m *ProtoTransaction_WitnessAddressType) Reset()         { *m = ProtoTransaction_WitnessAddressType{} }
func (m *ProtoTransaction_WitnessAddressType) String() string { return proto.CompactTextString(m) }
func (*ProtoTransaction_WitnessAddressType) ProtoMessage()    {}
func (*ProtoTransaction_WitnessAddressType) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{0, 2}
}

func (m *ProtoTransaction_WitnessAddressType) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *ProtoTransaction_WitnessAddressType) GetWitnessProgram() string {
	if m != nil {
		return m.WitnessProgram
	}
	return ""
}

func (m *ProtoTransaction_WitnessAddressType) ToString() string {
	if m != nil {
		if len(m.WitnessProgram) <= 4 && m.WitnessProgram == "burn" {
			return "burn"
		}
		// Convert data to base32:
		conv, err := bech32.ConvertBits([]byte(m.WitnessProgram), 8, 5, true)
		if err != nil {
			fmt.Println("Error:", err)
			return ""
		}
		encoded, err := bech32.Encode("sys", conv)
		if err != nil {
			fmt.Println("Error:", err)
			return ""
		}
		return encoded
	}
	return ""
}


type ProtoTransaction_RangeAmountPairType struct {
	WitnessAddress *ProtoTransaction_WitnessAddressType `protobuf:"bytes,1,opt,name=WitnessAddress,json=witnessAddress" json:"WitnessAddress,omitempty"`
	ValueSat       []byte                               `protobuf:"bytes,2,opt,name=ValueSat,json=valueSat,proto3" json:"ValueSat,omitempty"`
}

func (m *ProtoTransaction_RangeAmountPairType) Reset()         { *m = ProtoTransaction_RangeAmountPairType{} }
func (m *ProtoTransaction_RangeAmountPairType) String() string { return proto.CompactTextString(m) }
func (*ProtoTransaction_RangeAmountPairType) ProtoMessage()    {}
func (*ProtoTransaction_RangeAmountPairType) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{0, 3}
}

func (m *ProtoTransaction_RangeAmountPairType) GetWitnessAddress() *ProtoTransaction_WitnessAddressType {
	if m != nil {
		return m.WitnessAddress
	}
	return nil
}

func (m *ProtoTransaction_RangeAmountPairType) GetValueSat() []byte {
	if m != nil {
		return m.ValueSat
	}
	return nil
}

type ProtoTransaction_AssetAllocationTupleType struct {
	Asset          uint32                               `protobuf:"varint,1,opt,name=Asset,json=asset" json:"Asset,omitempty"`
	WitnessAddress *ProtoTransaction_WitnessAddressType `protobuf:"bytes,2,opt,name=WitnessAddress,json=witnessAddress" json:"WitnessAddress,omitempty"`
}

func (m *ProtoTransaction_AssetAllocationTupleType) Reset() {
	*m = ProtoTransaction_AssetAllocationTupleType{}
}
func (m *ProtoTransaction_AssetAllocationTupleType) String() string { return proto.CompactTextString(m) }
func (*ProtoTransaction_AssetAllocationTupleType) ProtoMessage()    {}
func (*ProtoTransaction_AssetAllocationTupleType) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{0, 4}
}

func (m *ProtoTransaction_AssetAllocationTupleType) GetAsset() uint32 {
	if m != nil {
		return m.Asset
	}
	return 0
}

func (m *ProtoTransaction_AssetAllocationTupleType) GetWitnessAddress() *ProtoTransaction_WitnessAddressType {
	if m != nil {
		return m.WitnessAddress
	}
	return nil
}

type ProtoTransaction_AssetAllocationType struct {
	AssetAllocationTuple         *ProtoTransaction_AssetAllocationTupleType `protobuf:"bytes,1,opt,name=AssetAllocationTuple,json=assetAllocationTuple" json:"AssetAllocationTuple,omitempty"`
	ListSendingAllocationAmounts []*ProtoTransaction_RangeAmountPairType    `protobuf:"bytes,2,rep,name=ListSendingAllocationAmounts,json=listSendingAllocationAmounts" json:"ListSendingAllocationAmounts,omitempty"`
}

func (m *ProtoTransaction_AssetAllocationType) Reset()         { *m = ProtoTransaction_AssetAllocationType{} }
func (m *ProtoTransaction_AssetAllocationType) String() string { return proto.CompactTextString(m) }
func (*ProtoTransaction_AssetAllocationType) ProtoMessage()    {}
func (*ProtoTransaction_AssetAllocationType) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{0, 5}
}

func (m *ProtoTransaction_AssetAllocationType) GetAssetAllocationTuple() *ProtoTransaction_AssetAllocationTupleType {
	if m != nil {
		return m.AssetAllocationTuple
	}
	return nil
}

func (m *ProtoTransaction_AssetAllocationType) GetListSendingAllocationAmounts() []*ProtoTransaction_RangeAmountPairType {
	if m != nil {
		return m.ListSendingAllocationAmounts
	}
	return nil
}

func init() {
	proto.RegisterType((*ProtoTransaction)(nil), "bchain.ProtoTransaction")
	proto.RegisterType((*ProtoTransaction_VinType)(nil), "bchain.ProtoTransaction.VinType")
	proto.RegisterType((*ProtoTransaction_VoutType)(nil), "bchain.ProtoTransaction.VoutType")
	proto.RegisterType((*ProtoTransaction_WitnessAddressType)(nil), "bchain.ProtoTransaction.WitnessAddressType")
	proto.RegisterType((*ProtoTransaction_RangeAmountPairType)(nil), "bchain.ProtoTransaction.RangeAmountPairType")
	proto.RegisterType((*ProtoTransaction_AssetAllocationTupleType)(nil), "bchain.ProtoTransaction.AssetAllocationTupleType")
	proto.RegisterType((*ProtoTransaction_AssetAllocationType)(nil), "bchain.ProtoTransaction.AssetAllocationType")
}

func init() { proto.RegisterFile("tx.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 541 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x94, 0x4f, 0x8e, 0xda, 0x30,
	0x14, 0xc6, 0x65, 0x12, 0x20, 0x79, 0xc3, 0xfc, 0x91, 0x07, 0x55, 0x51, 0xc4, 0x22, 0x9d, 0x45,
	0x15, 0xa9, 0x15, 0x52, 0xa9, 0x7a, 0x00, 0xda, 0xcd, 0x48, 0x1d, 0x55, 0xc8, 0x41, 0xe9, 0xda,
	0x04, 0x0b, 0xac, 0x06, 0x9b, 0xc6, 0x0e, 0x65, 0xa4, 0x6e, 0xdb, 0x1b, 0xf4, 0x14, 0xbd, 0x5b,
	0xcf, 0x50, 0xd9, 0x18, 0x0a, 0x14, 0xba, 0x99, 0xe5, 0xfb, 0x9e, 0x9f, 0xbf, 0xdf, 0xfb, 0xac,
	0x04, 0x02, 0xbd, 0xee, 0x2f, 0x2b, 0xa9, 0x25, 0x6e, 0x4d, 0x8a, 0x39, 0xe5, 0xe2, 0xee, 0x67,
	0x08, 0x37, 0x23, 0xa3, 0x8c, 0x2b, 0x2a, 0x14, 0x2d, 0x34, 0x97, 0x02, 0x63, 0xf0, 0xc7, 0x6b,
	0x3e, 0x8d, 0x50, 0x82, 0xd2, 0x0e, 0xf1, 0xf5, 0x9a, 0x4f, 0xf1, 0x0d, 0x78, 0xf7, 0x6c, 0x1d,
	0x35, 0xac, 0xe4, 0xcd, 0xd9, 0x1a, 0xf7, 0x20, 0x7c, 0x57, 0xca, 0xe2, 0xb3, 0xe6, 0x0b, 0x16,
	0x79, 0x09, 0x4a, 0x7d, 0x12, 0x4e, 0xb6, 0x02, 0x8e, 0x21, 0x78, 0xd8, 0x36, 0xfd, 0x04, 0xa5,
	0x97, 0x24, 0xd8, 0xf5, 0x9e, 0x41, 0xeb, 0x9e, 0xf1, 0xd9, 0x5c, 0x47, 0x4d, 0xdb, 0x69, 0xcd,
	0x6d, 0x85, 0x07, 0xe0, 0xe5, 0x5c, 0x44, 0xad, 0xc4, 0x4b, 0x2f, 0x06, 0x49, 0x7f, 0x83, 0xd8,
	0x3f, 0xc6, 0xeb, 0xe7, 0x5c, 0x8c, 0x1f, 0x97, 0x8c, 0x78, 0x2b, 0x2e, 0xf0, 0x5b, 0xf0, 0x73,
	0x59, 0xeb, 0xa8, 0x6d, 0x87, 0x9e, 0x9f, 0x1f, 0x92, 0xb5, 0xb6, 0x53, 0xfe, 0x4a, 0xd6, 0x1a,
	0x47, 0xd0, 0xce, 0x59, 0xa5, 0xb8, 0x14, 0x51, 0x90, 0xa0, 0xb4, 0x49, 0xda, 0xab, 0x4d, 0x19,
	0xff, 0x42, 0xd0, 0x76, 0x0e, 0x66, 0x89, 0xf7, 0x92, 0x8b, 0x09, 0x55, 0xcc, 0x86, 0x11, 0x92,
	0xa0, 0x70, 0xf5, 0x2e, 0xa4, 0xc6, 0x5e, 0x48, 0xd8, 0xc1, 0x78, 0x76, 0xad, 0x8d, 0xd3, 0x1d,
	0x74, 0xb2, 0xa2, 0xe2, 0x4b, 0x9d, 0xf1, 0x99, 0x49, 0xd0, 0xb7, 0xe7, 0x3b, 0x6a, 0x4f, 0x33,
	0x3e, 0x19, 0xfb, 0x52, 0x33, 0x51, 0x30, 0x17, 0x49, 0xa0, 0x5c, 0x6d, 0x62, 0x1e, 0x4e, 0xa7,
	0x15, 0x53, 0x8a, 0x29, 0x1b, 0x4d, 0x48, 0x42, 0xba, 0x15, 0xe2, 0x6f, 0x10, 0x6c, 0x37, 0x33,
	0xb7, 0xe4, 0xb4, 0xac, 0x59, 0x46, 0xb5, 0x7b, 0xba, 0x60, 0xe5, 0x6a, 0xdc, 0x01, 0xf4, 0xd1,
	0xa2, 0x5e, 0x12, 0x24, 0x70, 0x0a, 0xd7, 0x1b, 0xa6, 0x51, 0x3d, 0xf9, 0xc0, 0x1e, 0x0d, 0x96,
	0x67, 0x07, 0xae, 0xd5, 0xa1, 0x7c, 0xe8, 0xee, 0x1f, 0xbb, 0xe7, 0x80, 0x3f, 0x71, 0x2d, 0x98,
	0x52, 0xee, 0x90, 0xe5, 0xd8, 0xcb, 0x16, 0x59, 0xc7, 0x6d, 0xb6, 0xf8, 0x05, 0x5c, 0xb9, 0xf3,
	0xa3, 0x4a, 0xce, 0x2a, 0xba, 0xb0, 0x48, 0x21, 0xb9, 0xfa, 0x7a, 0xa0, 0xc6, 0x3f, 0x10, 0xdc,
	0x12, 0x2a, 0x66, 0x6c, 0xb8, 0x90, 0xb5, 0xd0, 0x23, 0xca, 0x2b, 0x7b, 0x73, 0xb6, 0x9b, 0x77,
	0x7e, 0xd6, 0xe0, 0x62, 0xf0, 0xf2, 0xec, 0xb3, 0xff, 0x8b, 0xb7, 0x33, 0x73, 0xda, 0x41, 0x6c,
	0x8d, 0xc3, 0xd8, 0xe2, 0xef, 0x08, 0xa2, 0xa1, 0x52, 0x4c, 0x0f, 0xcb, 0x52, 0x16, 0xd4, 0x5c,
	0x39, 0xae, 0x97, 0x25, 0xb3, 0x34, 0x5d, 0x68, 0xda, 0x9e, 0xdb, 0xb2, 0x49, 0x4d, 0x71, 0x82,
	0xb1, 0xf1, 0x64, 0xc6, 0xf8, 0x37, 0x82, 0xdb, 0x63, 0x0e, 0x83, 0xc0, 0xa0, 0x7b, 0x0a, 0xcf,
	0xc5, 0xf2, 0xfa, 0xac, 0xe5, 0xb9, 0x9d, 0x48, 0x97, 0x9e, 0xe8, 0xe0, 0x25, 0xf4, 0x1e, 0xb8,
	0xd2, 0x19, 0x13, 0x53, 0x2e, 0x66, 0x7f, 0xbb, 0x9b, 0xe7, 0x31, 0x1b, 0x9a, 0x8f, 0xef, 0xd5,
	0x59, 0xbb, 0x13, 0x6f, 0x49, 0x7a, 0xe5, 0x7f, 0x6e, 0x9c, 0xb4, 0xec, 0x6f, 0xea, 0xcd, 0x9f,
	0x00, 0x00, 0x00, 0xff, 0xff, 0xd5, 0xc5, 0xc5, 0x32, 0xb2, 0x04, 0x00, 0x00,
}