package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/chuckpreslar/inflect"
	_ "github.com/go-sql-driver/mysql"
	"github.com/serenize/snaker"
)

const (
	// Proto3 is a string describing the proto3 syntax type.
	Proto3 = "proto3"
)

// GenerateSchema generates a protobuf schema from a database connection.
func GenerateSchema(db *sql.DB) (*Schema, error) {
	s := &Schema{}

	dbs, err := dbSchema(db)
	if nil != err {
		return nil, err
	}

	s.Syntax = Proto3
	s.Package = dbs

	cols, err := dbColumns(db, dbs)
	if nil != err {
		return nil, err
	}

	typesFromColumns(s, cols)
	if nil != err {
		return nil, err
	}

	return s, nil
}

// typesFromColumns creates the appropriate schema properties from a collection of column types.
func typesFromColumns(s *Schema, cols []Column) error {
	messageMap := map[string]*Message{}

	for _, c := range cols {
		messageName := snaker.SnakeToCamel(c.TableName)
		messageName = inflect.Singularize(messageName)

		msg, ok := messageMap[messageName]
		if !ok {
			messageMap[messageName] = &Message{Name: messageName}
			msg = messageMap[messageName]
		}

		err := parseColumn(s, msg, c)
		if nil != err {
			return err
		}
	}

	for _, v := range messageMap {
		s.Messages = append(s.Messages, v)
	}

	return nil
}

func dbSchema(db *sql.DB) (string, error) {
	var schema string

	err := db.QueryRow("SELECT SCHEMA()").Scan(&schema)

	return schema, err
}

func dbColumns(db *sql.DB, schema string) ([]Column, error) {
	q := "SELECT TABLE_NAME, COLUMN_NAME, IS_NULLABLE, DATA_TYPE, " +
		"CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE, COLUMN_TYPE " +
		"FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME, ORDINAL_POSITION"

	rows, err := db.Query(q, schema)
	if nil != err {
		return nil, err
	}

	cols := []Column{}

	for rows.Next() {
		cs := Column{}
		err := rows.Scan(&cs.TableName, &cs.ColumnName, &cs.IsNullable, &cs.DataType,
			&cs.CharacterMaximumLength, &cs.NumericPrecision, &cs.NumericScale, &cs.ColumnType)
		if err != nil {
			log.Fatal(err)
		}

		cols = append(cols, cs)
	}
	if err := rows.Err(); nil != err {
		return nil, err
	}

	return cols, nil
}

// Schema is a representation of a protobuf schema.
type Schema struct {
	Syntax   string
	Package  string
	Imports  []string
	Messages []*Message
	Enums    []*Enum
}

type MessageCollection []*Message

type EnumCollection []*Enum

func (s *Schema) AddImport(imports string) {
	shouldAdd := true
	for _, si := range s.Imports {
		if si == imports {
			shouldAdd = false
			break
		}
	}

	if shouldAdd {
		s.Imports = append(s.Imports, imports)
	}

}

func (s *Schema) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(fmt.Sprintf("syntax = '%s';\n", s.Syntax))
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("package '%s';\n", s.Package))
	buf.WriteString("\n")
	buf.WriteString("// Imports")
	buf.WriteString("\n\n")
	for _, i := range s.Imports {
		buf.WriteString(fmt.Sprintf("import \"%s\";\n", i))
	}
	buf.WriteString("\n")
	buf.WriteString("// Messages")
	buf.WriteString("\n\n")
	for _, m := range s.Messages {
		buf.WriteString(fmt.Sprintf("%s\n", m))
	}
	buf.WriteString("\n")
	buf.WriteString("// Enums")
	buf.WriteString("\n\n")
	for _, e := range s.Enums {
		buf.WriteString(fmt.Sprintf("%s\n", e))
	}
	buf.WriteString("\n")

	return buf.String()
}

type Enum struct {
	Name   string
	Fields []EnumField
}

func (e *Enum) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(fmt.Sprintf("enum %s {\n", e.Name))
	for _, f := range e.Fields {
		buf.WriteString(fmt.Sprintf("%s%s;\n", "  ", f)) // two space indentation
	}
	buf.WriteString("}\n")

	return buf.String()
}

