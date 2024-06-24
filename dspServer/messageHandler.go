package main

import (
	"encoding/json"
	"fmt"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

type NetworkError struct {
	Message string `json:"errorMessage"`
	Action  string `json:"action"`
}

var NetworkErr NetworkError

func (err *NetworkError) Error() string {
	return fmt.Sprintf("%s while trying to execute action %s", err.Message, err.Action)
}

func (err *NetworkError) ToJSON() []byte {
	val, marshErr := json.Marshal(err)
	if marshErr != nil {
		panic(marshErr)
	}
	return val
}

func (err *NetworkError) New(message string, action string) *NetworkError {
	return &NetworkError{Message: message, Action: action}
}

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

type MessageFunctionHandler[MessageType any] func(payload MessageType, user *ConnectedUser) ([]byte, *NetworkError)

func GetPayload[outType any](socketMessage *SocketMessage) (outType, *NetworkError) {
	var payload outType
	payloadStr, err := json.Marshal(socketMessage.Payload)

	if err != nil {
		return payload, NetworkErr.New("unable to marshal", "payload")
	}

	unmarshErr := json.Unmarshal(payloadStr, &payload)

	if unmarshErr != nil {
		return payload, NetworkErr.New("incorrect message structrue", "payload")
	}

	return payload, nil
}

func HandleSpecificMessage[MessageType any](user *ConnectedUser, receivedMessage *SocketMessage, c *websocket.Conn, handleFunc MessageFunctionHandler[MessageType]) {
	payload, err := GetPayload[MessageType](receivedMessage)

	if err != nil {
		c.WriteMessage(websocket.TextMessage, err.ToJSON())
		return
	}

	res, err := handleFunc(payload, user)

	if err != nil {
		c.WriteMessage(websocket.TextMessage, err.ToJSON())
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
		c.WriteMessage(websocket.TextMessage, NetworkErr.New("unknown format", "types").ToJSON())
	}
}

func HandleNewRoomEvent(payload NewRoomMessage, owner *ConnectedUser) ([]byte, *NetworkError) {
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

func HandleJoinRoomEvent(payload JoinRoomMessage, user *ConnectedUser) ([]byte, *NetworkError) {
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

func HandleLeaveRoomEvent(payload LeaveRoomMessage, user *ConnectedUser) ([]byte, *NetworkError) {
	error := RoomList.leaveRoom(payload.RoomId, *user)

	if error != nil {
		return []byte(""), error
	}

	return []byte(""), nil
}

func HandleUserInitEvent(payload InitUserMessage, user *ConnectedUser) ([]byte, *NetworkError) {
	user.SetDSP(payload.UserDSP)
	return []byte(""), nil
}
