package boardcard

import (
	"fmt"
	"zinx-mj/game/gamedefine"
)

// 牌桌的麻将牌
type BoardCard struct {
	CardsTotal   []int // 总的牌
	Cards        []int // 剩余的牌
	DiscardCards []int // 玩家已出的牌
	DrawCards    []int // 玩家已摸的牌
	CardSuits    []int // 花色
}

/*
 * Descrp: 创建一组花色牌
 * Param: cardType 花色类型
 * Create: zhangyi 2020-07-02 18:18:49
 */
func NewBoardCardBySuit(cardSuits ...int) *BoardCard {
	bc := &BoardCard{}
	for _, cardType := range cardSuits {
		for i := 1; i <= 9; i++ { // 1到9
			for j := 0; j < 4; j++ { // 每张牌4张
				bc.CardsTotal = append(bc.CardsTotal, cardType*gamedefine.CARD_BASE+i)
			}
		}
	}
	bc.Cards = make([]int, len(bc.CardsTotal))
	copy(bc.Cards, bc.CardsTotal)
	bc.CardSuits = cardSuits
	return bc
}

func (bc *BoardCard) GetSuits() []int {
	return bc.CardSuits
}

/*
 * Descrp: 从前面摸一张牌
 * Create: zhangyi 2020-08-02 15:03:14
 */
func (b *BoardCard) DrawForward() (int, error) {
	cards, err := b.DrawForwardMult(1)
	if err != nil {
		return 0, err
	}
	return cards[0], nil
}

/*
 * Descrp: 从前面摸多张牌
 * Create: zhangyi 2020-08-02 15:03:14
 */
func (b *BoardCard) DrawForwardMult(num int) ([]int, error) {
	var draws []int
	if b.GetLeftDrawCardNum() < num {
		return draws, fmt.Errorf("left card num not enough, need=%d, left=%d", b.GetLeftDrawCardNum(), num)
	}
	draws = b.Cards[:num]
	b.Cards = b.Cards[num:]
	b.DrawCards = append(b.DrawCards, draws...)
	return draws, nil
}

/*
 * Descrp: 从后面摸一张牌
 * Create: zhangyi 2020-08-02 15:03:14
 */
func (b *BoardCard) DrawBackward() (int, error) {
	cards, err := b.DrawBackwardMult(1)
	if err != nil {
		return 0, err
	}
	return cards[0], nil
}

/*
 * Descrp: 从前面摸多张牌
 * Create: zhangyi 2020-08-02 15:03:14
 */
func (b *BoardCard) DrawBackwardMult(num int) ([]int, error) {
	var draws []int
	totalNum := b.GetLeftDrawCardNum()
	if totalNum < num {
		return draws, fmt.Errorf("left card num not enough, need=%d, left=%d", totalNum, num)
	}
	draws = b.Cards[totalNum-num:]
	b.Cards = b.Cards[:totalNum-num]
	b.DrawCards = append(b.DrawCards, draws...)
	return draws, nil
}

/*
 * Descrp: 剩余的可摸牌数量
 * Create: zhangyi 2020-08-02 15:16:37
 */
func (b *BoardCard) GetLeftDrawCardNum() int {
	return len(b.Cards)
}
