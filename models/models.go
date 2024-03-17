package models

import "time"

type ResponseOrderStruct struct {
	OrderId                 int64  `json:"orderId"`
	Symbol                  string `json:"symbol"`
	Status                  string `json:"status"`
	ClientOrderId           string `json:"clientOrderId"`
	Price                   string `json:"price"`
	AvgPrice                string `json:"avgPrice"`
	OrigQty                 string `json:"origQty"`
	ExecutedQty             string `json:"executedQty"`
	CumQty                  string `json:"cumQty"`
	CumQuote                string `json:"cumQuote"`
	TimeInForce             string `json:"timeInForce"`
	Type                    string `json:"type"`
	ReduceOnly              bool   `json:"reduceOnly"`
	ClosePosition           bool   `json:"closePosition"`
	Side                    string `json:"side"`
	PositionSide            string `json:"positionSide"`
	StopPrice               string `json:"stopPrice"`
	WorkingType             string `json:"workingType"`
	PriceProtect            bool   `json:"priceProtect"`
	OrigType                string `json:"origType"`
	PriceMatch              string `json:"priceMatch"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode"`
	GoodTillDate            int    `json:"goodTillDate"`
	UpdateTime              int64  `json:"updateTime"`
}

type CryptoPosition struct {
	EntryPrice       string `json:"entryPrice"`
	BreakEvenPrice   string `json:"breakEvenPrice"`
	MarginType       string `json:"marginType"`
	IsAutoAddMargin  string `json:"isAutoAddMargin"`
	IsolatedMargin   string `json:"isolatedMargin"`
	Leverage         string `json:"leverage"`
	LiquidationPrice string `json:"liquidationPrice"`
	MarkPrice        string `json:"markPrice"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	PositionAmt      string `json:"positionAmt"`
	Notional         string `json:"notional"`
	IsolatedWallet   string `json:"isolatedWallet"`
	Symbol           string `json:"symbol"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	PositionSide     string `json:"positionSide"`
	UpdateTime       int64  `json:"updateTime"`
}

type PriceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
	Time   int64  `json:"time"`
}

type DeleteResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
type Historico struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Value     string    `gorm:"column:value" json:"value"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type Bots struct {
	Coin string `json:"coin"`
}

type BotHistory struct {
	AccountKey    string    `gorm:"column:account_key" json:"account_key"`
	HistDate      time.Time `gorm:"column:hist_date" json:"hist_date"`
	CurrValue     float64   `gorm:"column:curr_value" json:"curr_value"`
	Command       string    `gorm:"column:command" json:"command"`
	CommandParams string    `gorm:"column:commmand_params" json:"commmand_params"`
	AccumRoi      float64   `gorm:"column:accum_roi" json:"accum_roi"`
	TradingName   string    `gorm:"column:trading_name" json:"trading_name"`
}

type ResponseQuery struct {
	HistDate  time.Time `gorm:"hist_date" json:"hist_date"`
	Coin      string    `gorm:"coin" json:"coin"`
	Tend      string    `gorm:"tend" json:"tend"`
	CurrValue float64   `gorm:"curr_value" json:"curr_value"`
	TP        float64   `gorm:"SP" json:"SP"`
	SL        float64   `gorm:"SL" json:"SL"`
}

type HistoricoAll struct {
	HistDate     time.Time `gorm:"column:hist_date" json:"hist_date"`
	TradingName  string    `gorm:"column:trading_name" json:"trading_name"`
	CurrentValue string    `gorm:"column:curr_value" json:"curr_value"`
}

func (Historico) TableName() string {
	return "historico"
}

func (BotHistory) TableName() string {
	return "bot_history"
}

func (ResponseQuery) TableName() string {
	return "v_selected_orders"
}

func (HistoricoAll) TableName() string { return "hist_trading_values" }
