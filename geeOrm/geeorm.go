package geeOrm

import (
	"database/sql"
	"fmt"
	"qitian/geeOrm/dialect"
	"qitian/geeOrm/log"
	"qitian/geeOrm/session"
	"strings"
)

// geeorm: 负责和用户交互

// Engine 负责交互前的工作：连接/测试数据库；交互后的工作：关闭连接
type Engine struct {
	db      *sql.DB
	dialect dialect.Dialect
}

type txFunc func(*session.Session) (interface{}, error)

func NewEngine(driver, source string) (e *Engine, err error) {
	db, err := sql.Open(driver, source)
	if err != nil {
		log.Error(err)
		return
	}
	if err = db.Ping(); err != nil {
		return
	}

	dial, ok := dialect.GetDialect(driver)
	if !ok {
		log.Errorf("dialect %s Not Found", driver)
		return
	}
	e = &Engine{db: db, dialect: dial}
	log.Info("Connect database success")
	return
}

func (engine *Engine) Close() {
	if err := engine.db.Close(); err != nil {
		log.Error("Failed to close database")
	}
	log.Info("Close database success")
}

// NewSession 通过 Engine 建立 Session
func (engine *Engine) NewSession() *session.Session {
	return session.New(engine.db, engine.dialect)
}

func (engine *Engine) Transaction(f txFunc) (result interface{}, err error) {
	s := engine.NewSession()
	if err := s.Begin(); err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = s.Rollback()
			panic(p)
		} else if err != nil {
			_ = s.Rollback()
		} else {
			err = s.Commit()
		}
	}()
	return f(s)
}

// difference returns a - b 即: 新表 - 旧表 = 新增字段，旧表 - 新表 = 删除字段
// a: a表的字段  b: b表的字段
func difference(a []string, b []string) (diff []string) {
	mapB := map[string]bool{}
	for _, v := range b {
		mapB[v] = true
	}

	for _, v := range a {
		if _, exist := mapB[v]; !exist {
			diff = append(diff, v)
		}
	}
	return
}

// Migrate 迁移表
// value: 新表的结构体对象
func (engine *Engine) Migrate(value interface{}) error {
	// 利用事务 保持原子性
	_, err := engine.Transaction(func(s *session.Session) (result interface{}, err error) {
		if !s.Model(value).HasTable() {
			log.Infof("table %s doesn't exist", s.RefTable().Name)
			return nil, s.CreateTable()
		}
		// table 是新表的结构
		table := s.RefTable()
		// 查询旧表
		rows, _ := s.Raw(fmt.Sprintf("SELECT * FROM %s LIMIT 1", table.Name)).QueryRows()
		columns, _ := rows.Columns()
		// 增加的列
		addCols := difference(table.FieldNames, columns)
		// 删去的列
		delCols := difference(columns, table.FieldNames)
		log.Infof("added cols %v, deleted cols %v", addCols, delCols)

		// 在旧表中添加列
		for _, col := range addCols {
			f := table.GetField(col)
			sqlStr := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table.Name, f.Name, f.Type)
			if _, err = s.Raw(sqlStr).Exec(); err != nil {
				return
			}
		}

		if len(delCols) == 0 {
			return
		}
		// 只选旧表中指定列(即新表的所有列)，转移到新表，删除旧表，新表改名为旧表
		tmp := "tmp_" + table.Name
		fieldStr := strings.Join(table.FieldNames, ", ")
		s.Raw(fmt.Sprintf("CREATE TABLE %s AS SELECT %s from %s;", tmp, fieldStr, table.Name))
		s.Raw(fmt.Sprintf("DROP TABLE %s;", table.Name))
		s.Raw(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tmp, table.Name))
		_, err = s.Exec()
		return
	})
	return err
}
