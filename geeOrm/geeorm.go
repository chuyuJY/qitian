package geeOrm

import (
	"database/sql"
	"qitian/geeOrm/dialect"
	"qitian/geeOrm/log"
	"qitian/geeOrm/session"
)

// geeorm: 负责和用户交互

// Engine 负责交互前的工作：连接/测试数据库；交互后的工作：关闭连接
type Engine struct {
	db      *sql.DB
	dialect dialect.Dialect
}

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
	log.Error("Close database success")
}

// NewSession 通过 Engine 建立 Session
func (engine *Engine) NewSession() *session.Session {
	return session.New(engine.db, engine.dialect)
}
