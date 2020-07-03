package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
	"time"

	"../../internal/robot"
	"../../internal/session"
	"../../internal/user"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Handler struct {
	logger         *zap.SugaredLogger
	userStorage    user.Storage
	sessionStorage session.Storage
	robotStorage   robot.Storage
	upgrader       websocket.Upgrader
	tmpl           map[string]*template.Template
	wsClients      WSClients
	robotsChan     chan robot.Robot
}

type WSClients struct {
	wsConn []*websocket.Conn
	mutex  sync.Mutex
}

func (clients *WSClients) AddClient(h *Handler, conn *websocket.Conn) {
	clients.mutex.Lock()
	clients.wsConn = append(clients.wsConn, conn)
	clients.mutex.Unlock()
	h.logger.Infof("added client, total clients: %d\n", len(h.wsClients.wsConn))
}

func (clients *WSClients) removeClientByID(h *Handler, id int) {
	clients.mutex.Lock()
	clients.wsConn = append(clients.wsConn[:id], clients.wsConn[id+1:]...)
	clients.mutex.Unlock()
	h.logger.Infof("removed client #%d, total clients %d\n", id, len(clients.wsConn))
}

func (clients *WSClients) BroadcastMessage(h *Handler, message []byte) {
	for i, c := range clients.wsConn {
		err := c.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			h.logger.Infof("can't broadcast message: %+v\n", err)
			clients.removeClientByID(h, i)
		}
	}
}

// nolint: gomnd
func NewHandler(logger *zap.Logger, userStorage user.Storage, sessionStorage session.Storage, robotStorage robot.Storage) (*Handler, error) {
	templates := make(map[string]*template.Template)
	templates["robots_list"] = template.Must(template.ParseFiles("html/robots.html", "html/base.html", "html/robot_table.html"))
	templates["user_robots"] = template.Must(template.ParseFiles("html/user_robots.html", "html/base.html", "html/robot_table.html"))
	templates["robot_info"] = template.Must(template.ParseFiles("html/robot_info.html", "html/base.html"))

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	h := Handler{
		logger:         logger.Sugar(),
		userStorage:    userStorage,
		sessionStorage: sessionStorage,
		robotStorage:   robotStorage,
		upgrader:       upgrader,
		tmpl:           templates,
	}
	h.robotsChan = make(chan robot.Robot)

	return &h, nil
}

func (h *Handler) NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/signup", h.PostSignup)
		r.Post("/signin", h.PostSignin)
		r.Route("/users/{id}", func(r chi.Router) {
			r.Put("/", h.PutUser)
			r.Get("/", h.GetUser)
			r.Get("/robots", h.GetUserRobots)
		})
		r.Route("/robot", func(r chi.Router) {
			r.Post("/", h.CreateRobot)
			r.Route("/{id}", func(r chi.Router) {
				r.Delete("/", h.DeleteRobotByID)
				// r.Put("/", h.UpdateRobotByID)
				r.Get("/", h.GetRobotDetails)
				r.Put("/favorite", h.AddRobotToFavorite)
				r.Put("/activate", h.ActivateRobot)
				r.Put("/deactivate", h.DeactivateRobot)
			})
			r.HandleFunc("/robots_ws", h.WSRobotUpdate)
		})
		r.Route("/robots", func(r chi.Router) {
			r.Get("/", h.GetRobots)
		})
	})

	return r
}

