/*
Copyright (c) 2014 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, sub to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package leaplib

import (
	"database/sql"
	"errors"
	"fmt"
	// Blank because SQL driver
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

/*--------------------------------------------------------------------------------------------------
 */

type SQLConfig struct {
	DSN        string `json:"dsn"`
	Address    string `json:"db_target"`
	Parameters string `json:"db_params"`
}

func DefaultSQLConfig() SQLConfig {
	return SQLConfig{
		DSN:        "",
		Address:    "",
		Parameters: "",
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
SQLStore - A document store implementation for an SQL database.
*/
type SQLStore struct {
	config     DocumentStoreConfig
	db         *sql.DB
	getStmt    *sql.Stmt
	createStmt *sql.Stmt
	updateStmt *sql.Stmt
}

/*
Create - Create a new document in a database table.
*/
func (m *SQLStore) Create(id string, doc *Document) error {
	return nil
}

/*
Store - Store document in a database table.
*/
func (m *SQLStore) Store(id string, doc *Document) error {
	return nil
}

/*
Fetch - Fetch document from a database table.
*/
func (m *SQLStore) Fetch(id string) (*Document, error) {
	return nil, nil
}

/*
GetSQLStore - Just a func that returns an SQLStore
*/
func GetSQLStore(config DocumentStoreConfig) (DocumentStore, error) {
	var db *sql.DB
	var get, create, update *sql.Stmt
	var err error
	var dsn string

	if len(config.SQLConfig.DSN) > 0 {
		dsn = config.SQLConfig.DSN
	} else {
		if len(config.SQLConfig.Address) > 0 {
			credentials := config.Username
			if len(config.Password) > 0 {
				credentials = fmt.Sprintf("%v:%v", config.Username, config.Password)
			}
			dsn = config.SQLConfig.Address
			if len(credentials) > 0 {
				dsn = fmt.Sprintf("%v@%v", credentials, config.SQLConfig.Address)
			}
			if len(config.SQLConfig.Parameters) > 0 {
				dsn = fmt.Sprintf("%v?%v", dsn, config.SQLConfig.Parameters)
			}
		}
	}

	if len(dsn) == 0 {
		return nil, fmt.Errorf("attempted to connect to %v database without a valid config target", config.Type)
	}

	switch config.Type {
	case "mysql", "sqlite":
		db, err = sql.Open(config.Type, dsn)
		if err != nil {
			return nil, err
		}
	case "postgres":
		db, err = sql.Open("postgres", fmt.Sprintf("postgres://%v", dsn))
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unrecognised sql_store type")
	}

	switch config.Type {
	case "mysql":
		get, err = db.Prepare(fmt.Sprintf("SELECT %v, %v, %v FROM %v WHERE %v = ?"))
		if err != nil {
			return nil, errors.New("failed to prepare get statement")
		}
		create, err = db.Prepare(fmt.Sprintf("INSERT TODO"))
		if err != nil {
			return nil, errors.New("failed to prepare create statement")
		}
		update, err = db.Prepare(fmt.Sprintf("UPDATE %v SET %v = $1 WHERE %v = $2"))
		if err != nil {
			return nil, errors.New("failed to prepare update statement")
		}
	}

	return &SQLStore{db: db, config: config, getStmt: get, createStmt: create, updateStmt: update}, nil
}

/*--------------------------------------------------------------------------------------------------
 */
