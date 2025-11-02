package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	adminCookieValue = "1"
)

// RequireAdmin ensures the incoming request has a valid admin cookie. When the cookie
// is missing or invalid the client is redirected to the login page, preserving the
// originally requested path so the user can be sent back after authenticating.
func RequireAdmin(cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if v, err := c.Cookie(cookieName); err == nil && v == adminCookieValue {
			c.Next()
			return
		}

		target := c.Request.URL.RequestURI()
		redirectURL := "/login"
		if target != "" && target != "/" {
			redirectURL = redirectURL + "?next=" + url.QueryEscape(target)
		}

		c.Redirect(http.StatusFound, redirectURL)
		c.Abort()
	}
}
