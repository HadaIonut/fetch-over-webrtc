package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

var (
	upgrader = newUpgrader()
)

type MessageTypes int

const (
	NewRoom = iota
	JoinRoom
	LeaveRoom
)

type ConnectedUser struct {
	UserId     uuid.UUID `json:"userId"`
	connection *websocket.Conn
}

func (User *ConnectedUser) GetJson() []byte {
	json, err := json.Marshal(User)

	if err != nil {
		return []byte("")
	}

	return json
}

func handleMessge(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
	if messageType != 1 {
		c.WriteMessage(websocket.TextMessage, []byte("unsupported message type"))
		return
	}

	receivedMessage := SocketMessage{}
	err := json.Unmarshal(data, &receivedMessage)

	HandleKnownMessages(&receivedMessage, c)

	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("invalid json"))
		return
	}
}

func HandleUserHearthBeat(c *websocket.Conn) chan bool {
	ticker := time.NewTicker(1 * time.Minute)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				fmt.Println("timer done")
				return
			case <-ticker.C:
				c.WriteMessage(websocket.PingMessage, []byte("ping"))
			}
		}
	}()
	return done
}

func newUpgrader() *websocket.Upgrader {
	u := websocket.NewUpgrader()
	var heartBeatTimer chan bool

	u.OnOpen(func(c *websocket.Conn) {
		NewUser := ConnectedUser{UserId: uuid.New(), connection: c}
		c.SetSession(NewUser)
		heartBeatTimer = HandleUserHearthBeat(c)

		fmt.Println("OnOpen:", c.RemoteAddr().String())
		c.WriteMessage(websocket.TextMessage, NewUser.GetJson())
	})

	u.OnMessage(handleMessge)

	u.OnClose(func(c *websocket.Conn, err error) {
		heartBeatTimer <- true
		fmt.Println(c.Session().(ConnectedUser).UserId)
		fmt.Println("OnClose:", c.RemoteAddr().String(), err)
	})
	return u
}

func onWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("Upgraded:", conn.RemoteAddr().String())
}

func main() {
	mux := &http.ServeMux{}
	mux.HandleFunc("/ws", onWebsocket)
	engine := nbhttp.NewEngine(nbhttp.Config{
		Network:                 "tcp",
		Addrs:                   []string{"localhost:8080"},
		MaxLoad:                 1000000,
		ReleaseWebsocketPayload: true,
		Handler:                 mux,
	})

	err := engine.Start()
	if err != nil {
		fmt.Printf("nbio.Start failed: %v\n", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	engine.Shutdown(ctx)
}
