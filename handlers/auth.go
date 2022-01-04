package handlers

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/shareed2k/goth_fiber"
	"github.com/thetkpark/cscms-temp-storage/data"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"github.com/thetkpark/cscms-temp-storage/service/jwt"
	"github.com/thetkpark/cscms-temp-storage/service/token"
	"go.uber.org/zap"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AuthRouteHandler struct {
	log           *zap.SugaredLogger
	userDataStore data.UserDataStore
	jwtManager    jwt.Manager
	tokenManager  token.Manager
	entrypoint    string
}

func NewAuthRouteHandler(l *zap.SugaredLogger, userDataStore data.UserDataStore, jwtManager jwt.Manager, tokenManager token.Manager, entry string) *AuthRouteHandler {
	return &AuthRouteHandler{
		log:           l,
		userDataStore: userDataStore,
		jwtManager:    jwtManager,
		tokenManager:  tokenManager,
		entrypoint:    entry,
	}
}

func (a AuthRouteHandler) OauthProviderCallback(c *fiber.Ctx) error {
	gothUser, err := goth_fiber.CompleteUserAuth(c)
	if err != nil {
		a.log.Error("unable to complete auth\n" + err.Error())
		return c.Redirect(a.entrypoint)
	}

	// Check existing user
	user, err := a.userDataStore.FindByProviderAndEmail(gothUser.Provider, gothUser.Email)
	if err != nil {
		a.log.Error("unable to find existing user\n" + err.Error())
		return c.Redirect(a.entrypoint)
	}

	if user == nil {
		// Create new user
		username := a.getUserName(gothUser.NickName, gothUser.FirstName, gothUser.Name, gothUser.Email)
		user, err = a.userDataStore.Create(gothUser.Email, username, gothUser.Provider, gothUser.AvatarURL)
		if err != nil {
			a.log.Error("unable to create user\n" + err.Error())
			return c.Redirect(a.entrypoint)
		}
	}

	// Create JWT
	jwtToken, err := a.jwtManager.Generate(strconv.Itoa(int(user.ID)))
	if err != nil {
		a.log.Error("unable to generate JWT\n" + err.Error())
		return c.Redirect(a.entrypoint)
	}

	// Create cookie and attach
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    jwtToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Secure:   true,
		HTTPOnly: true,
		SameSite: "lax",
	})

	// Goth Session
	//if err := goth_fiber.StoreInSession("userId", strconv.Itoa(int(user.ID)), c); err != nil {
	//	a.log.Error("unable to store userId in session\n" + err.Error())
	//	return c.Redirect("http://localhost:5050")
	//}

	return c.Redirect(a.entrypoint)
}

// GetUserInfo handlers
// @Summary Get user info
// @Description Get the user information
// @Tags Auth
// @Success  200 {object} model.User
// @Failure  401 {object}  handlers.ErrorResponse
// @Router /auth/user [get]
func (a *AuthRouteHandler) GetUserInfo(c *fiber.Ctx) error {
	user := c.UserContext().Value("user")
	if user == nil {
		return NewHTTPError(a.log, fiber.StatusUnauthorized, "Unauthorized", nil)
	}
	userModel, ok := user.(*model.User)
	if !ok {
		return NewHTTPError(a.log, fiber.StatusUnauthorized, "Unauthorized", nil)
	}
	return c.JSON(userModel)
}

// Logout handlers
// @Summary Logout
// @Description Clear the cookie
// @Tags Auth
// @Success  200
// @Router /auth/logout [get]
func (a *AuthRouteHandler) Logout(c *fiber.Ctx) error {
	a.clearCookie(c)
	return c.SendStatus(fiber.StatusOK)
}

// GenerateAPIToken handlers
// @Summary Generate new api token
// @Description Generate new api token for the user
// @Tags Auth
// @Success  201 {object} model.User
// @Failure  401 {object}  handlers.ErrorResponse
// @Failure  500 {object}  handlers.ErrorResponse
// @Router /api/auth/token [post]
func (a *AuthRouteHandler) GenerateAPIToken(c *fiber.Ctx) error {
	user := c.UserContext().Value("user")
	if user == nil {
		return NewHTTPError(a.log, fiber.StatusUnauthorized, "Unauthorized", nil)
	}
	userModel, ok := user.(*model.User)
	if !ok {
		return NewHTTPError(a.log, fiber.StatusInternalServerError, "unable to parse file model", fmt.Errorf("unable to parse file model"))
	}

	apiKey, err := a.tokenManager.GenerateAPIToken()
	if err != nil {
		return NewHTTPError(a.log, fiber.StatusInternalServerError, "unable to generate api token", err)
	}

	userModel.APIKey = apiKey
	err = a.userDataStore.UpdateAPIKey(userModel.ID, apiKey)
	if err != nil {
		return NewHTTPError(a.log, fiber.StatusInternalServerError, "unable to save new api token", err)
	}

	return c.Status(fiber.StatusCreated).JSON(userModel)
}

func (a *AuthRouteHandler) ParseUser(c *fiber.Ctx) error {
	var user *model.User = nil
	jwtToken := c.Cookies("token", "")
	// Check if JWT token in cookie is found
	if len(jwtToken) == 0 {
		// Check api-token in request header
		apiKey := c.Get("x-api-key", "")
		if len(apiKey) == 0 {
			return c.Next()
		}

		// Get user from api-token
		userModel, err := a.userDataStore.FindByAPIKey(apiKey)
		if err != nil {
			return NewHTTPError(a.log, fiber.StatusInternalServerError, "unable to get user by api token", err)
		}
		if userModel == nil {
			return c.Next()
		}
		user = userModel
	} else {
		// Validate JWT token to get user ID in aud field
		userIdString, err := a.jwtManager.Validate(jwtToken)
		if err != nil {
			a.clearCookie(c)
			return c.Next()
		}

		// Get user from userID
		userIdInt, err := strconv.Atoi(userIdString)
		if err != nil {
			a.clearCookie(c)
			return c.Next()
		}
		user, err = a.userDataStore.FindById(uint(userIdInt))
		if err != nil {
			return NewHTTPError(a.log, fiber.StatusInternalServerError, "unable to get user by id", err)
		} else if user == nil {
			a.clearCookie(c)
			return c.Next()
		}
	}

	c.SetUserContext(context.WithValue(c.UserContext(), "user", user))
	return c.Next()
}

func (a *AuthRouteHandler) AuthenticatedOnly(c *fiber.Ctx) error {
	user := c.UserContext().Value("user")
	if user == nil {
		return NewHTTPError(a.log, fiber.StatusUnauthorized, "Unauthenticated", nil)
	}
	return c.Next()
}

func (a *AuthRouteHandler) getUserName(nickname, firstname, name, email string) string {
	if len(nickname) != 0 {
		return nickname
	}
	if len(firstname) != 0 {
		return firstname
	}
	if len(name) != 0 {
		return strings.Split(name, " ")[0]
	}

	emailRegex := regexp.MustCompile("(.+)@.+")
	subMatch := emailRegex.FindStringSubmatch(email)
	return subMatch[1]
}

func (a *AuthRouteHandler) clearCookie(c *fiber.Ctx) {
	_ = goth_fiber.Logout(c)
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Expires:  time.Now().Add(-(time.Hour * 24)),
		Secure:   false,
		HTTPOnly: true,
		SameSite: "lax",
	})
}
