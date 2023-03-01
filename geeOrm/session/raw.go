package session

import (
	"database/sql"
	"qitian/geeOrm/clause"
	"qitian/geeOrm/dialect"
	"qitian/geeOrm/log"
	"qitian/geeOrm/schema"
	"strings"
)

// session: 负责和数据库交互

type Session struct {
	db       *sql.DB         // 数据库引擎
	dialect  dialect.Dialect // 类型转换
	refTable *schema.Schema  // 表结构
	clause   clause.Clause   // 生成 sql 语句
	sql      strings.Builder // 传入 sql 语句
	sqlVars  []interface{}   // 参数
}

func New(db *sql.DB, dialect dialect.Dialect) *Session {
	return &Session{
		db:      db,
		dialect: dialect,
	}
}

// Clear 清除当前存储的sql语句和参数
func (s *Session) Clear() {
	s.sql.Reset()
	s.clause = clause.Clause{}
	s.sqlVars = nil
}

func (s *Session) DB() *sql.DB {
	return s.db
}

// Raw 写入sql语句和参数
func (s *Session) Raw(sql string, values ...interface{}) *Session {
	s.sql.WriteString(sql)
	s.sql.WriteString(" ")
	s.sqlVars = append(s.sqlVars, values...)
	return s
}

// Exec 执行sql语句
func (s *Session) Exec() (result sql.Result, err error) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	if result, err = s.DB().Exec(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}

// QueryRow 查询单行
func (s *Session) QueryRow() *sql.Row {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	return s.DB().QueryRow(s.sql.String(), s.sqlVars...)
}

// QueryRows 查询多行
func (s *Session) QueryRows() (rows *sql.Rows, err error) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	if rows, err = s.DB().Query(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}
