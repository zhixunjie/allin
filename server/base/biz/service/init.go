package service

import "github.com/allin/server/contrib/room"

var (
	User *UserSvc
	Room *RoomSvc
)

// Init initialises all service singletons.
// roomManager is created in main and injected here to avoid a package-level cycle.
func Init(roomManager *room.Manager) {
	User = NewUserSvc()
	Room = NewRoomSvc(roomManager)
}
