package clause

import "strings"

// Clause 组合 SQL 独立语句为完整句子
type Clause struct {
	sql     map[Type]string        // SQL 语句
	sqlVars map[Type][]interface{} // 参数
}

// Type SQL 语句种类
type Type int

const (
	INSERT Type = iota
	VALUES
	SELECT
	LIMIT
	WHERE
	ORDERBY
	UPDATE
	DELETE
	COUNT
)

// Set 按照SQL语句的Name和参数, 得到子句
func (c *Clause) Set(name Type, vars ...interface{}) {
	if c.sql == nil {
		c.sql = make(map[Type]string)
		c.sqlVars = make(map[Type][]interface{})
	}
	sql, vars := generators[name](vars...)
	c.sql[name] = sql
	c.sqlVars[name] = vars
}

// Build 按照传入类型的顺序，构造最终的SQL语句
func (c *Clause) Build(orders ...Type) (string, []interface{}) {
	var sqls []string
	var vars []interface{}
	for _, order := range orders {
		if sql, ok := c.sql[order]; ok {
			sqls = append(sqls, sql)
			vars = append(vars, c.sqlVars[order]...)
		}
	}
	return strings.Join(sqls, " "), vars
}
