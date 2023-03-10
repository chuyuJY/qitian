package geerpc

import (
	"fmt"
	"html/template"
	"net/http"
)

const debugText = `<html>
	<body>
	<title>GeeRPC Services</title>
	{{range .}}
	<hr>
	Service {{.Name}}
	<hr>
		<table>
		<th align=center>Method</th><th align=center>Calls</th>
		{{range $name, $mtype := .Method}}
			<tr>
			<td align=left font=fixed>{{$name}}({{$mtype.ArgType}}, {{$mtype.ReplyType}}) error</td>
			<td align=center>{{$mtype.NumCalls}}</td>
			</tr>
		{{end}}
		</table>
	{{end}}
	</body>
	</html>`

var debug = template.Must(template.New("RPC debug").Parse(debugText))

type debugHTTP struct {
	*Server
}

type debugService struct {
	Name   string
	Method map[string]*methodType
}

func (server debugHTTP) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 建立数据的排序版本
	var services []debugService
	server.serviceMap.Range(func(key, value any) bool {
		svc := value.(*service)
		services = append(services, debugService{
			Name:   key.(string),
			Method: svc.method,
		})
		return true
	})
	// 返回一个 HTML 报文，这个报文将展示注册所有的 service 的每一个方法的调用情况
	err := debug.Execute(w, services)
	if err != nil {
		_, _ = fmt.Fprintln(w, "rpc: error executing template:", err.Error())
	}
}
