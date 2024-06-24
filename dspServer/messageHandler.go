package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

type NewRoomMessage struct {
	RoomId     string `json:"roomId"`
	MaxMembers int    `json:"maxMembers"`
}
type JoinRoomMessage struct {
	RoomId string `json:"roomId"`
}
type LeaveRoomMessage struct {
	RoomId string `json:"roomId"`
}
type InitUserMessage struct {
	UserDSP string `json:"userDSP"`
}
type SocketMessage struct {
	TypeName MessageTypes
	Payload  interface{}
}

type MessageFunctionHandler[MessageType any] func(payload MessageType, user *ConnectedUser) ([]byte, error)

func GetPayload[outType any](socketMessage *SocketMessage) (outType, error) {
	var payload outType
	payloadStr, err := json.Marshal(socketMessage.Payload)

	if err != nil {
		return payload, errors.New("unable to marshal")
	}

	unmarshErr := json.Unmarshal(payloadStr, &payload)

	if unmarshErr != nil {
		return payload, errors.New("incorrect message structrue")
	}

	return payload, nil
}

func HandleSpecificMessage[MessageType any](user *ConnectedUser, receivedMessage *SocketMessage, c *websocket.Conn, handleFunc MessageFunctionHandler[MessageType]) {
	payload, err := GetPayload[MessageType](receivedMessage)

	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}

	res, err := handleFunc(payload, user)

	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}

	c.WriteMessage(websocket.TextMessage, res)
	return
}

func HandleKnownMessages(receivedMessage *SocketMessage, c *websocket.Conn) {
	user := c.Session().(ConnectedUser)
	fmt.Println(InitUser, receivedMessage.TypeName)

	switch receivedMessage.TypeName {
	case NewRoom:
		HandleSpecificMessage(&user, receivedMessage, c, HandleNewRoomEvent)
		c.SetSession(user)
		return
	case JoinRoom:
		HandleSpecificMessage(&user, receivedMessage, c, HandleJoinRoomEvent)
		c.SetSession(user)
		return
	case LeaveRoom:
		HandleSpecificMessage(&user, receivedMessage, c, HandleLeaveRoomEvent)
		c.SetSession(user)
		return
	case InitUser:
		HandleSpecificMessage(&user, receivedMessage, c, HandleUserInitEvent)
		c.SetSession(user)
		return
	default:
		c.WriteMessage(websocket.TextMessage, []byte("unknown format"))
	}
}

func HandleNewRoomEvent(payload NewRoomMessage, owner *ConnectedUser) ([]byte, error) {
	newRoom, error := RoomList.createNewRoom(payload.RoomId, payload.MaxMembers, owner)

	if error != nil {
		return []byte(""), error
	}

	roomBytes, err := json.Marshal(*newRoom)

	if err != nil {
		panic(err)
	}

	fmt.Println("roomList:", RoomList, newRoom)

	return roomBytes, nil
}

func HandleJoinRoomEvent(payload JoinRoomMessage, user *ConnectedUser) ([]byte, error) {
	room, error := RoomList.joinRoom(payload.RoomId, *user)

	if error != nil {
		return []byte(""), error
	}

	roomBytes, err := json.Marshal(room)

	if err != nil {
		panic(err)
	}

	return roomBytes, nil
}

func HandleLeaveRoomEvent(payload LeaveRoomMessage, user *ConnectedUser) ([]byte, error) {
	error := RoomList.leaveRoom(payload.RoomId, *user)

	if error != nil {
		return []byte(""), error
	}

	return []byte(""), nil
}

func HandleUserInitEvent(payload InitUserMessage, user *ConnectedUser) ([]byte, error) {
	fmt.Println(payload)
	user.SetDSP(payload.UserDSP)
	return []byte(""), nil
}
