/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"seata.apache.org/seata-go-samples/quick_start/order/model"
	"seata.apache.org/seata-go-samples/quick_start/order/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{
		svc: svc,
	}
}

func (o *OrderHandler) Route(engine *gin.Engine) {
	group := engine.Group("/order")
	group.POST("/create", o.Create)
}

func (o *OrderHandler) Create(ctx *gin.Context) {
	var order Order
	if err := ctx.Bind(&order); err != nil {
		ctx.String(http.StatusInternalServerError, "internal error")
		return
	}
	id, err := o.svc.Create(ctx, o.toModel(order))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "internal error")
		return
	}
	ctx.JSON(http.StatusOK, Message{
		Code: http.StatusOK,
		Data: id,
	})
}

func (o *OrderHandler) toModel(order Order) model.Order {
	return model.Order{
		UserID: order.UserID,
		Money:  order.Money,
	}
}
