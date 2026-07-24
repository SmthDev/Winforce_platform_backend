package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecodearcher/limen"
)


const ContextUserKey = "user"



func RequireAuth(instance *limen.Limen) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := instance.GetSession(c.Request)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		c.Set(ContextUserKey, session.User)
		c.Next()
	}
}


func CurrentUser(c *gin.Context) (*limen.User, bool) {
	value, exists := c.Get(ContextUserKey)
	if !exists {
		return nil, false
	}
	user, ok := value.(*limen.User)
	return user, ok
}
