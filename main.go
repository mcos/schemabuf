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
	db := flag.String("db", "mysql", "the database type")
	host := flag.String("host", "localhost", "the database host")
	port := flag.Int("port", 3306, "the database port")
	user := flag.String("user", "root", "the database user")
	password := flag.String("password", "root", "the database password")
	schema := flag.String("schema", "db_name", "the database schema")

	flag.Parse()

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", *user, *password, *host, *port, *schema)
	conn, err := sql.Open(*db, connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	s, err := schemabuf.GenerateSchema(conn)

	fmt.Println(s)
}
