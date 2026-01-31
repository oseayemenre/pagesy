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

type hub struct {
	admins     map[*websocket.Conn]bool
	connect    chan *websocket.Conn
	disconnect chan *websocket.Conn
	broadcast  chan interface{}
}

func newHub() *hub {
	return &hub{
		admins:     make(map[*websocket.Conn]bool),
		connect:    make(chan *websocket.Conn),
		disconnect: make(chan *websocket.Conn),
		broadcast:  make(chan interface{}),
	}
}

func (s *server) run() {
	for {
		select {
		case conn := <-s.hub.connect:
			s.hub.admins[conn] = true
		case conn := <-s.hub.disconnect:
			conn.Close()
			delete(s.hub.admins, conn)
		case event := <-s.hub.broadcast:
			for conn := range s.hub.admins {
				if err := conn.WriteJSON(event); err != nil {
					continue
				}
			}
		}
	}
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
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

	go s.run()
	s.hub.connect <- conn
	defer func() {
		s.hub.disconnect <- conn
	}()

	var event event

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			return
		}

		if err := json.Unmarshal(message, &event); err != nil {
			if err := conn.WriteJSON(&errorResponse{Error: fmt.Sprintf("unable to marshal json, %v", err)}); err != nil {
				return
			}
			continue
		}

		switch event.Type {
		case NEW_BOOK:
			s.hub.broadcast <- &event
		}
	}
}
