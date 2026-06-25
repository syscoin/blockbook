package db

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	vlq "github.com/bsm/go-vlq"
	"github.com/golang/glog"
	"github.com/juju/errors"
	gorocksdb "github.com/linxGnu/grocksdb"
	syscoinwire "github.com/syscoin/syscoinwire/syscoin/wire"
	"github.com/trezor/blockbook/bchain"
)

var AssetCache map[uint64]bchain.Asset
var SetupAssetCacheFirstTime bool = true
var assetCacheMu sync.RWMutex // SYSCOIN

const syscoinSYSXAssetGuid uint64 = 123456 // SYSCOIN

// SYSCOIN: SYSX is the consensus bridge asset. Its display metadata is
// canonical even if an older DB/cache row contains incomplete native metadata.
func builtinSYSXAsset(transactions uint32) *bchain.Asset {
	return &bchain.Asset{
		Transactions: transactions,
		AssetObj: syscoinwire.AssetType{
			Contract:  make([]byte, 20),
			Symbol:    []byte("SYSX"),
			Precision: 8,
		},
		MetaData: []byte("Syscoin Native Asset"),
	}
}

// SYSCOIN: indexing can continue with minimal metadata if NEVM metadata is
// temporarily unavailable; later asset cache updates can replace the display data.
func fallbackSyscoinAsset(guid uint64) *bchain.Asset {
	return &bchain.Asset{
		AssetObj: syscoinwire.AssetType{
			Symbol:    []byte(fmt.Sprintf("%d", guid)),
			Precision: 8,
		},
	}
}

func isFallbackSyscoinAsset(guid uint64, asset *bchain.Asset) bool {
	if asset == nil {
		return false
	}
	return string(asset.AssetObj.Symbol) == fmt.Sprintf("%d", guid) &&
		asset.AssetObj.Precision == 8 &&
		len(asset.AssetObj.Contract) == 0 &&
		len(asset.MetaData) == 0
}

func (d *RocksDB) fetchNEVMAssetDetails(guid uint64) (*bchain.Asset, error) {
	if d.chain == nil {
		return nil, errors.New("GetAsset: asset not found in DB")
	}
	fetcher, ok := d.chain.(nevmAssetFetcher)
	if !ok {
		return nil, errors.New("GetAsset: NEVM asset fetcher is not configured")
	}
	return fetcher.FetchNEVMAssetDetails(guid)
}

// GetTxAssetsCallback is called by GetTransactions/GetTxAssets for each found tx
type GetTxAssetsCallback func(txids []string) error

type syscoinAssetParser interface {
	PackAssetKey(assetGuid uint64, height uint32) []byte
	UnpackAssetKey(key []byte) (uint64, uint32)
	PackAssetTxIndex(txAsset *bchain.TxAsset) []byte
	UnpackAssetTxIndex(buf []byte) []*bchain.TxAssetIndex
	PackAsset(asset *bchain.Asset) ([]byte, error)
	UnpackAsset(buf []byte) (*bchain.Asset, error)
	GetAssetsMaskFromVersion(nVersion int32) bchain.AssetsMask
}

type nevmAssetFetcher interface {
	FetchNEVMAssetDetails(assetGuid uint64) (*bchain.Asset, error)
}

func (d *RocksDB) syscoinAssetParser() syscoinAssetParser {
	p, _ := d.chainParser.(syscoinAssetParser)
	return p
}

func packAssetGuid(guid uint64) []byte {
	key := make([]byte, vlq.MaxLen64)
	l := vlq.PutUint(key, guid)
	return key[:l]
}

func unpackAssetGuid(buf []byte) (uint64, int) {
	return vlq.Uint(buf)
}

