# schemabuf

[![GoDoc](https://godoc.org/github.com/mcos/schemabuf/schemabuf?status.svg)](https://godoc.org/github.com/mcos/schemabuf/schemabuf)

Generates a protobuf schema from your mysql database schema.

### Uses
#### Use from the command line:

`go install github.com/mcos/schemabuf`

```
$ schemabuf -h

Usage of schemabuf:
  -db string
        the database type (default "mysql")
  -host string
        the database host (default "localhost")
  -ignore_tables string
        a comma spaced list of tables to ignore
  -singularize_table_name bool
        singularize table name (message name) or not (default true)
  -package string
        the protocol buffer package. defaults to the database schema. (default "db_name")
  -password string
        the database password (default "root")
  -port int
        the database port (default 3306)
  -schema string
        the database schema (default "db_name")
  -field_comment string
        comment to field of message. Format: comment_prefix,position.
        Position could be "top" or "right".
        The comment will be generated with format: `comment_prefix:"field"`.
        Example: `@injected_tag db:"id"`, with "@injected_tag db:" is comment_prefix,
        and "id" is the field name of message.
  -go_package string
        Set value to option `option go_package="?";` when proto file was generated.
  -user string
        the database user (default "root")
```

```
$ schemabuf -host my.database.com -port 3307 -user foo -schema bar -package my_package -ignore_tables=billing,passwords > foobar.proto
```

#### Use as an imported library

```go
import "github.com/mcos/schemabuf"

func main() {
    connStr := config.get("dbConnStr")
    pkg := "my_package"

    db, err := sql.Open(*dbType, connStr)
    if err != nil {
        log.Fatal(err)
    }

    defer db.Close()

    s, err := schemabuf.GenerateSchema(db, pkg, nil)

	if nil != err {
		log.Fatal(err)
	}

	if nil != s {
		fmt.Println(s)
	}
}
```
