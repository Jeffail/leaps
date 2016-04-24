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

package store

import (
	"database/sql"
	"errors"
	"fmt"

	// Blank because SQL driver
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

//--------------------------------------------------------------------------------------------------

// TableConfig - Fields for specifying the table labels of the SQL database target.
type TableConfig struct {
	Name       string `json:"table" yaml:"table"`
	IDCol      string `json:"id_column" yaml:"id_column"`
	ContentCol string `json:"content_column" yaml:"content_column"`
}

// NewTableConfig - Default table configuration.
func NewTableConfig() TableConfig {
	return TableConfig{
		Name:       "leaps_documents",
		IDCol:      "ID",
		ContentCol: "CONTENT",
	}
}

// SQLConfig - The configuration fields for an SQL document store solution.
type SQLConfig struct {
	DSN         string      `json:"dsn" yaml:"dsn"`
	TableConfig TableConfig `json:"db_table" yaml:"db_table"`
}

// NewSQLConfig - A default SQL configuration.
func NewSQLConfig() SQLConfig {
	return SQLConfig{
		DSN:         "",
		TableConfig: NewTableConfig(),
	}
}

/*--------------------------------------------------------------------------------------------------
 */

// SQL - A document store implementation for an SQL database.
type SQL struct {
	config     SQLConfig
	db         *sql.DB
	createStmt *sql.Stmt
	updateStmt *sql.Stmt
	readStmt   *sql.Stmt
}

/*
NewMySQL - Returns an SQL store type for connecting to MySQL databases.

DSN Should be of the format:
[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
*/
func NewMySQL(config SQLConfig) (Type, error) {
	return newSQL(mysql, config)
}

/*
NewPostgreSQL - Returns an SQL store type for connecting to PostgreSQL databases.

DSN Should be of the format:
postgresql://[user[:password]@][netloc][:port][/dbname][?param1=value1&...]
*/
func NewPostgreSQL(config SQLConfig) (Type, error) {
	return newSQL(postgres, config)
}

//--------------------------------------------------------------------------------------------------

// Supported SQL DB Types.
const (
	postgres = iota
	mysql    = iota
)

// SQL Type errors.
var (
	ErrMissingDSN          = errors.New("Missing DSN")
	ErrUnrecognizedSQLType = errors.New("SQL Type not recognized")
)

func newSQL(dbType int, config SQLConfig) (Type, error) {
	var (
		db                            *sql.DB
		createStr, updateStr, readStr string
		create, update, read          *sql.Stmt
		err                           error
	)
	if len(config.DSN) == 0 {
		return nil, ErrMissingDSN
	}

	/* Now we set up prepared statements. This ensures at initialization that we can successfully
	 * connect to the database.
	 */

	switch dbType {
	case postgres:
		db, err = sql.Open("postgres", config.DSN)
		createStr = "INSERT INTO %v (%v, %v) VALUES ($1, $2)"
		updateStr = "UPDATE %v SET %v = $1 WHERE %v = $2"
		readStr = "SELECT %v FROM %v WHERE %v = $1"
	case mysql:
		db, err = sql.Open("mysql", config.DSN)
		createStr = "INSERT INTO %v (%v, %v) VALUES (?, ?)"
		updateStr = "UPDATE %v SET %v = ? WHERE %v = ?"
		readStr = "SELECT %v FROM %v WHERE %v = ?"
	default:
		return nil, ErrUnrecognizedSQLType
	}
	if err != nil {
		return nil, err
	}

	create, err = db.Prepare(fmt.Sprintf(createStr,
		config.TableConfig.Name,
		config.TableConfig.IDCol,
		config.TableConfig.ContentCol,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare create statement: %v", err)
	}
	update, err = db.Prepare(fmt.Sprintf(updateStr,
		config.TableConfig.Name,
		config.TableConfig.ContentCol,
		config.TableConfig.IDCol,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare update statement: %v", err)
	}
	read, err = db.Prepare(fmt.Sprintf(readStr,
		config.TableConfig.ContentCol,
		config.TableConfig.Name,
		config.TableConfig.IDCol,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare get statement: %v", err)
	}

	return &SQL{
		db:         db,
		config:     config,
		createStmt: create,
		updateStmt: update,
		readStmt:   read,
	}, nil
}

//--------------------------------------------------------------------------------------------------

// Create - Create a new document in a database table.
func (m *SQL) Create(doc Document) error {
	_, err := m.createStmt.Exec(doc.ID, doc.Content)
	return err
}

// Update - Update document in a database table.
func (m *SQL) Update(doc Document) error {
	_, err := m.updateStmt.Exec(doc.Content, doc.ID)
	return err
}

// Read - Read document from a database table.
func (m *SQL) Read(id string) (Document, error) {
	var document Document
	document.ID = id

	err := m.readStmt.QueryRow(id).Scan(&document.Content)

	switch {
	case err == sql.ErrNoRows:
		return Document{}, ErrDocumentNotExist
	case err != nil:
		return Document{}, err
	}
	return document, nil
}

//--------------------------------------------------------------------------------------------------
