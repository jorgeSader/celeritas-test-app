package data

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	db2 "github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/mysql"
	"github.com/upper/db/v4/adapter/postgresql"
	"github.com/upper/db/v4/adapter/sqlite"
)

var db *sql.DB
var upper db2.Session

type Models struct {
	// any models inserted here(and in the New function)
	// are easily accessible throughout the entire application

}

func New(databasePool *sql.DB) Models {
	db = databasePool

	dbType := strings.ToLower(os.Getenv("DATABASE_TYPE"))

	switch dbType {

	case "mysql", "mariadb":
		upper, _ = mysql.New(databasePool)

	case "postgresql", "postgres":
		upper, _ = postgresql.New(databasePool)

	case "sqlite", "turso", "libsql":
		upper, _ = sqlite.New(databasePool)

	case "mongo", "mongodb":
		// TODO Add mongo/mongodb model

	default:
	}

	return Models{}
}

// GetInsertID extracts the integer ID from an up.ID value returned by an insert operation.
// TODO this assumes that all my IDs are ints. Some DBs, like Mongo use UUIDs instead.
func GetInsertID(i db2.ID) int {
	idType := fmt.Sprintf("%T", i)
	if idType == "int64" {
		return int(i.(int64))
	}
	return i.(int)
}
