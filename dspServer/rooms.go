package main

import (
	"encoding/json"
	"errors"

	"github.com/lesismal/nbio/nbhttp/websocket"
)

type Room struct {
	RoomId     string          `json:"roomId"`
	MaxMembers int             `json:"maxMembers"`
	Users      []ConnectedUser `json:"users"`
}

type Rooms map[string]Room

var RoomList Rooms

func (rooms *Rooms) createNewRoom(roomId string, maxMembers int) (*Room, error) {

	newRoom := Room{RoomId: roomId, MaxMembers: maxMembers}

	if (*rooms) == nil {
		(*rooms) = make(map[string]Room)
	}

	_, exists := (*rooms)[roomId]

	if exists {
		return nil, errors.New("Room already exists")
	}

	(*rooms)[roomId] = newRoom

	return &newRoom, nil
}

func (room *Room) notifyMembers() {
	for _, v := range room.Users {
		value, err := json.Marshal(room)

		if err != nil {
			panic(err)
		}

		v.connection.WriteMessage(websocket.TextMessage, value)
	}
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
		entry.notifyMembers()

		return nil
	}
	return errors.New("Room not found")
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
	for i := userIndex; i < len((*rooms)[roomId].Users)-1; i++ {
		room.Users[i] = room.Users[i+1]
	}
	(*rooms)[roomId] = room

	return nil
}