func (h *Handler) renderTemplate(w http.ResponseWriter, name string, template string, viewModel interface{}) {
	tmpl, ok := h.tmpl[name]
	if !ok {
		http.Error(w, "can't find template", http.StatusInternalServerError)
	}

	err := tmpl.ExecuteTemplate(w, template, viewModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// nolint: gomnd
func (h *Handler) PostSignup(w http.ResponseWriter, r *http.Request) {
	var userData user.User

	err := json.NewDecoder(r.Body).Decode(&userData)
	if err != nil || !userData.CheckCorrectData() {
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusInternalServerError)
		return
	}

	passwordHash, err := user.HashPassword(userData.Password)
	if err != nil {
		http.Error(w, "{\"error\": \"can't hash password\"}", http.StatusInternalServerError)
		return
	}

	userData.Password = passwordHash

	if err := h.userStorage.Create(&userData); err != nil {
		h.logger.Errorf("Can't add user: %s", err)
		http.Error(w, fmt.Sprintf("{\"error\": \"user %s is already registered\"}", userData.Email), http.StatusConflict)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

// nolint: gomnd
func (h *Handler) PostSignin(w http.ResponseWriter, r *http.Request) {
	var userData user.User

	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		http.Error(w, "{\"error\": \"can't unwrap data\"}", http.StatusInternalServerError)
		return
	}

	u, err := h.userStorage.FindByEmail(userData.Email)
	if err != nil || !user.CheckPasswordHash(userData.Password, u.Password) {
		http.Error(w, "{\"error\": \" incorrect email or password\"}", http.StatusBadRequest)
		h.logger.Errorf("Can't find user: %s", err)

		return
	}

	sess, err := h.sessionStorage.FindByID(u.ID)
	if err == nil {
		if time.Now().Before(sess.ValidUntil) {
			err = json.NewEncoder(w).Encode(sess)
			if err != nil {
				http.Error(w, "{\"error\": \"can't return data\"}", http.StatusInternalServerError)
			}

			return
		}

		if err = h.sessionStorage.DeleteByID(u.ID); err != nil {
			h.logger.Errorf("%s", err)
		}
	}

	token, _ := session.GenerateToken()
	sessionData := session.Session{
		SessionID: token,
		UserID:    u.ID,
	}

	if err = h.sessionStorage.Create(&sessionData); err != nil {
		h.logger.Errorf("Can't add session: %s", err)
		http.Error(w, fmt.Sprintf("{\"error\": \"can't create session for user %d\"}", sessionData.UserID), http.StatusInternalServerError)

		return
	}

	err = json.NewEncoder(w).Encode(sessionData)
	if err != nil {
		http.Error(w, "{\"error\": \"can't return data\"}", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PutUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusBadRequest)
		return
	}

	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByID(id)
	if err != nil || token != sess.SessionID || !time.Now().Before(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusNotAcceptable)

		return
	}

	var userData user.User

	if err = json.NewDecoder(r.Body).Decode(&userData); err != nil {
		http.Error(w, "{\"error\": \"wrong input data\"}", http.StatusBadRequest)
		return
	}

	userData.ID = id

	if err = h.userStorage.UpdateByID(&userData); err != nil {
		http.Error(w, "{\"error\": \"could not update\"}", http.StatusInternalServerError)
		return
	}

	u, err := h.userStorage.FindByID(id)
	if err != nil {
		http.Error(w, "{\"error\": \"could not find\"}", http.StatusNotFound)
		return
	}

	userShort := user.ShortUser{FirstName: u.FirstName, LastName: u.LastName, Email: u.Email, Birthday: u.Birthday}
	err = json.NewEncoder(w).Encode(userShort)

	if err != nil {
		http.Error(w, "{\"error\": \"can't return data\"}", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || time.Now().After(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusBadRequest)

		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusInternalServerError)
		return
	}

	u, err := h.userStorage.FindByID(id)
	if err != nil {
		http.Error(w, "{\"error\": \"could not find\"}", http.StatusNotFound)
		return
	}

	userShort := user.ShortUser{FirstName: u.FirstName, LastName: u.LastName, Email: u.Email, Birthday: u.Birthday}
	err = json.NewEncoder(w).Encode(userShort)

	if err != nil {
		http.Error(w, "{\"error\": \"can't return data\"}", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) CreateRobot(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || time.Now().After(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusBadRequest)

		return
	}

	var robotData robot.Robot

	err = json.NewDecoder(r.Body).Decode(&robotData)
	if err != nil || robotData.BuyPrice >= robotData.SellPrice{
		http.Error(w, "{\"error\": \"wrong input data\"}", http.StatusBadRequest)
		return
	}

	if err := h.robotStorage.Create(&robotData); err != nil {
		h.logger.Errorf("Can't add robot: %s", err)
		http.Error(w, "{\"error\": \"can't create robot\"}", http.StatusBadRequest)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) GetUserRobots(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || time.Now().After(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusNotAcceptable)

		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusInternalServerError)
		return
	}

	_, err = h.userStorage.FindByID(id)
	if err != nil {
		http.Error(w, "{\"error\": \"user not found\"}", http.StatusNotFound)
		return
	}

	var robots []*robot.Robot

	robots, err = h.robotStorage.GetAllRobotsByOwnerID(id)
	if err != nil {
		h.logger.Errorf("Can't get list of robots: %s", err)
		http.Error(w, "{\"error\": \"can't get list of robots\"}", http.StatusInternalServerError)

		return
	}

	fmt.Println(robots)

	accept := r.Header.Get("Accept")

	switch accept {
	case "application/json":
		err = json.NewEncoder(w).Encode(robots)
		if err != nil {
			http.Error(w, "{\"error\": \"can't return data\"}", http.StatusBadRequest)
			return
		}
	default:
		h.renderTemplate(w, "user_robots", "base", struct {
			Robots []*robot.Robot
		}{robots})
	}
}

func (h *Handler) getListOfRobots(id int64, ticker string) ([]*robot.Robot, int, string) {
	var robots []*robot.Robot

	var err error

	haveID := id > 0
	haveTicker := ticker != ""
	haveTickerAndID := haveTicker && haveID

	switch {
	case haveTickerAndID:
		robots, err = h.robotStorage.GetAllRobotsByOwnerIDAndTicker(id, ticker)
	case haveID:
		robots, err = h.robotStorage.GetAllRobotsByOwnerID(id)
	case haveTicker:
		robots, err = h.robotStorage.GetAllRobotsByTicker(ticker)
	default:
		robots, err = h.robotStorage.GetAllRobots()
	}

	if err != nil {
		h.logger.Errorf("Can't get list of robots: %s", err)
		return nil, http.StatusInternalServerError, "can't get list of robots"
	}

	return robots, 0, ""
}

// nolint: gomnd
func (h *Handler) GetRobots(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || time.Now().After(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusBadRequest)

		return
	}

	var robots []*robot.Robot

	var id int64

	var ticker string

	query := r.URL.Query()
	ownerID, idOk := query["owner_user_id"]
	tickerList, tickerOk := query["ticker"]

	if idOk && len(ownerID) == 1 && ownerID[0] != "" {
		id, err = strconv.ParseInt(ownerID[0], 10, 64)
		if err != nil {
			http.Error(w, "{\"error\": \"can't parse id\"}", http.StatusInternalServerError)
			return
		}

		if _, err = h.userStorage.FindByID(id); err != nil {
			http.Error(w, "{\"error\": \"user not found\"}", http.StatusBadRequest)
			return
		}
	}

	if tickerOk && len(tickerList) == 1 {
		ticker = tickerList[0]
	}

	robots, code, comment := h.getListOfRobots(id, ticker)

	if code != 0 {
		http.Error(w, fmt.Sprintf("{\"error\": \"%s\"}", comment), code)
		return
	}

	accept := r.Header.Get("Accept")

	switch accept {
	case "application/json":
		err = json.NewEncoder(w).Encode(robots)
		if err != nil {
			http.Error(w, "{\"error\": \"can't return data\"}", http.StatusInternalServerError)
			return
		}
	// case "text/html":
	default:
		h.renderTemplate(w, "robots_list", "base", struct {
			OwnerUserID int64
			Ticker      string
			Robots      []*robot.Robot
		}{id, ticker, robots})
	}
}

