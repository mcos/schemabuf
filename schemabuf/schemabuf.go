package schemabuf

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/chuckpreslar/inflect"
	"github.com/serenize/snaker"
)

const (
	// proto3 is a describing the proto3 syntax type.
	proto3 = "proto3"

	// indent represents the indentation amount for fields. the style guide suggests
	// two spaces
	indent = "  "
)

// GenerateSchema generates a protobuf schema from a database connection and a package name.
// A list of tables to ignore may also be supplied.
// The returned schema implements the `fmt.Stringer` interface, in order to generate a string
// representation of a protobuf schema.
// Do not rely on the structure of the Generated schema to provide any context about
// the protobuf types. The schema reflects the layout of a protobuf file and should be used
// to pipe the output of the `Schema.String()` to a file.
func GenerateSchema(db *sql.DB, pkg string, ignoreTables []string) (*Schema, error) {
	s := &Schema{}

	dbs, err := dbSchema(db)
	if nil != err {
		return nil, err
	}

	s.Syntax = proto3
	if "" != pkg {
		s.Package = pkg
	}

	cols, err := dbColumns(db, dbs)
	if nil != err {
		return nil, err
	}

	err = typesFromColumns(s, cols, ignoreTables)
	if nil != err {
		return nil, err
	}

	sort.Sort(s.Imports)
	sort.Sort(s.Messages)
	sort.Sort(s.Enums)

	return s, nil
}

// typesFromColumns creates the appropriate schema properties from a collection of column types.
func typesFromColumns(s *Schema, cols []Column, ignoreTables []string) error {
	messageMap := map[string]*Message{}
	ignoreMap := map[string]bool{}
	for _, ig := range ignoreTables {
		ignoreMap[ig] = true
	}

	for _, c := range cols {
		if _, ok := ignoreMap[c.TableName]; ok {
			continue
		}

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
	defer rows.Close()
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
	Imports  sort.StringSlice
	Messages MessageCollection
	Enums    EnumCollection
}

// MessageCollection represents a sortable collection of messages.
type MessageCollection []*Message

func (mc MessageCollection) Len() int {
	return len(mc)
}

func (mc MessageCollection) Less(i, j int) bool {
	return mc[i].Name < mc[j].Name
}

func (mc MessageCollection) Swap(i, j int) {
	mc[i], mc[j] = mc[j], mc[i]
}

// EnumCollection represents a sortable collection of enums.
type EnumCollection []*Enum

func (ec EnumCollection) Len() int {
	return len(ec)
}

func (ec EnumCollection) Less(i, j int) bool {
	return ec[i].Name < ec[j].Name
}

func (ec EnumCollection) Swap(i, j int) {
	ec[i], ec[j] = ec[j], ec[i]
}

// AppendImport adds an import to a schema if the specific import does not already exist in the schema.
func (s *Schema) AppendImport(imports string) {
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

// String returns a string representation of a Schema.
func (s *Schema) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(fmt.Sprintf("syntax = '%s';\n", s.Syntax))
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("package %s;\n", s.Package))
	buf.WriteString("\n")
	buf.WriteString("// ------------------------------------ \n")
	buf.WriteString("// Imports\n")
	buf.WriteString("// ------------------------------------ \n\n")

	for _, i := range s.Imports {
		buf.WriteString(fmt.Sprintf("import \"%s\";\n", i))
	}

	buf.WriteString("\n")
	buf.WriteString("// ------------------------------------ \n")
	buf.WriteString("// Messages\n")
	buf.WriteString("// ------------------------------------ \n\n")

	for _, m := range s.Messages {
		buf.WriteString(fmt.Sprintf("%s\n", m))
	}

	buf.WriteString("\n")
	buf.WriteString("// ------------------------------------ \n")
	buf.WriteString("// Enums\n")
	buf.WriteString("// ------------------------------------ \n\n")

	for _, e := range s.Enums {
		buf.WriteString(fmt.Sprintf("%s\n", e))
	}

	buf.WriteString("\n")

	return buf.String()
}

