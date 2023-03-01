package schema

import (
	"go/ast"
	"qitian/geeOrm/dialect"
	"reflect"
)

// 利用反射(reflect)完成结构体和数据库表结构的映射，包括表名、字段名、字段类型、字段 tag 等。

type Field struct {
	Name string // 字段名
	Type string // 类型
	Tag  string // 约束条件
}

// Schema 代表一张表
type Schema struct {
	Model      interface{}       // 被映射的对象
	Name       string            // 表名
	Fields     []*Field          // 字段名
	FieldNames []string          // 所有的字段名
	fieldMap   map[string]*Field // 映射 [字段名: *Field]
}

func (schema *Schema) GetField(name string) *Field {
	return schema.fieldMap[name]
}

// Parse 将任意的对象解析为 Schema 实例
func Parse(dest interface{}, d dialect.Dialect) *Schema {
	// reflect.Indirect() 获取指针指向的实例
	modelType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	schema := &Schema{
		Model:    dest,
		Name:     modelType.Name(), // 结构体的名称作为表名
		fieldMap: make(map[string]*Field),
	}

	// modelType.NumField() 获取结构体的字段数
	for i := 0; i < modelType.NumField(); i++ {
		p := modelType.Field(i)
		if !p.Anonymous && ast.IsExported(p.Name) {
			field := &Field{
				Name: p.Name,
				Type: d.DataTypeOf(reflect.Indirect(reflect.New(p.Type))),
			}
			// 额外的约束条件，比如主键之类的
			if v, ok := p.Tag.Lookup("geeorm"); ok {
				field.Tag = v
			}
			schema.Fields = append(schema.Fields, field)
			schema.FieldNames = append(schema.FieldNames, p.Name)
			schema.fieldMap[p.Name] = field
		}
	}
	return schema
}

// RecordValues 展开各字段的参数
func (schema *Schema) RecordValues(dest interface{}) []interface{} {
	destValue := reflect.Indirect(reflect.ValueOf(dest))
	var fieldValues []interface{}
	for _, field := range schema.Fields {
		fieldValues = append(fieldValues, destValue.FieldByName(field.Name).Interface())
	}
	return fieldValues
}