func (h *Handler) DeleteRobotByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusInternalServerError)
		return
	}

	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || time.Now().After(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusBadRequest)

		return
	}

	robotData, err := h.robotStorage.FindByID(id)
	if err != nil {
		h.logger.Errorf("Can't get robot: %s", err)
		http.Error(w, "{\"error\": \"can't get robot\"}", http.StatusNotFound)

		return
	}

	// h.logger.Info(robotData.DeletedAt)

	if sess.UserID != robotData.OwnerUserID || robotData.DeletedAt.Valid {
		h.logger.Infof("No properties to delete")
		http.Error(w, "{\"error\": \"not available\"}", http.StatusBadRequest)

		return
	}

	err = h.robotStorage.DeleteByID(id)
	if err != nil {
		http.Error(w, "{\"error\": \"error while deleting robot\"}", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) AddRobotToFavorite(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || time.Now().After(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusBadRequest)

		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusInternalServerError)
		return
	}

	robotData, err := h.robotStorage.FindByID(id)
	if err != nil {
		h.logger.Errorf("Can't get parent robot: %s", err)
		http.Error(w, "{\"error\": \"can't get parent robot\"}", http.StatusBadRequest)

		return
	}

	robotData.OwnerUserID = sess.UserID
	robotData.ParentRobotID = id
	robotData.FactYield = 0
	robotData.DealsCount = 0
	robotData.IsFavorite = true
	robotData.CreatedAt = time.Now()

	if err = h.robotStorage.Create(robotData); err != nil {
		h.logger.Errorf("Can't add robot: %s", err)
		http.Error(w, "{\"error\": \"can't create robot\"}", http.StatusInternalServerError)

		return
	}

	h.robotsChan <- *robotData
	err = json.NewEncoder(w).Encode(robotData)

	if err != nil {
		return
	}
}

