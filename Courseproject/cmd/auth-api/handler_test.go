package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"../../internal/database"
	"../../internal/session"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testCase struct {
	Accept  string
	Request string
	Code    int
}

func NewTestServer() (*httptest.Server, error) {
	r := chi.NewRouter()

	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	userStorage := database.NewUserStorage()
	sessionStorage := database.NewSessionStorage()
	robotStorage := database.NewRobotStorage()

	h, err := NewHandler(logger, userStorage, sessionStorage, robotStorage)
	if err != nil {
		logger.Sugar().Fatalf("Can't create server: %s", err)
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/signup", h.PostSignup)
		r.Post("/signin", h.PostSignin)
		r.Route("/users/{id}", func(r chi.Router) {
			r.Put("/", h.PutUser)
			r.Get("/", h.GetUser)
			// r.Get("/robots", h.GetUserRobots)
		})
		r.Route("/robot", func(r chi.Router) {
			r.Post("/", h.CreateRobot)
			// r.Route("/{id}", func(r chi.Router) {
			// 		r.Delete("/", h.DeleteRobotByID)
			// 		// r.Put("/", h.UpdateRobotByID)
			// r.Get("/", h.GetRobotDetails)
			// 		r.Put("/favorite", h.AddRobotToFavorite)
			// 		r.Put("/activate", h.ActivateRobot)
			// 		r.Put("/deactivate", h.DeactivateRobot)
			// })
			// 	r.HandleFunc("/robots_ws", h.WSRobotUpdate)
		})
		// r.Route("/robots", func(r chi.Router) {
		// 	r.Get("/", h.GetRobots)
		// })
	})

	return httptest.NewServer(r), nil
}

func TestHandler_PostSignUp(t *testing.T) {
	tc := testCase{
		Request: `{"first_name": "Golang","last_name": "Developer", "email": "go_dev@tinkoff.ru","password": "password"}`,
		Accept:  "application/json",
		Code:    http.StatusCreated,
	}

	r := require.New(t)

	ts, err := NewTestServer()
	r.NoError(err)

	client := http.Client{Timeout: time.Second}
	resp, err := client.Post(fmt.Sprintf("%s/api/v1/signup", ts.URL), tc.Accept, bytes.NewBuffer([]byte(tc.Request)))

	r.NoError(err)
	r.Equal(tc.Code, resp.StatusCode)
	resp.Body.Close()
}

func TestHandler_PostSignin(t *testing.T) {
	tc := testCase{
		Request: `{"first_name": "Golang","last_name": "Developer", "email": "go_dev@tinkoff.ru","password": "password"}`,
		Accept:  "application/json",
		Code:    http.StatusOK,
	}

	r := require.New(t)

	ts, err := NewTestServer()
	r.NoError(err)

	client := http.Client{Timeout: time.Second}
	resp, _ := client.Post(fmt.Sprintf("%s/api/v1/signup", ts.URL), tc.Accept, bytes.NewBuffer([]byte(tc.Request)))
	resp.Body.Close()

	resp, err = client.Post(fmt.Sprintf("%s/api/v1/signin", ts.URL), tc.Accept, bytes.NewBuffer([]byte(tc.Request)))

	r.NoError(err)
	r.Equal(tc.Code, resp.StatusCode)
	resp.Body.Close()
}

func TestHandler_PutUser(t *testing.T) {
	tc := testCase{
		Request: `{"first_name": "Golang","last_name": "Developer", "email": "go_dev@tinkoff.ru","password": "password"}`,
		Accept:  "application/json",
		Code:    http.StatusOK,
	}

	r := require.New(t)

	ts, err := NewTestServer()
	r.NoError(err)

	client := http.Client{Timeout: time.Second}
	resp, _ := client.Post(fmt.Sprintf("%s/api/v1/signup", ts.URL), tc.Accept, bytes.NewBuffer([]byte(tc.Request)))
	resp.Body.Close()

	resp, err = client.Post(fmt.Sprintf("%s/api/v1/signin", ts.URL), tc.Accept, bytes.NewBuffer([]byte(tc.Request)))

	r.NoError(err)

	var ans session.Session

	err = json.NewDecoder(resp.Body).Decode(&ans)

	r.NoError(err)

	token := ans.SessionID

	resp.Body.Close()

	tc.Request = `{"email": "go_dev@tinkoff.ru","password": "new_password"}`
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/users/%d", ts.URL, ans.UserID), bytes.NewBuffer([]byte(tc.Request)))

	r.NoError(err)
	req.Header.Add("Authorization", token)

	resp, err = client.Do(req)

	r.NoError(err)

	defer resp.Body.Close()
}

func TestHandler_GetUser(t *testing.T) {
	tc := testCase{
		Request: `{"first_name": "Golang","last_name": "Developer", "email": "go_dev@tinkoff.ru","password": "password"}`,
		Accept:  "application/json",
		Code:    http.StatusOK,
	}

	r := require.New(t)

	ts, err := NewTestServer()
	r.NoError(err)

	client := http.Client{Timeout: time.Second}
	resp, _ := client.Post(fmt.Sprintf("%s/api/v1/signup", ts.URL), tc.Accept, bytes.NewBuffer([]byte(tc.Request)))
	resp.Body.Close()

	resp, err = client.Post(fmt.Sprintf("%s/api/v1/signin", ts.URL), tc.Accept, bytes.NewBuffer([]byte(tc.Request)))

	r.NoError(err)

	var ans session.Session

	err = json.NewDecoder(resp.Body).Decode(&ans)

	r.NoError(err)

	token := ans.SessionID

	resp.Body.Close()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/users/%d", ts.URL, ans.UserID), bytes.NewBuffer([]byte(tc.Request)))

	r.NoError(err)
	req.Header.Add("Authorization", token)

	resp, err = client.Do(req)

	r.NoError(err)

	defer resp.Body.Close()
}

func Test_CreateRobot(t *testing.T) {
	u := `{"first_name": "Golang","last_name": "Developer", "email": "go_dev@tinkoff.ru","password": "password"}`
	tc := testCase{
		Request: `{}`,
		Accept:  "application/json",
		Code:    http.StatusOK,
	}

	r := require.New(t)

	ts, err := NewTestServer()
	r.NoError(err)

	client := http.Client{Timeout: time.Second}
	resp, _ := client.Post(fmt.Sprintf("%s/api/v1/signup", ts.URL), tc.Accept, bytes.NewBuffer([]byte(u)))
	resp.Body.Close()

	resp, err = client.Post(fmt.Sprintf("%s/api/v1/signin", ts.URL), tc.Accept, bytes.NewBuffer([]byte(u)))

	r.NoError(err)

	var ans session.Session

	err = json.NewDecoder(resp.Body).Decode(&ans)

	r.NoError(err)

	token := ans.SessionID

	resp.Body.Close()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/robot/", ts.URL), bytes.NewBuffer([]byte(tc.Request)))

	r.NoError(err)
	req.Header.Add("Authorization", token)

	resp, err = client.Do(req)

	r.NoError(err)

	defer resp.Body.Close()
}
