package service

import (
	"github.com/allin/server/base/biz/dao"
	"github.com/allin/server/base/biz/model"
	"github.com/allin/server/contrib/room"
)

type RoomSvc struct {
	manager *room.Manager
}

func NewRoomSvc(manager *room.Manager) *RoomSvc {
	return &RoomSvc{manager: manager}
}

// Create 创建新房间，由 RoomManager 分配房间码并启动游戏引擎。
func (svc *RoomSvc) Create(hostUserID int64, cfg room.RoomConfig) (*room.Room, error) {
	return svc.manager.Create(hostUserID, cfg)
}

// Get 根据房间码查询房间信息。
func (svc *RoomSvc) Get(code string) (*room.Room, error) {
	return svc.manager.Get(code)
}

// Close 关闭指定房间，停止游戏引擎并释放内存。
func (svc *RoomSvc) Close(code string) {
	svc.manager.Close(code)
}

// GetHands 返回指定房间最近 limit 手的结算摘要，按手牌编号倒序。
func (svc *RoomSvc) GetHands(code string, limit int) ([]model.HandHistoryEntry, error) {
	entries, err := dao.HandHistoryDao.GetByRoomCode(code, limit)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []model.HandHistoryEntry{}
	}
	return entries, nil
}
