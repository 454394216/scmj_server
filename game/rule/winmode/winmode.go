package winmode

import (
	"zinx-mj/game/rule/irule"
	"zinx-mj/game/table/tableoperate"
)

const (
	WIN_DRAW_MODE_MIN   = iota
	WIN_DRAW_MODE_PLAIN // 普通自摸
	WIN_DRAW_MODE_GOD   // 天胡
	WIN_DRAW_MODE_KONG  // 杠上花
	WIN_DRAW_MODE_MAX   // 哨兵

	WIN_DISCARD_MODE_MIN
	WIN_DISCARD_MODE_PLAIN    // 普通放炮
	WIN_DISCARD_MODE_DEVIL    // 地胡
	WIN_DISCARD_MODE_RUB_KONG // 抢杠
	WIN_DISCARD_MODE_KONG     // 杠上炮
	WIN_DISCARD_MODE_MAX
)

type WinModeModel struct {
}

func NewWinModeModel() *WinModeModel {
	return &WinModeModel{}
}

func (w *WinModeModel) GetWinMode(info irule.WinModeInfo) int {
	if info.DrawWin { // 自摸
		if len(info.TurnDraw) == 1 && info.Dealer == info.WinPid { // 庄家第一张牌
			return WIN_DRAW_MODE_GOD
		} else {
			if len(info.TurnOps) >= 2 {
				lastOp := info.TurnOps[len(info.TurnOps)-2].OpType // 上上一个是扛
				if lastOp == tableoperate.OPERATE_KONG_CONCEALED || lastOp == tableoperate.OPERATE_KONG_EXPOSED || lastOp == tableoperate.OPERATE_KONG_RAIN {
					return WIN_DRAW_MODE_KONG
				}
			}
			return WIN_DRAW_MODE_PLAIN
		}
	} else { // 点炮
		if len(info.Discards) == 1 {
			return WIN_DISCARD_MODE_DEVIL
		} else {
			lastOp := info.TurnOps[len(info.TurnOps)-1].OpType
			if lastOp == tableoperate.OPERATE_KONG_EXPOSED {
				return WIN_DISCARD_MODE_RUB_KONG
			}
			if len(info.TurnOps) >= 2 {
				// 杠上炮的话，当前玩家上上个是杠的动作
				prevOp := info.TurnOps[len(info.TurnOps)-2].OpType
				if prevOp == tableoperate.OPERATE_KONG_EXPOSED || prevOp == tableoperate.OPERATE_KONG_CONCEALED || prevOp == tableoperate.OPERATE_KONG_RAIN {
					return WIN_DISCARD_MODE_KONG
				}
			}
			return WIN_DISCARD_MODE_PLAIN
		}
	}
}

func IsDrawWin(mode int) bool {
	return WIN_DRAW_MODE_MIN < mode && mode < WIN_DRAW_MODE_MAX
}
