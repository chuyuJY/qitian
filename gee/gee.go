package gee

import (
	"html/template"
	"net/http"
	"strings"
)

// HandlerFunc 提供给框架用户的，用来定义路由映射的处理方法
type HandlerFunc func(c *Context)

type Engine struct {
	*RouteGroup
	router        *router
	groups        []*RouteGroup      // store all groups
	htmlTemplates *template.Template // for html render	将所有的模板加载进内存
	funcMap       template.FuncMap   // for html render	所有的自定义模板渲染函数
}

func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouteGroup = &RouteGroup{engine: engine}
	engine.groups = []*RouteGroup{engine.RouteGroup}
	return engine
}

func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	engine.router.addRoute(method, pattern, handler)
}

// pattern 其实就是路径
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}

func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

// 实现 Handler 接口
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 先执行中间件
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := NewContext(w, req)
	c.handlers = middlewares
	c.engine = engine
	engine.router.handle(c)
}

func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}
