/**
  @author: decision
  @date: 2023/9/18
  @note:
**/

package main

import "github.com/gin-gonic/gin"

func GraphFieldsService(c *gin.Context) {
	c.JSON(200, gin.H{
		"edges_fields": []gin.H{
			{"field_name": "id", "type": "number"},
			{"field_name": "source", "type": "number"},
			{"field_name": "target", "type": "number"},
		},
		"nodes_fields": []gin.H{
			{"field_name": "id", "type": "number"},
			{"field_name": "title", "type": "string"},
			{"field_name": "identity", "type": "string"},
		},
	})
}

func GraphDataService(c *gin.Context) {
	graphLock.RLock()
	defer graphLock.RUnlock()

	c.JSON(200, gin.H{
		"edges": edges,
		"nodes": nodes,
	})
}

func GraphHealthService(c *gin.Context) {
	c.JSON(200, "")
}
