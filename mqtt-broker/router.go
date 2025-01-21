package mqttbroker

import (
	"log"
	"strings"

	"github.com/golangid/candi/codebase/factory/types"
)

type router struct {
	root *node
}

func newRouteNode() *node {
	return &node{
		children: make([]*node, 0),
	}
}

func (n *router) addRoute(path string, handler types.WorkerHandler) (normalizePath string) {
	path = strings.Trim(path, "/")
	parent := n.root
	var token string

	for path != "" {
		child := newRouteNode()
		token, path = nextPath(path)
		if strings.HasPrefix(token, ":") {
			child.paramName = strings.TrimSpace(token[1:])
			child.paramNode = true
			normalizePath += "+/"
		} else {
			child.path = token
			normalizePath += token + "/"
		}
		parent = parent.insertChild(child)
	}
	if len(parent.handler.HandlerFuncs) > 0 {
		log.Panicf("'%s' has already a handler", path)
	}
	parent.handler = handler
	return strings.Trim(normalizePath, "/")
}

func (n *router) match(path string) (h types.WorkerHandler, params map[string]string) {
	path = strings.Trim(path, "/")
	var matched bool
	var token string
	params = make(map[string]string)
	route := n.root

	for path != "" {
		token, path = nextPath(path)
		matched = false
		for _, c := range route.children {
			if c.path == token {
				route = c
				matched = true
				break
			}
		}

		if !matched {
			if route.paramChild != nil {
				route = route.paramChild
				matched = true
				params[route.paramName] = token
				continue
			}
			return
		}
	}
	return route.handler, params
}

type node struct {
	paramNode  bool
	paramName  string
	path       string
	children   []*node
	paramChild *node
	handler    types.WorkerHandler
}

func (n *node) insertChild(nn *node) *node {
	if child := n.findChild(nn); child != nil {
		return child
	}

	if n.paramChild != nil && nn.paramNode {
		if n.paramChild.paramName != nn.paramName {
			panic("Param name must be same for")
		}
		return n.paramChild
	}
	if nn.paramNode {
		n.paramChild = nn
	} else {
		n.children = append(n.children, nn)
	}
	return nn
}

func (n *node) findChild(nn *node) *node {
	for _, c := range n.children {
		if c.path == nn.path {
			return c
		}
	}
	return nil
}

func nextPath(path string) (string, string) {
	i := strings.Index(path, "/")
	if i == -1 {
		return path, ""
	}
	return path[:i], path[i+1:]
}
