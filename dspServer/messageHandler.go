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

type SocketMessage struct {
	TypeName MessageTypes
	Payload  interface{}
}
type MessageFunctionHandler[MessageType any] func(payload MessageType, user ConnectedUser) error

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

func HandleSpecificMessage[MessageType any](user ConnectedUser, receivedMessage *SocketMessage, c *websocket.Conn, handleFunc MessageFunctionHandler[MessageType]) {
	payload, err := GetPayload[MessageType](receivedMessage)

	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}

	err = handleFunc(payload, user)

	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}

	c.WriteMessage(websocket.TextMessage, []byte("new room received"))
	return
}

func HandleKnownMessages(receivedMessage *SocketMessage, c *websocket.Conn) {
	user := c.Session().(ConnectedUser)

	switch receivedMessage.TypeName {
	case NewRoom:
		HandleSpecificMessage(user, receivedMessage, c, HandleNewRoomEvent)
		return
	case JoinRoom:
		HandleSpecificMessage(user, receivedMessage, c, HandleJoinRoomEvent)
		return
	case LeaveRoom:
		HandleSpecificMessage(user, receivedMessage, c, HandleLeaveRoomEvent)
		return
	default:
		c.WriteMessage(websocket.TextMessage, []byte("unknown format"))
	}
}

func HandleNewRoomEvent(payload NewRoomMessage, owner ConnectedUser) error {
	newRoom, error := RoomList.createNewRoom(payload.RoomId, payload.MaxMembers)

	if error != nil {
		return error
	}

	error = RoomList.joinRoom(payload.RoomId, owner)

	if error != nil {
		return error
	}
	fmt.Println("roomList:", RoomList, newRoom)

	return nil
}

func HandleJoinRoomEvent(payload JoinRoomMessage, user ConnectedUser) error {
	error := RoomList.joinRoom(payload.RoomId, user)

	if error != nil {
		return error
	}

	return nil
}

func HandleLeaveRoomEvent(payload LeaveRoomMessage, user ConnectedUser) error {
	error := RoomList.leaveRoom(payload.RoomId, user)

	if error != nil {
		return error
	}

	return nil
}
