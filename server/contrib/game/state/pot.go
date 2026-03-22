package state

import "sort"

// Pot 表示主池或边池。
type Pot struct {
	Amount   int64    // 池中筹码总额
	Eligible []string // 有资格赢得该池的用户 ID 列表
}

// BuildPots 根据所有玩家的 TotalBet 金额计算主池和边池。
// 应在所有下注完成后调用（未被跟注的下注已退回筹码）。
func BuildPots(seats [9]*Player) []Pot {
	type contrib struct {
		userID   string
		totalBet int64
		folded   bool
	}

	var cs []contrib
	for _, p := range seats {
		if p != nil && p.TotalBet > 0 {
			cs = append(cs, contrib{p.UserID, p.TotalBet, p.Folded})
		}
	}
	if len(cs) == 0 {
		return nil
	}

	sort.Slice(cs, func(i, j int) bool {
		return cs[i].totalBet < cs[j].totalBet
	})

	var pots []Pot
	prevCap := int64(0)

	for _, c := range cs {
		if c.totalBet <= prevCap {
			continue
		}
		cap := c.totalBet
		amount := int64(0)
		var eligible []string

		for _, c2 := range cs {
			lo := min64(c2.totalBet, prevCap)
			hi := min64(c2.totalBet, cap)
			if hi > lo {
				amount += hi - lo
			}
			if !c2.folded && c2.totalBet >= cap {
				eligible = append(eligible, c2.userID)
			}
		}

		if amount > 0 {
			pots = append(pots, Pot{Amount: amount, Eligible: eligible})
		}
		prevCap = cap
	}

	return pots
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
