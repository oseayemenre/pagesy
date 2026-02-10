package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

type eventType int

const (
	NEW_BOOK eventType = iota
)

func (e eventType) String() string {
	switch e {
	case NEW_BOOK:
		return "NEW_BOOK"
	}
	return "Unknown event"
}

type event struct {
	Type    eventType
	Payload interface{}
}

type client struct {
	id   string
	conn *websocket.Conn
}

type hub struct {
	regular           map[string]*websocket.Conn
	admins            map[string]*websocket.Conn
	connectAdmin      chan *client
	connectRegular    chan *client
	disconnectAdmin   chan string
	disconnectRegular chan string
	broadcast         chan *event
}

func newHub() *hub {
	return &hub{
		regular:           make(map[string]*websocket.Conn),
		admins:            make(map[string]*websocket.Conn),
		connectAdmin:      make(chan *client),
		connectRegular:    make(chan *client),
		disconnectAdmin:   make(chan string),
		disconnectRegular: make(chan string),
		broadcast:         make(chan *event),
	}
}

func (s *server) handleNewBookEvent(event *event) {
	for _, conn := range s.hub.admins {
		if err := conn.WriteJSON(event); err != nil {
			continue
		}
	}
}

func (s *server) run() {
	for {
		select {
		case client := <-s.hub.connectAdmin:
			s.hub.admins[client.id] = client.conn
		case client := <-s.hub.connectRegular:
			s.hub.regular[client.id] = client.conn
		case id := <-s.hub.disconnectAdmin:
			if conn, ok := s.hub.admins[id]; ok {
				conn.Close()
				delete(s.hub.admins, id)
			}
		case id := <-s.hub.disconnectRegular:
			if conn, ok := s.hub.regular[id]; ok {
				conn.Close()
				delete(s.hub.regular, id)
			}
		case event := <-s.hub.broadcast:
			switch event.Type {
			case NEW_BOOK:
				s.handleNewBookEvent(event)
			}
		}
	}
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user").(string)
	user, err := s.getUser(r.Context(), userID)
	if err != nil {
		s.logger.Error(err.Error())
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error upgrading ws connection, %v", err))
		encode(w, http.StatusInternalServerError, &errorResponse{Error: "internal server error"})
		return
	}

	var isAdmin bool
	for _, role := range user.roles {
		if role == "ADMIN" {
			isAdmin = true
			break
		}
	}

	if isAdmin == true {
		s.hub.connectAdmin <- &client{id: userID, conn: conn}
	} else {
		s.hub.connectRegular <- &client{id: userID, conn: conn}
	}

	defer func() {
		if isAdmin == true {
			s.hub.disconnectAdmin <- userID
		} else {
			s.hub.disconnectRegular <- userID
		}
	}()

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			break
		}

		var evt event
		if err := json.Unmarshal(message, &evt); err != nil {
			if err := conn.WriteJSON(&errorResponse{Error: fmt.Sprintf("unable to unmarshal json, %v", err)}); err != nil {
				break
			}
			continue
		}

		switch evt.Type {
		case NEW_BOOK:
			s.hub.broadcast <- &evt
		default:
			if err := conn.WriteMessage(websocket.TextMessage, []byte("event not found")); err != nil {
				return
			}
		}
	}
}
