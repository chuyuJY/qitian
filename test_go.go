package main

import (
	"fmt"
	"reflect"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	name string `geeorm:"primary"`
	age  int    `geeorm:"age"`
}

var (
	user1 = &User{"Tom", 18}
)

func main() {
	modelType := reflect.Indirect(reflect.ValueOf(&User{})).Type()
	//str := reflect.Indirect(reflect.ValueOf(user1)).Interface().(User)
	//str := reflect.ValueOf(user1).Interface().(*User)
	//modelKind := reflect.ValueOf(&User{}).Kind()	// ptr
	//modelKind := reflect.Indirect(reflect.ValueOf(&User{})).Kind()	// struct
	//fmt.Println(reflect.ValueOf(user1).Elem() == reflect.Indirect(reflect.ValueOf(user1))) // true
	fmt.Println(modelType.Field(0).Tag.Get("geeorm"))
}
