package comments

// type Handler struct {
// 	services services.Comments
// }

// func NewHandler(services *services.Comments) *Handler {
// 	return &Handler{
// 		services: services,
// 	}
// }

// func Register(api *gin.RouterGroup, services services.Comments, middleware *middleware.Middleware) {
// 	h := NewHandler(services)
// 	//TODO реализовать

// 	comments := api.Group("/tickets/:id/comments", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
// 	{
// 		comments.GET("", h.getByTicket)

// 		comments.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Write()))
// 		comments.POST("", h.create)

// 		comments.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Delete()))
// 		comments.DELETE("/:id", h.delete)
// 	}
// }

// func (h *Handler) getByTicket(c *gin.Context) {
// 	response.SendError(c, models.ErrNotImplemented)
// }

// func (h *Handler) create(c *gin.Context) {
// 	response.SendError(c, models.ErrNotImplemented)
// }

// func (h *Handler) delete(c *gin.Context) {
// 	response.SendError(c, models.ErrNotImplemented)
// }
