package handler

import (
	"context"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

type AdminBlockedIPsHandler struct {
	blocked *repository.BlockedIPRepository
	audit   *service.AuditService
}

func NewAdminBlockedIPsHandler(blocked *repository.BlockedIPRepository, audit *service.AuditService) *AdminBlockedIPsHandler {
	return &AdminBlockedIPsHandler{blocked: blocked, audit: audit}
}

func (h *AdminBlockedIPsHandler) Register(admin fiber.Router) {
	admin.Get("/blocked-ips", h.list)
	admin.Post("/blocked-ips", h.add)
	admin.Delete("/blocked-ips/:id", h.remove)
}

func (h *AdminBlockedIPsHandler) list(c *fiber.Ctx) error {
	rows, err := h.blocked.List(c.Context())
	if err != nil {
		log.Printf("list blocked ips: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not list blocked IPs")
	}
	return c.JSON(fiber.Map{"blockedIps": rows})
}

type addBlockedIPRequest struct {
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}

func (h *AdminBlockedIPsHandler) add(c *fiber.Ctx) error {
	var req addBlockedIPRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	req.IP = strings.TrimSpace(req.IP)
	if req.IP == "" {
		return fiber.NewError(fiber.StatusBadRequest, "an IP address is required")
	}

	row, err := h.blocked.Add(c.Context(), req.IP, strings.TrimSpace(req.Reason))
	if err != nil {
		log.Printf("add blocked ip: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not block IP")
	}
	go h.audit.Log(context.Background(), actorEmail(c), "ip.blocked", "ip="+row.IP, c.IP())
	return c.Status(fiber.StatusCreated).JSON(row)
}

func (h *AdminBlockedIPsHandler) remove(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.blocked.Remove(c.Context(), id); err != nil {
		log.Printf("remove blocked ip: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "could not unblock IP")
	}
	go h.audit.Log(context.Background(), actorEmail(c), "ip.unblocked", "id="+id.String(), c.IP())
	return c.JSON(fiber.Map{"ok": true})
}
