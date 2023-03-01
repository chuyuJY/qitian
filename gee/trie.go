package gee

import "strings"

// 请求类型为根, 各建一颗trie树, 例如: GET
type node struct {
	pattern  string  // 待匹配路由, 例如: /p/:lang
	part     string  //路由的一部分, 例如: :lang
	children []*node // 子节点, 例如: [doc, tutorial, intro]
	isWild   bool    // 是否模糊匹配, part 含有 : 或 * 时为true
}

// 寻找第一个匹配成功的节点
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// 寻找所有匹配成功的节点
func (n *node) matchChildren(part string) []*node {
	children := []*node{}
	for _, child := range n.children {
		if child.part == part || child.isWild {
			children = append(children, child)
		}
	}
	return children
}

// 插入节点: 路由注册
func (n *node) insert(pattern string, parts []string, height int) {
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	part := parts[height]
	child := n.matchChild(part)
	if child == nil {
		child = &node{
			part:   part,
			isWild: part[0] == ':' || part[0] == '*',
		}
		n.children = append(n.children, child)
	}
	// 递归插入
	child.insert(pattern, parts, height+1)
}

// 查询节点: 路由发现
func (n *node) search(parts []string, height int) *node {
	// 若 遍历完毕parts 或 发现通配符 *
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}
	part := parts[height]
	children := n.matchChildren(part)
	for _, child := range children {
		target := child.search(parts, height+1)
		if target != nil {
			return target
		}
	}
	return nil
}