func (h *Handler) ActivateRobot(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || !time.Now().Before(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusBadRequest)

		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusBadRequest)
		return
	}

	robotData, err := h.robotStorage.FindByID(id)
	if err != nil {
		h.logger.Errorf("Can't get robot: %s", err)
		http.Error(w, "{\"error\": \"can't get robot\"}", http.StatusBadRequest)

		return
	}



	if robotData.IsActive || robotData.OwnerUserID != sess.UserID || time.Now().After(robotData.PlanStart) && time.Now().Before(robotData.PlanEnd) {
		h.logger.Errorf("Can't activate robot")
		http.Error(w, "{\"error\": \"can't activate robot now\"}", http.StatusBadRequest)

		return
	}

	if err := h.robotStorage.ActivateByID(id); err != nil {
		h.logger.Errorf("Can't activate robot: %s", err)
		http.Error(w, "{\"error\": \"error while activating robot\"}", http.StatusInternalServerError)

		return
	}

	robotData, err = h.robotStorage.FindByID(id)
	if err != nil {
		h.logger.Errorf("Can't get parent robot: %s", err)
		http.Error(w, "{\"error\": \"can't get parent robot\"}", http.StatusBadRequest)

		return
	}

	h.robotsChan <- *robotData
}

func (h *Handler) DeactivateRobot(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || time.Now().After(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusBadRequest)

		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusBadRequest)
		return
	}

	robotData, err := h.robotStorage.FindByID(id)
	if err != nil {
		h.logger.Errorf("Can't get parent robot: %s", err)
		http.Error(w, "{\"error\": \"can't get parent robot\"}", http.StatusBadRequest)

		return
	}

	if !robotData.IsActive || robotData.OwnerUserID != sess.UserID || time.Now().After(robotData.PlanStart) && time.Now().Before(robotData.PlanEnd) {
		h.logger.Errorf("Can't activate robot")
		http.Error(w, "{\"error\": \"can't activate robot now\"}", http.StatusBadRequest)
	}

	if err = h.robotStorage.DeactivateByID(id); err != nil {
		h.logger.Errorf("Can't add robot: %s", err)
		http.Error(w, "{\"error\": \"error while deactivating robot\"}", http.StatusInternalServerError)

		return
	}

	robotData, err = h.robotStorage.FindByID(id)
	if err != nil {
		h.logger.Errorf("Can't get robot: %s", err)
		http.Error(w, "{\"error\": \"can't get robot\"}", http.StatusInternalServerError)

		return
	}
	h.robotsChan <- *robotData
}

func (h *Handler) GetRobotDetails(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	sess, err := h.sessionStorage.FindByToken(token)
	if err != nil || time.Now().After(sess.ValidUntil) {
		h.logger.Errorf("Unvalid token: %s", err)
		http.Error(w, "{\"error\": \"unvalid token\"}", http.StatusBadRequest)

		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.logger.Errorf("Can't parse robot id: %s", err)
		http.Error(w, "{\"error\": \"bad data\"}", http.StatusInternalServerError)

		return
	}

	robotData, err := h.robotStorage.FindByID(id)
	if err != nil {
		h.logger.Errorf("Can't get robot data: %s", err)
		http.Error(w, "{\"error\": \"can't get robot data\"}", http.StatusNotFound)

		return
	}

	accept := r.Header.Get("Accept")

	switch accept {
	case "application/json":
		err = json.NewEncoder(w).Encode(robotData)
		if err != nil {
			http.Error(w, "{\"error\": \"can't return data\"}", http.StatusInternalServerError)
			return
		}
	default:
		h.renderTemplate(w, "robot_info", "base", robotData)
	}
}

func (h *Handler) WSRobotUpdate(w http.ResponseWriter, r *http.Request) {
	h.logger.Infof("New ws client\n")

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	defer func() {
		_ = conn.Close()
	}()
	h.wsClients.AddClient(h, conn)

	for {
		robot := <-h.robotsChan
		res, err := json.Marshal(robot)

		if err != nil {
			h.logger.Infof("can't marshal message: %+v\n", err)
			continue
		}

		h.wsClients.BroadcastMessage(h, res)
	}
}
