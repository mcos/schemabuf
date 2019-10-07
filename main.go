package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"

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
	packageName := flag.String("package", *schema, "the protocol buffer package. defaults to the database schema.")
	ignoreTableStr := flag.String("ignore_tables", "", "a comma spaced list of tables to ignore")
	singularizeTblName := flag.Bool("singularize_table_name", true, "singularize table name (message name)")
	fieldCommentStr := flag.String("field_comment", ",", "field comment, format: comment_prefix,position")
	goPkgStr := flag.String("go_package", "", "value for `option go_package` option")

	flag.Parse()

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", *user, *password, *host, *port, *schema)
	db, err := sql.Open(*dbType, connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	ignoreTables := strings.Split(*ignoreTableStr, ",")
	cmtInfo := strings.Split(*fieldCommentStr, ",")
	genOptions := schemabuf.GenerationOptions{
		PkgName:              *packageName,
		SingularizeTblName:   *singularizeTblName,
		FieldCommentPrefix:   cmtInfo[0],
		FieldCommentPosition: cmtInfo[1],
		GoPackage:            *goPkgStr,
	}
	s, err := schemabuf.GenerateSchema(db, ignoreTables, genOptions)

	if nil != err {
		log.Fatal(err)
	}

	if nil != s {
		fmt.Println(s)
	}
}
