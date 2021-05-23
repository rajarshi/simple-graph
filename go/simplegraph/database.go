package simplegraph

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	SQLITE                  = "sqlite3"
	WITH_FOREIGN_KEY_PRAGMA = "%s?_foreign_keys=true"
)

func resolveDbFileReference(names ...string) (string, error) {
	args := len(names)
	switch args {
	case 1:
		return fmt.Sprintf(WITH_FOREIGN_KEY_PRAGMA, names[0]), nil
	case 2:
		return fmt.Sprintf(WITH_FOREIGN_KEY_PRAGMA, filepath.Join(names[0], names[1])), nil
	default:
		return "", errors.New("invalid database file reference")
	}
}

func evaluate(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func Initialize(database ...string) {
	init := func(db *sql.DB) error {
		for _, statement := range strings.Split(Schema, ";") {
			sql := strings.TrimSpace(statement)
			if len(sql) > 0 {
				stmt, err := db.Prepare(sql)
				evaluate(err)
				stmt.Exec()
			}
		}
		return nil
	}

	dbReference, err := resolveDbFileReference(database...)
	evaluate(err)
	db, dbErr := sql.Open(SQLITE, dbReference)
	evaluate(dbErr)
	defer db.Close()
	init(db)
}

func insert(node string, database ...string) (int64, error) {
	ins := func(db *sql.DB) (sql.Result, error) {
		stmt, stmtErr := db.Prepare(InsertNode)
		evaluate(stmtErr)
		return stmt.Exec(node)
	}

	dbReference, err := resolveDbFileReference(database...)
	evaluate(err)
	db, dbErr := sql.Open(SQLITE, dbReference)
	evaluate(dbErr)
	defer db.Close()
	in, inErr := ins(db)
	if inErr != nil {
		return 0, inErr
	}
	return in.RowsAffected()
}

func AddNodeAndId(node []byte, identifier string, database ...string) (int64, error) {
	closingBraceIdx := bytes.LastIndexByte(node, '}')
	if closingBraceIdx > 0 {
		addId := []byte(fmt.Sprintf(", \"id\": %q", identifier))
		node = append(node[:closingBraceIdx], addId...)
		node = append(node, '}')
	}
	return insert(string(node), database...)
}

func AddNode(node []byte, database ...string) (int64, error) {
	return insert(string(node), database...)
}

func ConnectNodesWithProperties(sourceId string, targetId string, properties []byte, database ...string) (int64, error) {
	connect := func(db *sql.DB) (sql.Result, error) {
		stmt, stmtErr := db.Prepare(InsertEdge)
		evaluate(stmtErr)
		return stmt.Exec(sourceId, targetId, string(properties))
	}

	dbReference, err := resolveDbFileReference(database...)
	evaluate(err)
	db, dbErr := sql.Open(SQLITE, dbReference)
	evaluate(dbErr)
	defer db.Close()
	cx, cxErr := connect(db)
	if cxErr != nil {
		return 0, cxErr
	}
	return cx.RowsAffected()
}

func ConnectNodes(sourceId string, targetId string, database ...string) (int64, error) {
	return ConnectNodesWithProperties(sourceId, targetId, []byte(`{}`), database...)
}

func RemoveNode(identifier string, database ...string) bool {
	delete := func(db *sql.DB) bool {
		edgeStmt, edgeErr := db.Prepare(DeleteEdge)
		evaluate(edgeErr)
		nodeStmt, nodeErr := db.Prepare(DeleteNode)
		evaluate(nodeErr)
		tx, txErr := db.Begin()
		evaluate(txErr)

		var err error
		_, err = tx.Stmt(edgeStmt).Exec(identifier, identifier)
		if err != nil {
			tx.Rollback()
			return false
		}
		_, err = tx.Stmt(nodeStmt).Exec(identifier)
		if err != nil {
			tx.Rollback()
			return false
		}
		tx.Commit()
		return true
	}

	dbReference, err := resolveDbFileReference(database...)
	evaluate(err)
	db, dbErr := sql.Open(SQLITE, dbReference)
	evaluate(dbErr)
	defer db.Close()
	return delete(db)
}

func FindNode(identifier string, database ...string) (string, error) {
	find := func(db *sql.DB) (string, error) {
		stmt, err := db.Prepare(SearchNodeById)
		evaluate(err)
		defer stmt.Close()
		var body string
		err = stmt.QueryRow(identifier).Scan(&body)
		if err == sql.ErrNoRows {
			return "", err
		}
		evaluate(err)
		return body, nil
	}

	dbReference, err := resolveDbFileReference(database...)
	evaluate(err)
	db, dbErr := sql.Open(SQLITE, dbReference)
	evaluate(dbErr)
	defer db.Close()
	return find(db)
}
