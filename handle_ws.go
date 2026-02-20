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
	CHAPTER_UPLOADED
)

func (e eventType) String() string {
	switch e {
	case NEW_BOOK:
		return "NEW_BOOK"
	case CHAPTER_UPLOADED:
		return "CHAPTER_UPLOADED"
	}
	return "Unknown event"
}

type event struct {
	Type    eventType
	Payload interface{}
}

type chapterUploadEvent struct {
	BookId  string
	Message string
}

type client struct {
	id   string
	conn *websocket.Conn
	send chan []byte
}

type roomUser struct {
	roomID string
	userID string
}

type hub struct {
	regular            map[string]*client
	admins             map[string]*client
	rooms              map[string]map[string]*client
	joinRoom           chan *roomUser
	connectAdmin       chan *client
	connectRegular     chan *client
	disconnectAdmin    chan string
	disconnectRegular  chan string
	disconnectRoomUser chan *roomUser
	broadcast          chan *event
}

func newHub() *hub {
	return &hub{
		regular:            make(map[string]*client),
		admins:             make(map[string]*client),
		rooms:              make(map[string]map[string]*client),
		joinRoom:           make(chan *roomUser),
		connectAdmin:       make(chan *client),
		connectRegular:     make(chan *client),
		disconnectAdmin:    make(chan string),
		disconnectRegular:  make(chan string),
		disconnectRoomUser: make(chan *roomUser),
		broadcast:          make(chan *event),
	}
}

// writing to pump because some connections can be slow
func (c *client) writePump() {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (s *server) handleNewBookEvent(event *event) {
	body, err := json.Marshal(event)
	if err != nil {
		return
	}
	for _, client := range s.hub.admins {
		client.send <- body
	}
}

func (s *server) handleNewChapterUploadedEvent(event *event) {
	payload := event.Payload.(chapterUploadEvent)

	if room, ok := s.hub.rooms[payload.BookId]; ok {
		for _, client := range room {
			client.send <- []byte(payload.Message)
		}
	}
}

func (s *server) run() {
	for {
		select {
		case client := <-s.hub.connectAdmin:
			s.hub.admins[client.id] = client
		case client := <-s.hub.connectRegular:
			s.hub.regular[client.id] = client
		case ru := <-s.hub.joinRoom:
			if s.hub.rooms[ru.roomID] == nil {
				s.hub.rooms[ru.roomID] = map[string]*client{}
			}
			if client, ok := s.hub.regular[ru.userID]; ok {
				s.hub.rooms[ru.roomID][ru.userID] = client
			}

		case id := <-s.hub.disconnectAdmin:
			if client, ok := s.hub.admins[id]; ok {
				delete(s.hub.admins, id)
				client.conn.Close()
			}
		case id := <-s.hub.disconnectRegular:
			if client, ok := s.hub.regular[id]; ok {
				delete(s.hub.regular, id)
				client.conn.Close()
			}
		case room := <-s.hub.disconnectRoomUser:
			delete(s.hub.rooms[room.roomID], room.userID)
		case event := <-s.hub.broadcast:
			switch event.Type {
			case NEW_BOOK:
				s.handleNewBookEvent(event)
			case CHAPTER_UPLOADED:
				s.handleNewChapterUploadedEvent(event)
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

	newClient := &client{id: userID, conn: conn, send: make(chan []byte)}
	go newClient.writePump()

	if isAdmin == true {
		s.hub.connectAdmin <- newClient
	} else {
		s.hub.connectRegular <- newClient
	}

	// Add client to book room if he has book in his library
	bookIDs, err := s.getUserLibrary(r.Context(), userID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error getting book, %v", err))
		conn.WriteMessage(websocket.CloseMessage, []byte("internal server error"))
		return
	}
	for _, bookID := range bookIDs {
		s.hub.joinRoom <- &roomUser{roomID: bookID, userID: userID}
		defer func(id string) {
			s.hub.disconnectRoomUser <- &roomUser{
				roomID: id,
				userID: userID,
			}
		}(bookID)
	}

	defer func() {
		if isAdmin {
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
	}
}
