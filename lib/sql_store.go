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

package lib

import (
	"database/sql"
	"errors"
	"fmt"

	// Blank because SQL driver
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
TableConfig - The configuration fields for specifying the table labels of the SQL database target.
*/
type TableConfig struct {
	Name       string `json:"table" yaml:"table"`
	IDCol      string `json:"id_column" yaml:"id_column"`
	ContentCol string `json:"content_column" yaml:"content_column"`
}

/*
DefaultTableConfig - Default table configuration.
*/
func DefaultTableConfig() TableConfig {
	return TableConfig{
		Name:       "leaps_documents",
		IDCol:      "ID",
		ContentCol: "CONTENT",
	}
}

/*
SQLConfig - The configuration fields for an SQL document store solution.
*/
type SQLConfig struct {
	DSN         string      `json:"dsn" yaml:"dsn"`
	TableConfig TableConfig `json:"db_table" yaml:"db_table"`
}

/*
DefaultSQLConfig - A default SQL configuration.
*/
func DefaultSQLConfig() SQLConfig {
	return SQLConfig{
		DSN:         "",
		TableConfig: DefaultTableConfig(),
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
	createStmt *sql.Stmt
	updateStmt *sql.Stmt
	readStmt   *sql.Stmt
}

/*
Create - Create a new document in a database table.
*/
func (m *SQLStore) Create(id string, doc *Document) error {
	_, err := m.createStmt.Exec(id, doc.Content)
	return err
}

/*
Store - Store document in a database table.
*/
func (m *SQLStore) Store(id string, doc *Document) error {
	_, err := m.updateStmt.Exec(doc.Content, id)
	return err
}

/*
Fetch - Fetch document from a database table.
*/
func (m *SQLStore) Fetch(id string) (*Document, error) {
	var document Document
	document.ID = id

	err := m.readStmt.QueryRow(id).Scan(&document.Content)

	switch {
	case err == sql.ErrNoRows:
		return nil, errors.New("document ID was not found in table")
	case err != nil:
		return nil, err
	}

	return &document, nil
}

/*
GetSQLStore - Just a func that returns an SQLStore
*/
func GetSQLStore(config DocumentStoreConfig) (DocumentStore, error) {
	var db *sql.DB
	var createStr, updateStr, readStr string
	var create, update, read *sql.Stmt
	var err error

	if len(config.SQLConfig.DSN) == 0 {
		return nil, fmt.Errorf("attempted to connect to %v database without a valid DSN", config.Type)
	}

	db, err = sql.Open(config.Type, config.SQLConfig.DSN)
	if err != nil {
		return nil, err
	}

	/* Now we set up prepared statements. This ensures at initialization that we can successfully
	 * connect to the database.
	 */

	switch config.Type {
	case "postgres":
		createStr = "INSERT INTO %v (%v, %v) VALUES ($1, $2)"
		updateStr = "UPDATE %v SET %v = $1 WHERE %v = $2"
		readStr = "SELECT %v FROM %v WHERE %v = $1"
	default:
		createStr = "INSERT INTO %v (%v, %v) VALUES (?, ?)"
		updateStr = "UPDATE %v SET %v = ? WHERE %v = ?"
		readStr = "SELECT %v FROM %v WHERE %v = ?"
	}

	create, err = db.Prepare(fmt.Sprintf(createStr,
		config.SQLConfig.TableConfig.Name,
		config.SQLConfig.TableConfig.IDCol,
		config.SQLConfig.TableConfig.ContentCol,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare create statement: %v", err)
	}
	update, err = db.Prepare(fmt.Sprintf(updateStr,
		config.SQLConfig.TableConfig.Name,
		config.SQLConfig.TableConfig.ContentCol,
		config.SQLConfig.TableConfig.IDCol,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare update statement: %v", err)
	}
	read, err = db.Prepare(fmt.Sprintf(readStr,
		config.SQLConfig.TableConfig.ContentCol,
		config.SQLConfig.TableConfig.Name,
		config.SQLConfig.TableConfig.IDCol,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare get statement: %v", err)
	}

	return &SQLStore{
		db:         db,
		config:     config,
		createStmt: create,
		updateStmt: update,
		readStmt:   read,
	}, nil
}

/*--------------------------------------------------------------------------------------------------
 */
