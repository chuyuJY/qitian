package main

import (
	"fmt"
	"reflect"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Name string `geeorm:"primary"`
	age  int    `geeorm:"age"`
}

var (
	user1 = &User{"Tom", 18}
)

func main() {
	// 通过反射，修改结构体对象的值
	//sValue := reflect.ValueOf(user1).Elem()
	//sValue.FieldByName("age").SetInt(12)          // 报错，因为是私有对象
	//sValue.FieldByName("Name").SetString("chuyu") // 正常
	//fmt.Println(user1)
	// TODO: 待命名
	//modelType := reflect.Indirect(reflect.ValueOf(&User{})).Type()
	//str := reflect.Indirect(reflect.ValueOf(user1)).Interface().(User)
	//str := reflect.ValueOf(user1).Interface().(*User)
	//modelKind := reflect.ValueOf(&User{}).Kind()	// ptr
	//modelKind := reflect.Indirect(reflect.ValueOf(&User{})).Kind()	// struct
	//fmt.Println(reflect.ValueOf(user1).Elem() == reflect.Indirect(reflect.ValueOf(user1))) // true
	//fmt.Println(modelType.Field(0).Tag.Get("geeorm"))
	// 测试 切片
	users := []User{}
	destSlice := reflect.Indirect(reflect.ValueOf(&users))
	//fmt.Println(destSlice.Type())        // []main.User
	//fmt.Println(destSlice.Type().Elem()) // main.User
	destType := destSlice.Type().Elem()
	dest := reflect.New(destType).Elem() // 构造 destType 的 reflect.Value 对象
	//fmt.Println(dest.Interface() == User{}) // true
	dest.FieldByName("Name").SetString("chuyu")                // 修改 Name 字段的值, {chuyu 0}
	n := dest.FieldByName("Name").Addr().Interface().(*string) // n := Name 字段的地址
	*n = "hello"                                               // 另一种方法, 修改 Name 字段的值, {hello 0}
	//fmt.Println(dest)
	destSlice.Set(reflect.Append(destSlice, dest)) // 向切片中，添加对象元素, [{hello 0}]
	fmt.Println(destSlice)
}
