package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bulgadev/pushUpCounter/Front"
	"github.com/bulgadev/pushUpCounter/utils"
	"github.com/labstack/echo/v4"
)

func loggedInUsername(c echo.Context, db *sql.DB) (string, error) {
	cookie, err := c.Cookie("session_token")
	if errors.Is(err, http.ErrNoCookie) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(cookie.Value)
	if token == "" {
		return "", nil
	}

	return utils.FindUsernameByToken(db, token)
}

func setSessionCookie(c echo.Context, token string) {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Scheme() == "https",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		MaxAge:   7 * 24 * 60 * 60,
	}

	c.SetCookie(cookie)
}

func main() {
	e := echo.New()

	var totalGoal int = 1500

	// 1. Initialize DuckDB
	db, err := utils.OpenDuckDB("data.db")
	if err != nil {
		e.Logger.Fatal(err)
	}
	defer db.Close()

	if err := utils.EnsureUsersTableSchema(db); err != nil {
		e.Logger.Fatal(err)
	}

	// 2. Routes
	e.GET("/", func(c echo.Context) error {
		counts, toGo, totalCount, err := utils.GetCountsAndTotals(db, totalGoal)
		if err != nil {
			e.Logger.Error(err)
			return c.String(500, "internal server error")
		}

		username, err := loggedInUsername(c, db)
		if err != nil {
			e.Logger.Error(err)
			return c.String(500, "internal server error")
		}

		component := Front.Page(counts, toGo, totalCount, username != "", username)
		return component.Render(c.Request().Context(), c.Response().Writer)
	})

	e.GET("/login", func(c echo.Context) error {
		component := Front.LoginPage("Developer", nil)
		return component.Render(c.Request().Context(), c.Response().Writer)
	})

	e.POST("/LoginRoute", func(c echo.Context) error {
		username := strings.TrimSpace(c.FormValue("username"))
		password := c.FormValue("password")

		if username == "" || password == "" {
			return c.String(http.StatusBadRequest, "username and password are required")
		}

		token, _, valid, err := utils.AuthenticateUserAndGetToken(db, username, password)
		if err != nil {
			e.Logger.Error(err)
			return c.String(http.StatusInternalServerError, "internal server error")
		}
		if !valid {
			return c.String(http.StatusUnauthorized, "invalid username or password")
		}

		setSessionCookie(c, token)

		if c.Request().Header.Get("HX-Request") == "true" {
			c.Response().Header().Set("HX-Redirect", "/")
			return c.String(http.StatusOK, "login successful")
		}

		return c.Redirect(http.StatusSeeOther, "/")
	})

	e.POST("/registerPushs", func(c echo.Context) error {
		sessionUser, err := loggedInUsername(c, db)
		if err != nil {
			e.Logger.Error(err)
			return c.String(http.StatusInternalServerError, "internal server error")
		}
		if sessionUser == "" {
			return c.String(http.StatusUnauthorized, "you must be logged in")
		}

		username := strings.TrimSpace(c.FormValue("username"))
		if username == "" {
			username = sessionUser
		}
		if username != sessionUser {
			return c.String(http.StatusForbidden, "username does not match active session")
		}

		pushUps, err := strconv.Atoi(strings.TrimSpace(c.FormValue("pushUps")))
		if err != nil || pushUps <= 0 {
			return c.String(http.StatusBadRequest, "pushUps must be a positive number")
		}

		if err := utils.RegisterPushUps(db, username, pushUps); err != nil {
			if errors.Is(err, utils.ErrUserNotFound) {
				return c.String(http.StatusNotFound, "user not found")
			}

			e.Logger.Error(err)
			return c.String(http.StatusInternalServerError, "internal server error")
		}

		counts, toGo, totalCount, err := utils.GetCountsAndTotals(db, totalGoal)
		if err != nil {
			e.Logger.Error(err)
			return c.String(http.StatusInternalServerError, "internal server error")
		}

		if c.Request().Header.Get("HX-Request") == "true" {
			message := "Registered " + strconv.Itoa(pushUps) + " push ups for " + username + "."
			return Front.PushRegisterResponse(message, counts, toGo, totalCount).Render(c.Request().Context(), c.Response().Writer)
		}

		return c.Redirect(http.StatusSeeOther, "/")
	})

	e.POST("/query", func(c echo.Context) error {
		counts, toGo, totalCount, err := utils.GetCountsAndTotals(db, totalGoal)
		if err != nil {
			e.Logger.Error(err)
			return c.String(500, "internal server error")
		}

		// Return only the fragment for HTMX, including totals
		return Front.RowList(counts, toGo, totalCount).Render(c.Request().Context(), c.Response().Writer)
	})

	e.Logger.Fatal(e.Start(":6432"))
}
