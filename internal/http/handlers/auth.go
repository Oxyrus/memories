package handlers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Oxyrus/memories/internal/http/render"
	"github.com/Oxyrus/memories/web/pages"
)

type AuthHandler struct {
	logger     *slog.Logger
	passcode   string
	cookieName string
}

func NewAuthHandler(logger *slog.Logger, passcode, cookieName string) *AuthHandler {
	return &AuthHandler{
		logger:     logger,
		passcode:   passcode,
		cookieName: cookieName,
	}
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	next := strings.TrimSpace(c.Query("next"))
	if next != "" {
		if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
			next = ""
		}
	}

	render.HTML(c, http.StatusOK, pages.Login(next))
}

func (h *AuthHandler) SubmitLogin(c *gin.Context) {
	passcode := strings.TrimSpace(c.PostForm("passcode"))
	if passcode == "" {
		h.logger.Warn("login attempt missing passcode", "ip", c.ClientIP())
		c.String(http.StatusBadRequest, "passcode is required")
		return
	}

	if passcode != h.passcode {
		h.logger.Warn("invalid login attempt", "ip", c.ClientIP())
		c.String(http.StatusUnauthorized, "invalid passcode")
		return
	}

	redirectTo := c.PostForm("next")
	if redirectTo == "" {
		redirectTo = "/albums"
	}

	maxAge := int((14 * 24 * time.Hour).Seconds())
	secure := c.Request.TLS != nil
	c.SetCookie(h.cookieName, "1", maxAge, "/", "", secure, true)

	h.logger.Info("admin login successful", "ip", c.ClientIP())
	c.Redirect(http.StatusFound, redirectTo)
}
