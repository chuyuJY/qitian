package gee

import (
	"log"
	"net/http"
	"path"
	"strings"
)

type router struct {
	roots    map[string]*node // 每种请求类型单独建一颗trie
	handlers map[string]HandlerFunc
}

// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 解析 pattern --> parts  只有一个 *xxx 可以被添加
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")

	parts := []string{}
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	log.Printf("Route %4s - %s", method, pattern)
	parts := parsePattern(pattern)

	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parts, 0)

	key := method + "-" + pattern
	r.handlers[key] = handler
}

func (r *router) getRoute(method string, pattern string) (*node, map[string]string) {
	searchParts := parsePattern(pattern)
	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}

	params := map[string]string{}
	target := root.search(searchParts, 0)
	if target != nil {
		// 解析 : 和 * 两种匹配符的参数
		parts := parsePattern(target.pattern)
		for index, part := range parts {
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			} else if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return target, params
	}
	return nil, nil
}

// 处理请求, 路由到真正处理的方法(handler)
func (r *router) handle(c *Context) {
	target, params := r.getRoute(c.Method, c.Path)
	if target != nil {
		c.Params = params
		key := c.Method + "-" + target.pattern
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	// 执行当前的函数列表 [middlewares..., handler]
	c.Next()
}

// 路由组
type RouteGroup struct {
	prefix      string
	middlewares []HandlerFunc // support middlewares
	parent      *RouteGroup   // support nesting
	engine      *Engine       // all groups share a Engine instance
}

// Group is defined to create a new RouterGroup
// remember all groups share the same Engine instance
func (group *RouteGroup) Group(prefix string) *RouteGroup {
	engine := group.engine
	newGroup := &RouteGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (group *RouteGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	// log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

func (group *RouteGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

func (group *RouteGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// 添加middleware
func (group *RouteGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

//	用户可以将磁盘上的某个文件夹root映射到路由relativePath
func (group *RouteGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	group.GET(urlPattern, handler)
}

// 添加静态资源方法
func (group *RouteGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		// 寻找资源并进行权限检查
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(c.W, c.Req)
	}
}
