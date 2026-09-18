package handler

import (
	"context"
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

type AdminUsersHandler struct {
	users *repository.UserRepository
	audit *service.AuditService
}

func NewAdminUsersHandler(users *repository.UserRepository, audit *service.AuditService) *AdminUsersHandler {
	return &AdminUsersHandler{users: users, audit: audit}
}

func (h *AdminUsersHandler) Register(admin fiber.Router) {
	admin.Get("/users", h.list)
	admin.Post("/users", h.create)
	admin.Put("/users/:id", h.update)
	admin.Post("/users/:id/password", h.resetPassword)
	admin.Delete("/users/:id", h.delete)
}

func userJSON(u model.User) fiber.Map {
	return fiber.Map{"id": u.ID, "email": u.Email, "role": u.Role, "createdAt": u.CreatedAt}
}

func (h *AdminUsersHandler) list(c *fiber.Ctx) error {
	offset, limit := paginationParams(c)
	users, total, err := h.users.List(c.Context(), c.Query("search"), offset, limit)
	if err != nil {
		log.Printf("list users: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not list users")
	}
	rows := make([]fiber.Map, len(users))
	for i, u := range users {
		rows[i] = userJSON(u)
	}
	return c.JSON(fiber.Map{"users": rows, "total": total})
}

type createUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *AdminUsersHandler) create(c *fiber.Ctx) error {
	var req createUserRequest
	if err := c.BodyParser(&req); err != nil || req.Email == "" || len(req.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "email and a password of at least 8 characters are required")
	}
	role := req.Role
	if role != model.RoleAdmin && role != model.RoleUser {
		role = model.RoleUser
	}

	hash, err := service.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not create user")
	}

	user, err := h.users.Create(c.Context(), req.Email, hash, role, "")
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "a user with this email may already exist")
	}
	go h.audit.Log(context.Background(), actorEmail(c), "user.created", "email="+user.Email+" role="+user.Role, c.IP())
	return c.Status(fiber.StatusCreated).JSON(userJSON(*user))
}

type updateUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *AdminUsersHandler) update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var req updateUserRequest
	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}
	role := req.Role
	if role != model.RoleAdmin && role != model.RoleUser {
		role = model.RoleUser
	}

	user, err := h.users.UpdateEmailAndRole(c.Context(), id, req.Email, role)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "could not update user")
	}
	go h.audit.Log(context.Background(), actorEmail(c), "user.updated", "email="+user.Email+" role="+user.Role, c.IP())
	return c.JSON(userJSON(*user))
}

type resetUserPasswordRequest struct {
	Password string `json:"password"`
}

func (h *AdminUsersHandler) resetPassword(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var req resetUserPasswordRequest
	if err := c.BodyParser(&req); err != nil || len(req.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}
	hash, err := service.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not update password")
	}
	if err := h.users.UpdatePassword(c.Context(), id, hash); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not update password")
	}
	go h.audit.Log(context.Background(), actorEmail(c), "user.password_reset", "userId="+id.String(), c.IP())
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AdminUsersHandler) delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.users.Delete(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not delete user")
	}
	go h.audit.Log(context.Background(), actorEmail(c), "user.deleted", "userId="+id.String(), c.IP())
	return c.JSON(fiber.Map{"ok": true})
}
