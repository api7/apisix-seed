package entity

import (
	"net"
	"strconv"
)

func mapKV2Node(key string, val float64) (*Node, error) {
	host, port, err := net.SplitHostPort(key)
	if err != nil {
		return nil, err
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}

	node := &Node{
		Host:   host,
		Port:   portInt,
		Weight: int(val),
	}
	return node, nil
}

func NodesFormat(obj interface{}) interface{} {
	nodes := make([]*Node, 0)
	switch value := obj.(type) {
	case map[string]float64:
		for key, val := range value {
			node, err := mapKV2Node(key, val)
			if err != nil {
				return obj
			}
			nodes = append(nodes, node)
		}
		return nodes
	case map[string]interface{}:
		for key, val := range value {
			node, err := mapKV2Node(key, val.(float64))
			if err != nil {
				return obj
			}
			nodes = append(nodes, node)
		}
		return nodes
	case []*Node:
		return obj
	case []interface{}:
		for _, v := range value {
			val := v.(map[string]interface{})
			node := &Node{
				Host:   val["host"].(string),
				Port:   int(val["port"].(float64)),
				Weight: int(val["weight"].(float64)),
			}
			nodes = append(nodes, node)
		}
		return nodes
	}

	return obj
}
