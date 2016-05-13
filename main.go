package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"

	"github.com/mcos/schemabuf/schemabuf"
)

func main() {
	dbType := flag.String("db", "mysql", "the database type")
	host := flag.String("host", "localhost", "the database host")
	port := flag.Int("port", 3306, "the database port")
	user := flag.String("user", "root", "the database user")
	password := flag.String("password", "root", "the database password")
	schema := flag.String("schema", "db_name", "the database schema")

	flag.Parse()

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", *user, *password, *host, *port, *schema)
	db, err := sql.Open(*dbType, connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	s, err := schemabuf.GenerateSchema(db)

	if nil != err {
		log.Fatal(err)
	}

	if nil != s {
		fmt.Println(s)
	}
}
