package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var rooms = make(map[string]*Room)

type ChatMessage struct {
	User string `json:"user"`
	Text string `json:"text"`
}
type User struct {
	Conn *websocket.Conn
	Name string
}

func (user *User) SendMessage(chatMessage *ChatMessage) error {
	log.Println(user.Name, "Sending message:", chatMessage)
	if err := user.Conn.WriteJSON(chatMessage); err != nil {
		log.Println(err)
		return err
	}
	return nil
}
func (user *User) ReceiveMessage() (ChatMessage, error) {
	cm := ChatMessage{}
	if err := user.Conn.ReadJSON(&cm); err != nil {
		log.Println(user.Name, "has error:", err)
		return cm, err
	}
	log.Println(user.Name, "Received message:", cm)
	return cm, nil
}

type Room struct {
	Channel chan ChatMessage
	Users   []*User
}

func (room *Room) Broadcast() {
	go func() {
		for {
			msg := <-room.Channel
			log.Println("Taking off channel")
			for _, user := range room.Users {
				log.Println("sending to", user)
				if err := user.SendMessage(&msg); err != nil {
					room.RemoveUser(user)
				}
			}
		}
	}()
}
func (room *Room) AddUser(user *User) {
	userClose := func(code int, text string) error {
		room.RemoveUser(user)
		return nil
	}
	user.Conn.SetCloseHandler(userClose)
	room.Users = append(room.Users, user)
	go func() {
		for {
			cm, err := user.ReceiveMessage()
			if err != nil {
				log.Println("Receive message issue, removing user", user.Name)
				room.RemoveUser(user)
				break
			}
			room.Channel <- cm
			log.Println("Writing to channel")
		}
	}()
}
func (room *Room) RemoveUser(user *User) {
	log.Println("Removing user ", user)
	for i, userVal := range room.Users {
		if userVal == user {
			room.Users = append(room.Users[:i], room.Users[i+1:]...)
			break
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func homepage(w http.ResponseWriter, r *http.Request) {
	log.Println(r)
	fmt.Fprint(w, "Home page")
}
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Println(r)
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	roomParam := r.URL.Query().Get("room")
	nameParam := r.URL.Query().Get("user")
	log.Println("New Connection request: room:", roomParam, "name", nameParam)
	if len(roomParam) == 0 || len(nameParam) == 0 {
		w.WriteHeader(400)
		fmt.Fprint(w, "Invalid Request Parameters")
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	user := User{
		Name: nameParam,
		Conn: ws,
	}
	if room, ok := rooms[roomParam]; ok {
		room.AddUser(&user)
	} else {
		newRoom := Room{Channel: make(chan ChatMessage)}
		newRoom.AddUser(&user)
		newRoom.Broadcast()
		rooms[roomParam] = &newRoom
	}
	log.Println("Client Successfully Connected...")
	log.Println(rooms)
}

func setUpRoutes() {
	http.HandleFunc("/", homepage)
	http.HandleFunc("/ws", wsEndpoint)
}
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Go WebSockets")
	setUpRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