// Enum represents a protocol buffer enumerated type.
type Enum struct {
	Name   string
	Fields []EnumField
}

// String returns a string representation of an Enum.
func (e *Enum) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(fmt.Sprintf("enum %s {\n", e.Name))

	for _, f := range e.Fields {
		buf.WriteString(fmt.Sprintf("%s%s;\n", indent, f))
	}

	buf.WriteString("}\n")

	return buf.String()
}

// AppendField appends an EnumField to an Enum.
func (e *Enum) AppendField(ef EnumField) error {
	for _, f := range e.Fields {
		if f.Tag() == ef.Tag() {
			return fmt.Errorf("tag `%d` is already in use by field `%s`", ef.Tag(), f.Name())
		}
	}

	e.Fields = append(e.Fields, ef)

	return nil
}

// EnumField represents a field in an enumerated type.
type EnumField struct {
	name string
	tag  int
}

// NewEnumField constructs an EnumField type.
func NewEnumField(name string, tag int) EnumField {
	name = strings.ToUpper(name)

	re := regexp.MustCompile(`([^\w]+)`)
	name = re.ReplaceAllString(name, "_")

	return EnumField{name, tag}
}

// String returns a string representation of an Enum.
func (ef EnumField) String() string {
	return fmt.Sprintf("%s = %d", ef.name, ef.tag)
}

// Name returns the name of the enum field.
func (ef EnumField) Name() string {
	return ef.name
}

// Tag returns the identifier tag of the enum field.
func (ef EnumField) Tag() int {
	return ef.tag
}

// newEnumFromStrings creates an enum from a name and a slice of strings that represent the names of each field.
func newEnumFromStrings(name string, ss []string) (*Enum, error) {
	enum := &Enum{}
	enum.Name = name

	for i, s := range ss {
		err := enum.AppendField(NewEnumField(s, i))
		if nil != err {
			return nil, err
		}
	}

	return enum, nil
}

// Service represents a protocol buffer service.
// TODO: Implement this in a schema.
type Service struct{}

// Message represents a protocol buffer message.
type Message struct {
	Name   string
	Fields []MessageField
}

// String returns a string representation of a Message.
func (m Message) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("message %s {\n", m.Name))
	for _, f := range m.Fields {
		buf.WriteString(fmt.Sprintf("%s%s;\n", indent, f))
	}
	buf.WriteString("}\n")

	return buf.String()
}

// AppendField appends a message field to a message. If the tag of the message field is in use, an error will be returned.
func (m *Message) AppendField(mf MessageField) error {
	for _, f := range m.Fields {
		if f.Tag() == mf.Tag() {
			return fmt.Errorf("tag `%d` is already in use by field `%s`", mf.Tag(), f.Name)
		}
	}

	m.Fields = append(m.Fields, mf)

	return nil
}

// MessageField represents the field of a message.
type MessageField struct {
	Typ  string
	Name string
	tag  int
}

// NewMessageField creates a new message field.
func NewMessageField(typ, name string, tag int) MessageField {
	return MessageField{typ, name, tag}
}

// Tag returns the unique numbered tag of the message field.
func (f MessageField) Tag() int {
	return f.tag
}

// String returns a string representation of a message field.
func (f MessageField) String() string {
	return fmt.Sprintf("%s %s = %d", f.Typ, f.Name, f.tag)
}

// Column represents a database column.
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

// parseColumn parses a column and inserts the relevant fields in the Message. If an enumerated type is encountered, an Enum will
// be added to the Schema. Returns an error if an incompatible protobuf data type cannot be found for the database column type.
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
		s.AppendImport("google/protobuf/timestamp.proto")

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

	err := msg.AppendField(field)
	if nil != err {
		return err
	}

	return nil
}