func (d *RocksDB) ConnectAllocationInput(addrDesc *bchain.AddressDescriptor, height uint32, version int32, balanceAsset *bchain.AssetBalance, btxID []byte, assetInfo *bchain.AssetInfo, blockTxAssetAddresses bchain.TxAssetAddressMap, assets map[uint64]*bchain.Asset, txAssets bchain.TxAssetMap) error {
	dBAsset, err := d.GetAsset(assetInfo.AssetGuid, assets)
	if err != nil {
		glog.Warningf("ConnectAllocationInput using fallback asset %d: %v", assetInfo.AssetGuid, err)
		dBAsset = fallbackSyscoinAsset(assetInfo.AssetGuid)
	}
	counted := d.addToAssetsMap(txAssets, assetInfo.AssetGuid, btxID, version, height)
	if !counted {
		dBAsset.Transactions++
		assets[assetInfo.AssetGuid] = dBAsset
	}
	counted = d.addToAssetAddressMap(blockTxAssetAddresses, assetInfo.AssetGuid, btxID, addrDesc)
	if !counted {
		balanceAsset.Transfers++
	}
	balanceAsset.BalanceSat.Sub(balanceAsset.BalanceSat, assetInfo.ValueSat)
	if balanceAsset.BalanceSat.Sign() < 0 {
		balanceAsset.BalanceSat.SetInt64(0)
	}
	balanceAsset.SentSat.Add(balanceAsset.SentSat, assetInfo.ValueSat)
	return nil
}

func (d *RocksDB) ConnectAllocationOutput(addrDesc *bchain.AddressDescriptor, height uint32, balanceAsset *bchain.AssetBalance, version int32, btxID []byte, assetInfo *bchain.AssetInfo, blockTxAssetAddresses bchain.TxAssetAddressMap, assets map[uint64]*bchain.Asset, txAssets bchain.TxAssetMap, memo []byte) error {
	dBAsset, err := d.GetAsset(assetInfo.AssetGuid, assets)
	if err != nil {
		glog.Warningf("ConnectAllocationOutput using fallback asset %d: %v", assetInfo.AssetGuid, err)
		dBAsset = fallbackSyscoinAsset(assetInfo.AssetGuid)
	}
	counted := d.addToAssetsMap(txAssets, assetInfo.AssetGuid, btxID, version, height)
	if !counted {
		dBAsset.Transactions++
		assets[assetInfo.AssetGuid] = dBAsset
	}
	// asset guid + txid + address of output/input must match for counted to be true
	counted = d.addToAssetAddressMap(blockTxAssetAddresses, assetInfo.AssetGuid, btxID, addrDesc)
	if !counted {
		balanceAsset.Transfers++
	}
	balanceAsset.BalanceSat.Add(balanceAsset.BalanceSat, assetInfo.ValueSat)
	return nil
}

func (d *RocksDB) DisconnectAllocationOutput(addrDesc *bchain.AddressDescriptor, balanceAsset *bchain.AssetBalance, btxID []byte, assetInfo *bchain.AssetInfo, blockTxAssetAddresses bchain.TxAssetAddressMap, assets map[uint64]*bchain.Asset, assetFoundInTx func(asset uint64, btxID []byte) bool) error {
	dBAsset, err := d.GetAsset(assetInfo.AssetGuid, assets)
	if err != nil {
		glog.Warningf("DisconnectAllocationOutput using fallback asset %d: %v", assetInfo.AssetGuid, err)
		dBAsset = fallbackSyscoinAsset(assetInfo.AssetGuid)
	}
	balanceAsset.BalanceSat.Sub(balanceAsset.BalanceSat, assetInfo.ValueSat)
	if balanceAsset.BalanceSat.Sign() < 0 {
		balanceAsset.BalanceSat.SetInt64(0)
	}
	exists := assetFoundInTx(assetInfo.AssetGuid, btxID)
	if !exists {
		dBAsset.Transactions--
		if dBAsset.Transactions == 0 {
			// signals for removal from asset db
			dBAsset.AssetObj.TotalSupply = -1
		}
	}
	counted := d.addToAssetAddressMap(blockTxAssetAddresses, assetInfo.AssetGuid, btxID, addrDesc)
	if !counted {
		balanceAsset.Transfers--
	}
	assets[assetInfo.AssetGuid] = dBAsset
	return nil
}
func (d *RocksDB) DisconnectAllocationInput(addrDesc *bchain.AddressDescriptor, balanceAsset *bchain.AssetBalance, btxID []byte, assetInfo *bchain.AssetInfo, blockTxAssetAddresses bchain.TxAssetAddressMap, assets map[uint64]*bchain.Asset, assetFoundInTx func(asset uint64, btxID []byte) bool) error {
	dBAsset, err := d.GetAsset(assetInfo.AssetGuid, assets)
	if err != nil {
		glog.Warningf("DisconnectAllocationInput using fallback asset %d: %v", assetInfo.AssetGuid, err)
		dBAsset = fallbackSyscoinAsset(assetInfo.AssetGuid)
	}
	balanceAsset.SentSat.Sub(balanceAsset.SentSat, assetInfo.ValueSat)
	balanceAsset.BalanceSat.Add(balanceAsset.BalanceSat, assetInfo.ValueSat)
	if balanceAsset.SentSat.Sign() < 0 {
		balanceAsset.SentSat.SetInt64(0)
	}
	exists := assetFoundInTx(assetInfo.AssetGuid, btxID)
	if !exists {
		dBAsset.Transactions--
	}
	counted := d.addToAssetAddressMap(blockTxAssetAddresses, assetInfo.AssetGuid, btxID, addrDesc)
	if !counted {
		balanceAsset.Transfers--
	}
	assets[assetInfo.AssetGuid] = dBAsset
	return nil
}

