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

type ResponseBookTicker struct {
	Symbol       string `json:"symbol"`
	BidPrice     string `json:"bidPrice"`
	BidQty       string `json:"bidQty"`
	AskPrice     string `json:"askPrice"`
	Time         int64  `json:"time"`
	LastUpdateID int64  `json:"lastUpdateId"`
}

type UserStruct struct {
	ApiKey      string `json:"apiKey"`
	SecretKey   string `json:"secretKey"`
	BaseURL     string `json:"baseURL"`
	Development bool   `json:"development"`
	Host        string `json:"host"`
	User        string `json:"user"`
	Pass        string `json:"pass"`
	Port        string `json:"port"`
	Dbname      string `json:"dbname"`
	TabelaHist  string `json:"tabelaHist"`
	AlertasDisc string `json:"alertasDisc"`
	WakeUp      int    `json:"wakeUp"`
	Sleep       int    `json:"sleep"`
	Meta        bool   `json:"meta"`
}
type Candle struct {
	Open  float64 `gorm:"open" json:"open"`
	High  float64 `gorm:"high" json:"high"`
	Low   float64 `gorm:"low" json:"low"`
	Close float64 `gorm:"close" json:"close"`
}

type Historico struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Value     string    `gorm:"column:value" json:"value"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type HistoricoAll struct {
	HistDate     time.Time `gorm:"column:hist_date" json:"hist_date"`
	TradingName  string    `gorm:"column:trading_name" json:"trading_name"`
	CurrentValue string    `gorm:"column:curr_value" json:"curr_value"`
}

type Bots struct {
	Symbol string `gorm:"symbol" json:"symbol"`
	User   string `gorm:"user" json:"user"`
}
type HookAlerts struct {
	User      string `gorm:"column:user" json:"user"`
	Symbol    string `gorm:"column:symbol" json:"symbol"`
	Side      string `gorm:"column:side" json:"side"`
	CreatedAt string `gorm:"column:created_at" json:"created_at"`
}

type ListBots struct {
	Symbol string `gorm:"symbol" json:"symbol"`
}
type KlineData struct {
	OpenTime                 int64   `json:"openTime"`
	Open                     float64 `json:"open"`
	High                     float64 `json:"high"`
	Low                      float64 `json:"low"`
	Close                    float64 `json:"close"`
	Volume                   float64 `json:"volume"`
	CloseTime                int64   `json:"closeTime"`
	QuoteAssetVolume         float64 `json:"quoteAssetVolume,string"`
	NumberOfTrades           int     `json:"numberOfTrades"`
	TakerBuyBaseAssetVolume  float64 `json:"takerBuyBaseAssetVolume,string"`
	TakerBuyQuoteAssetVolume float64 `json:"takerBuyQuoteAssetVolume,string"`
}

type VolumeData struct {
	Volume      float64 `json:"volume"`
	BuyVolume   float64 `json:"buyVolume"`
	SellVolume  float64 `json:"sellVolume"`
	RatioVolume float64 `json:"ratioVolume"`
}

func (Historico) TableName() string    { return "historico" }
func (HistoricoAll) TableName() string { return "hist_trading_values" }

func (ListBots) TableName() string { return "bots" }

func (HookAlerts) TableName() string { return "alerts" }
