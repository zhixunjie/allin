package service

import "github.com/allin/server/contrib/room"

type RoomSvc struct {
	manager *room.Manager
}

func NewRoomSvc(manager *room.Manager) *RoomSvc {
	return &RoomSvc{manager: manager}
}

func (svc *RoomSvc) Create(hostUserID int64, cfg room.RoomConfig) (*room.Room, error) {
	return svc.manager.Create(hostUserID, cfg)
}

func (svc *RoomSvc) Get(code string) (*room.Room, error) {
	return svc.manager.Get(code)
}

func (svc *RoomSvc) Close(code string) {
	svc.manager.Close(code)
}
