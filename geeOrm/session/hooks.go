package session

import (
	"qitian/geeOrm/log"
	"reflect"
)

// Hooks constants
const (
	BeforeQuery  = "BeforeQuery"
	AfterQuery   = "AfterQuery"
	BeforeUpdate = "BeforeUpdate"
	AfterUpdate  = "AfterUpdate"
	BeforeDelete = "BeforeDelete"
	AfterDelete  = "AfterDelete"
	BeforeInsert = "BeforeInsert"
	AfterInsert  = "AfterInsert"
)

// CallMethod 调用注册的 method
// method 是对象拥有的方法名, value 是对象的地址
func (s *Session) CallMethod(method string, value interface{}) {
	// s.RefTable().Model 或 value 即当前会话正在操作的对象,
	// 使用 MethodByName 方法反射得到该对象的方法。
	fm := reflect.ValueOf(s.RefTable().Model).MethodByName(method)
	if value != nil {
		fm = reflect.ValueOf(value).MethodByName(method)
	}
	param := []reflect.Value{reflect.ValueOf(s)} // 每一个钩子的入参类型均是 *Session
	if fm.IsValid() {
		if v := fm.Call(param); len(v) != 0 {
			if err, ok := v[0].Interface().(error); ok {
				log.Error(err)
			}
		}
	}
	return
}