func (e *Enum) AddField(ef EnumField) error {
	for _, f := range e.Fields {
		if f.Tag() == ef.Tag() {
			return fmt.Errorf("tag `%d` is already in use by field `%s`", ef.Tag(), f.Name)
		}
	}

	e.Fields = append(e.Fields, ef)

	return nil
}

type EnumField struct {
	name string
	tag  int
}

func NewEnumField(name string, tag int) EnumField {
	name = strings.ToUpper(name)

	re := regexp.MustCompile(`([^\w]+)`)
	name = re.ReplaceAllString(name, "_")

	return EnumField{name, tag}
}

func (ef EnumField) String() string {
	return fmt.Sprintf("%s = %d", ef.name, ef.tag)
}

func (ef EnumField) Name() string {
	return ef.name
}

func (ef EnumField) Tag() int {
	return ef.tag
}

func newEnumFromStrings(name string, ss []string) (*Enum, error) {
	enum := &Enum{}
	enum.Name = name

	for i, s := range ss {
		err := enum.AddField(NewEnumField(s, i+1))
		if nil != err {
			return nil, err
		}
	}

	return enum, nil
}

type Service struct{}

type Column struct {
	TableName              string
	ColumnName             string
	IsNullable             string
	DataType               string
	CharacterMaximumLength sql.NullInt64
	NumericPrecision       sql.NullInt64
	NumericScale           sql.NullInt64
	ColumnType             string
}

type Message struct {
	Name   string
	Fields []MessageField
}

func (m Message) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("message %s {\n", m.Name))
	for _, f := range m.Fields {
		buf.WriteString(fmt.Sprintf("%s%s;\n", "  ", f)) // two space indentation
	}
	buf.WriteString("}\n")

	return buf.String()
}

func (m *Message) AddField(mf MessageField) error {
	for _, f := range m.Fields {
		if f.Tag() == mf.Tag() {
			return fmt.Errorf("tag `%d` is already in use by field `%s`", mf.Tag(), f.Name)
		}
	}

	m.Fields = append(m.Fields, mf)

	return nil
}

type MessageField struct {
	Typ  string
	Name string
	tag  int
}

func NewMessageField(typ, name string, tag int) MessageField {
	return MessageField{typ, name, tag}
}

// Tag returns the unique numbered tag of the message field.
func (f MessageField) Tag() int {
	return f.tag
}

func (f MessageField) String() string {
	return fmt.Sprintf("%s %s = %d", f.Typ, f.Name, f.tag)
}

func parseColumn(s *Schema, msg *Message, col Column) error {
	typ := strings.ToLower(col.DataType)
	var fieldType string

	switch typ {
	case "char", "varchar", "text", "longtext", "mediumtext", "tinytext":
		fieldType = "string"
	case "enum", "set":
		// Parse c.ColumnType to get the enum list
		enumList := regexp.MustCompile(`[enum|set]\((.+?)\)`).FindStringSubmatch(col.ColumnType)
		enums := strings.FieldsFunc(enumList[1], func(c rune) bool {
			cs := string(c)
			return "," == cs || "'" == cs
		})

		enumName := inflect.Singularize(snaker.SnakeToCamel(col.TableName)) + snaker.SnakeToCamel(col.ColumnName)
		enum, err := newEnumFromStrings(enumName, enums)
		if nil != err {
			return err
		}

		s.Enums = append(s.Enums, enum)

		fieldType = enumName
	case "blob", "mediumblob", "longblob", "varbinary", "binary":
		fieldType = "bytes"
	case "date", "time", "datetime", "timestamp":
		s.AddImport("google/protobuf/timestamp.proto")

		fieldType = "google.protobuf.Timestamp"
	case "tinyint", "bool":
		fieldType = "bool"
	case "smallint", "int", "mediumint", "bigint":
		fieldType = "int32"
	case "float", "decimal", "double":
		fieldType = "float"
	}

	if "" == fieldType {
		return fmt.Errorf("no compatible protobuf type found for `%s`. column: `%s`.`%s`", col.DataType, col.TableName, col.ColumnName)
	}

	field := NewMessageField(fieldType, col.ColumnName, len(msg.Fields)+1)

	err := msg.AddField(field)
	if nil != err {
		return err
	}

	return nil
}

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

	s, err := GenerateSchema(conn)

	fmt.Println(s)
}
