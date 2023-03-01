package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

type Context struct {
	// origin objects
	W   http.ResponseWriter
	Req *http.Request

	// req info
	Path   string            // 其实就是 pattern
	Method string            // 请求方法
	Params map[string]string // 存储Path传递的参数

	// resp info
	StatusCode int // 响应码

	// middleware
	handlers []HandlerFunc // 存储中间件
	index    int           // 记录当前执行到第几个中间件

	// engine pointer
	engine *Engine
}

func NewContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		W:      w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

// 获取传递的参数
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// 获取Form的数据
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// 获取Query的数据
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.W.WriteHeader(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.W.Header().Set(key, value)
}

func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.W.Write([]byte(fmt.Sprintf(format, values...)))
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.W.Write(data)
}

func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.W)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.W, err.Error(), 500)
	}
}

func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.W, name, data); err != nil {
		c.String(500, err.Error())
	}
}

// middleware
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for c.index < s {
		c.handlers[c.index](c)
		c.index++
	}
}
