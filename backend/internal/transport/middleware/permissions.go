package middleware

// func (m *Middleware) CheckPermissions(required ...access.Permission) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		realm := c.GetHeader("realm")

// 		u, exists := c.Get(constants.CtxUser)
// 		if !exists {
// 			response.NewErrorResponse(c, http.StatusUnauthorized, "empty user", "сессия не найдена")
// 			c.Abort()
// 			return
// 		}

// 		user := u.(models.User)

// 		for _, r := range required {
// 	if _, ok := userPerms["*"]; ok {
// 	c.Next()
// 	return
// }

// 			ok, err := m.services.Permission.Enforce(
// 				user.ID,
// 				realm,
// 				string(r.Resource),
// 				string(r.Action),
// 			)

// 			if err != nil || !ok {
// 				c.AbortWithStatus(http.StatusForbidden)
// 				return
// 			}
// 		}

// 		// 🔥 ВАЖНО: теперь используем Key()
// 		accessAllowed, err := m.services.Permission.Enforce(
// 			user.ID,
// 			realm,
// 			string(required.Resource), // было menuItem
// 			string(required.Action),   // было method
// 		)

// 		if err != nil {
// 			response.NewErrorResponse(
// 				c,
// 				http.StatusInternalServerError,
// 				err.Error(),
// 				"Произошла ошибка: "+err.Error(),
// 			)
// 			c.Abort()
// 			return
// 		}

// 		if !accessAllowed {
// 			response.NewErrorResponse(
// 				c,
// 				http.StatusForbidden,
// 				"forbidden",
// 				"недостаточно прав",
// 			)
// 			c.Abort()
// 			return
// 		}

// 		c.Next()
// 	}
// }