func (d *RocksDB) SetupAssetCache() error {
	start := time.Now()
	assetCacheMu.Lock()
	defer assetCacheMu.Unlock()
	if AssetCache == nil {
		AssetCache = map[uint64]bchain.Asset{}
	}
	ro := gorocksdb.NewDefaultReadOptions()
	ro.SetFillCache(false)
	it := d.db.NewIteratorCF(d.ro, d.cfh[cfAssets])
	defer it.Close()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		assetKey, _ := unpackAssetGuid(it.Key().Data())
		data := it.Value().Data()

		parser := d.syscoinAssetParser()
		if parser == nil {
			return errors.New("SetupAssetCache: Syscoin asset parser is not configured")
		}
		assetDb, err := parser.UnpackAsset(data)
		if err != nil {
			glog.Errorf("Failed unpacking asset GUID %d, data (hex): %s, err: %v", assetKey, hex.EncodeToString(data), err)
			break
		}
		if assetKey == syscoinSYSXAssetGuid {
			assetDb = builtinSYSXAsset(assetDb.Transactions)
		}

		AssetCache[assetKey] = *assetDb
	}
	glog.Info("SetupAssetCache completed in ", time.Since(start))
	return nil
}

func (d *RocksDB) storeAssets(wb *gorocksdb.WriteBatch, assets map[uint64]*bchain.Asset) error {
	if assets == nil {
		return nil
	}
	assetCacheMu.Lock()
	defer assetCacheMu.Unlock()
	if AssetCache == nil {
		AssetCache = map[uint64]bchain.Asset{}
	}
	for guid, asset := range assets {
		if guid == syscoinSYSXAssetGuid {
			asset = builtinSYSXAsset(asset.Transactions)
		}
		AssetCache[guid] = *asset
		key := packAssetGuid(guid)
		// total supply of -1 signals asset to be removed from db - happens on disconnect of new asset
		if asset.AssetObj.TotalSupply == -1 {
			delete(AssetCache, guid)
			wb.DeleteCF(d.cfh[cfAssets], key)
		} else {
			parser := d.syscoinAssetParser()
			if parser == nil {
				return errors.New("storeAssets: Syscoin asset parser is not configured")
			}
			buf, err := parser.PackAsset(asset)
			if err != nil {
				return err
			}
			wb.PutCF(d.cfh[cfAssets], key, buf)
		}
	}
	return nil
}

func (d *RocksDB) GetAssetCache() *map[uint64]bchain.Asset {
	assetCacheMu.RLock()
	defer assetCacheMu.RUnlock()
	cache := make(map[uint64]bchain.Asset, len(AssetCache))
	for guid, asset := range AssetCache {
		cache[guid] = asset
	}
	return &cache
}

