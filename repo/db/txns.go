package db

import (
	"bytes"
	"database/sql"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"sync"
)

type TxnsDB struct {
	db   *sql.DB
	lock sync.RWMutex
}

func (t *TxnsDB) Put(txn *wire.MsgTx) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert into txns(txid, tx) values(?,?)")
	defer stmt.Close()
	if err != nil {
		tx.Rollback()
		return err
	}
	var buf bytes.Buffer
	txn.Serialize(&buf)
	_, err = stmt.Exec(txn.TxHash().String(), buf.Bytes())
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (t *TxnsDB) Get(txid chainhash.Hash) (*wire.MsgTx, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	stmt, err := t.db.Prepare("select tx from txns where txid=?")
	defer stmt.Close()
	var ret []byte
	err = stmt.QueryRow(txid.String()).Scan(&ret)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(ret)
	msgTx := wire.NewMsgTx(wire.TxVersion)
	msgTx.BtcDecode(r, 1)
	return msgTx, nil
}

func (t *TxnsDB) GetAll() ([]*wire.MsgTx, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	var ret []*wire.MsgTx
	stm := "select tx from txns"
	rows, err := t.db.Query(stm)
	if err != nil {
		return ret, err
	}
	defer rows.Close()
	for rows.Next() {
		var tx []byte
		if err := rows.Scan(&tx); err != nil {
			continue
		}
		r := bytes.NewReader(tx)
		msgTx := wire.NewMsgTx(wire.TxVersion)
		msgTx.BtcDecode(r, 1)
		ret = append(ret, msgTx)
	}
	return ret, nil
}

func (t *TxnsDB) Delete(txid *chainhash.Hash) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	_, err := t.db.Exec("delete from txns where txid=?", txid.String())
	if err != nil {
		return err
	}
	return nil
}
