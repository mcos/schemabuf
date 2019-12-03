package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/c5433137/schemabuf/schemabuf"
)

func main() {
	dbType := flag.String("db", "mysql", "the database type")
	host := flag.String("host", "localhost", "the database host")
	port := flag.Int("port", 3306, "the database port")
	user := flag.String("user", "root", "the database user")
	password := flag.String("password", "root", "the database password")
	schema := flag.String("schema", "db_name", "the database schema")
	packageName := flag.String("package", *schema, "the protocol buffer package. defaults to the database schema.")
	ignoreTableStr := flag.String("ignore_tables", "", "a comma spaced list of tables to ignore")
	genType := flag.String("gen_type", "default", "Currently supported golang,value for [default,golang]")

	flag.Parse()

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", *user, *password, *host, *port, *schema)
	db, err := sql.Open(*dbType, connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	ignoreTables := strings.Split(*ignoreTableStr, ",")

	s, err := schemabuf.GenerateSchema(db, *packageName, ignoreTables,*genType)

	if nil != err {
		log.Fatal(err)
	}

	if nil != s {
		fmt.Println(s)
	}
}