func (d *RocksDB) GetSetupAssetCacheFirstTime() bool {
	assetCacheMu.RLock()
	defer assetCacheMu.RUnlock()
	return SetupAssetCacheFirstTime
}

func (d *RocksDB) SetSetupAssetCacheFirstTime(cacheVal bool) {
	assetCacheMu.Lock()
	defer assetCacheMu.Unlock()
	SetupAssetCacheFirstTime = cacheVal
}

func (d *RocksDB) GetBaseAssetID(guid uint64) uint64 {
	return guid & 0xffffffff
}
func (d *RocksDB) GetNFTID(guid uint64) uint64 {
	return guid >> 32
}

func (d *RocksDB) GetAsset(guid uint64, assets map[uint64]*bchain.Asset) (*bchain.Asset, error) {
	var assetDb *bchain.Asset
	var assetL1 *bchain.Asset
	var ok bool
	var err error
	if assets != nil {
		if assetL1, ok = assets[guid]; ok {
			if guid == syscoinSYSXAssetGuid {
				return builtinSYSXAsset(assetL1.Transactions), nil
			}
			if isFallbackSyscoinAsset(guid, assetL1) {
				if assetDb, err = d.fetchNEVMAssetDetails(guid); err == nil {
					assetDb.Transactions = assetL1.Transactions
					assets[guid] = assetDb
					return assetDb, nil
				}
			}
			return assetL1, nil
		}
	}
	assetCacheMu.Lock()
	if AssetCache == nil {
		AssetCache = map[uint64]bchain.Asset{}
		// so it will store later in cache
		ok = false
	} else {
		var assetDbCache, ok = AssetCache[guid]
		if ok {
			if guid == syscoinSYSXAssetGuid {
				assetDb = builtinSYSXAsset(assetDbCache.Transactions)
				AssetCache[guid] = *assetDb
				assetCacheMu.Unlock()
				return assetDb, nil
			}
			if isFallbackSyscoinAsset(guid, &assetDbCache) {
				assetCacheMu.Unlock()
				if assetDb, err = d.fetchNEVMAssetDetails(guid); err == nil {
					assetDb.Transactions = assetDbCache.Transactions
					assetCacheMu.Lock()
					AssetCache[guid] = *assetDb
					assetCacheMu.Unlock()
					return assetDb, nil
				}
				return &assetDbCache, nil
			}
			assetCacheMu.Unlock()
			return &assetDbCache, nil
		}
	}
	assetCacheMu.Unlock()
	key := packAssetGuid(guid)
	val, err := d.db.GetCF(d.ro, d.cfh[cfAssets], key)
	if err != nil {
		return nil, err
	}
	defer val.Free()
	// nil data means the key was not found in DB
	if val.Data() == nil {
		if guid == syscoinSYSXAssetGuid {
			assetDb = builtinSYSXAsset(0)
			assetCacheMu.Lock()
			AssetCache[guid] = *assetDb
			assetCacheMu.Unlock()
			return assetDb, nil
		}
		assetDb, err := d.fetchNEVMAssetDetails(guid)
		if err != nil {
			return nil, err
		}

		assetCacheMu.Lock()
		AssetCache[guid] = *assetDb
		assetCacheMu.Unlock()
		return assetDb, nil
	}
	buf := val.Data()
	if len(buf) == 0 {
		return nil, errors.New("GetAsset: empty value in asset db")
	}
	parser := d.syscoinAssetParser()
	if parser == nil {
		return nil, errors.New("GetAsset: Syscoin asset parser is not configured")
	}
	assetDb, err = parser.UnpackAsset(buf)
	if err != nil {
		return nil, err
	}
	if guid == syscoinSYSXAssetGuid {
		assetDb = builtinSYSXAsset(assetDb.Transactions)
	}
	if isFallbackSyscoinAsset(guid, assetDb) {
		if fetched, fetchErr := d.fetchNEVMAssetDetails(guid); fetchErr == nil {
			fetched.Transactions = assetDb.Transactions
			assetDb = fetched
		}
	}
	// cache miss, add it, we also add it on storeAsset but on API queries we should not have to wait until a block
	// with this asset to store it in cache
	if !ok {
		assetCacheMu.Lock()
		AssetCache[guid] = *assetDb
		assetCacheMu.Unlock()
	}
	return assetDb, nil
}

