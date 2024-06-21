package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

type Room struct {
	RoomId     string          `json:"roomId"`
	MaxMembers int             `json:"maxMembers"`
	RoomOwner  ConnectedUser   `json:"roomOwner"`
	Users      []ConnectedUser `json:"users"`
}

type Rooms map[string]Room

var RoomList Rooms

func (rooms *Rooms) createNewRoom(roomId string, maxMembers int, owner *ConnectedUser) (*Room, error) {

	newRoom := Room{RoomId: roomId, MaxMembers: maxMembers, RoomOwner: *owner}
	newRoom.RoomOwner.connection.WriteMessage(websocket.TextMessage, []byte("test"))

	if (*rooms) == nil {
		(*rooms) = make(map[string]Room)
	}

	_, exists := (*rooms)[roomId]

	if exists {
		return nil, errors.New("Room already exists")
	}

	(*rooms)[roomId] = newRoom
	owner.IsRoomOwner = true

	return &newRoom, nil
}

func (room *Room) notifyOwner() {
	roomMembers, error := json.Marshal(room.Users)

	if error != nil {
		panic(error)
	}

	room.RoomOwner.connection.WriteMessage(websocket.TextMessage, roomMembers)
}

func (rooms *Rooms) joinRoom(roomId string, user ConnectedUser) error {
	if entry, ok := (*rooms)[roomId]; ok {

		for _, v := range entry.Users {
			if v.UserId == user.UserId {
				return errors.New("User already in room")
			}
		}

		entry.Users = append(entry.Users, user)
		(*rooms)[roomId] = entry
		entry.notifyOwner()

		return nil
	}
	return errors.New("Room not found")
}

func (rooms *Rooms) removeUser(user ConnectedUser) {
	fmt.Println()
	if user.IsRoomOwner {
		var roomIdToDelete string

		for roomId := range *rooms {
			if (*rooms)[roomId].RoomOwner.UserId == user.UserId {
				roomIdToDelete = roomId
				break
			}
		}

		err := rooms.deleteRoom(roomIdToDelete)

		if err != nil {
			fmt.Println(err)
		}

	} else {
		for roomId := range *rooms {
			rooms.leaveRoom(roomId, user)
		}
	}
}

func (rooms *Rooms) leaveRoom(roomId string, user ConnectedUser) error {
	if _, ok := (*rooms)[roomId]; !ok {
		return errors.New("Room not found")
	}

	userIndex := -1

	for i, entry := range (*rooms)[roomId].Users {
		if entry.UserId.String() == user.UserId.String() {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		return errors.New("User not in room")
	}

	room := (*rooms)[roomId]
	roomMemberCount := len((*rooms)[roomId].Users)

	for i := userIndex; i < roomMemberCount-1; i++ {
		room.Users[i] = room.Users[i+1]
	}
	room.Users = room.Users[:roomMemberCount-1]
	(*rooms)[roomId] = room

	room.notifyOwner()

	return nil
}

func (rooms *Rooms) deleteRoom(roomId string) error {
	if _, ok := (*rooms)[roomId]; !ok {
		return errors.New("Room not found")
	}

	usersToNotify := (*rooms)[roomId].Users
	for _, user := range usersToNotify {
		user.connection.WriteMessage(websocket.TextMessage, []byte("room closed"))
	}
	delete(*rooms, roomId)

	return nil

}
