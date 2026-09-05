package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// ProUserNode 用户可用的后台任务节点
type ProUserNode struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ProUserNodesResponse 返回当前用户可用节点列表
type ProUserNodesResponse struct {
	Nodes []*ProUserNode `json:"nodes"`
}

// ProUserListNodes 返回当前用户组可用节点列表，受组配置 AllowedNodes 过滤。
func ProUserListNodes(c *gin.Context) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	var allowedNodes []int
	if user.Edges.Group != nil && user.Edges.Group.Settings != nil {
		allowedNodes = user.Edges.Group.Settings.AllowedNodes
	}

	nodes, err := dep.NodeClient().ListActiveNodes(c, allowedNodes)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	res := &ProUserNodesResponse{Nodes: make([]*ProUserNode, 0, len(nodes))}
	for _, n := range nodes {
		res.Nodes = append(res.Nodes, &ProUserNode{ID: n.ID, Name: n.Name})
	}

	c.JSON(200, serializer.Response{Data: res})
}