func (d *RocksDB) storeTxAssets(wb *gorocksdb.WriteBatch, txassets bchain.TxAssetMap) error {
	for key, txAsset := range txassets {
		parser := d.syscoinAssetParser()
		if parser == nil {
			return errors.New("storeTxAssets: Syscoin asset parser is not configured")
		}
		buf := parser.PackAssetTxIndex(txAsset)
		wb.PutCF(d.cfh[cfTxAssets], []byte(key), buf)
	}
	return nil
}

// GetTxAssets finds all asset transactions for each asset
// Transaction are passed to callback function in the order from newest block to the oldest
func (d *RocksDB) GetTxAssets(assetGuid uint64, lower uint32, higher uint32, assetsBitMask bchain.AssetsMask, fn GetTxAssetsCallback) (err error) {
	parser := d.syscoinAssetParser()
	if parser == nil {
		return errors.New("GetTxAssets: Syscoin asset parser is not configured")
	}
	startKey := parser.PackAssetKey(assetGuid, higher)
	stopKey := parser.PackAssetKey(assetGuid, lower)
	it := d.db.NewIteratorCF(d.ro, d.cfh[cfTxAssets])
	defer it.Close()
	for it.Seek(startKey); it.Valid(); it.Next() {
		key := it.Key().Data()
		val := it.Value().Data()
		if bytes.Compare(key, stopKey) > 0 {
			break
		}
		txIndexes := parser.UnpackAssetTxIndex(val)
		if txIndexes != nil {
			txids := []string{}
			for _, txIndex := range txIndexes {
				mask := uint32(txIndex.Type)
				if (assetsBitMask == bchain.AllMask) || ((uint32(assetsBitMask) & mask) == mask) {
					txids = append(txids, hex.EncodeToString(txIndex.BtxID))
				}
			}
			if len(txids) > 0 {
				if err := fn(txids); err != nil {
					if _, ok := err.(*StopIteration); ok {
						return nil
					}
					return err
				}
			}
		}
	}
	return nil
}

// addToAssetsMap maintains mapping between assets and transactions in one block
// the return value is true if the tx was processed before, to not to count the tx multiple times
func (d *RocksDB) addToAssetsMap(txassets bchain.TxAssetMap, assetGuid uint64, btxID []byte, version int32, height uint32) bool {
	// check that the asset was already processed in this block
	// if not found, it has certainly not been counted
	parser := d.syscoinAssetParser()
	if parser == nil {
		return false
	}
	key := string(parser.PackAssetKey(assetGuid, height))
	at, found := txassets[key]
	if found {
		// if the tx is already in the slice
		for _, t := range at.Txs {
			if bytes.Equal(btxID, t.BtxID) {
				return true
			}
		}
	} else {
		at = &bchain.TxAsset{Txs: []*bchain.TxAssetIndex{}}
		txassets[key] = at
	}
	at.Txs = append(at.Txs, &bchain.TxAssetIndex{Type: parser.GetAssetsMaskFromVersion(version), BtxID: btxID})
	at.Height = height
	return false
}

// to control Transfer add/remove
func (d *RocksDB) addToAssetAddressMap(txassetAddresses bchain.TxAssetAddressMap, assetGuid uint64, btxID []byte, addrDesc *bchain.AddressDescriptor) bool {
	at, found := txassetAddresses[assetGuid]
	if found {
		// if the tx is already in the slice
		for _, t := range at.Txs {
			if bytes.Equal(btxID, t.BtxID) && bytes.Equal(*addrDesc, t.AddrDesc) {
				return true
			}
		}
	} else {
		at = &bchain.TxAssetAddress{Txs: []*bchain.TxAssetAddressIndex{}}
		txassetAddresses[assetGuid] = at
	}
	at.Txs = append(at.Txs, &bchain.TxAssetAddressIndex{AddrDesc: *addrDesc, BtxID: btxID})
	return false
}
