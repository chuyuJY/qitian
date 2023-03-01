package dialect

import "reflect"

// 为适配不同的数据库，映射数据类型和特定的 SQL 语句，创建 Dialect 层屏蔽数据库差异。

var dialectsMap = map[string]Dialect{}

type Dialect interface {
	DataTypeOf(typ reflect.Value) string                    // 用于将 Go 语言的类型转换为该数据库的数据类型
	TableExistSQL(tableName string) (string, []interface{}) // 返回某个表是否存在的 SQL 语句，参数是表名(table)，返回值为(查询语句, 参数)
}

// RegisterDialect 注册dialect实例
func RegisterDialect(name string, dialect Dialect) {
	dialectsMap[name] = dialect
}

// GetDialect 获取dialect实例
func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectsMap[name]
	return
}
