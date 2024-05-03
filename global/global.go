package global

import (
	"candles/models"
	"time"
)

var (
	Bots              []models.Bots
	Key               string
	ForTime           time.Duration
	ValueCompradoCoin float64
	Started           string
	CmdRun            bool
	OrdemAtiva        bool
	Red               func(a ...interface{}) string
	Green             func(a ...interface{}) string

	CurrentCoin      string
	Value            float64
	TP               float64
	Stop             float64
	StopLossAll      float64
	StopMovel        bool
	Side             string
	NextValue        float64
	SliceCurrentCoin []string
	Meta             bool
	Alavancagem      float64
	AllCandlesBTC    []models.Candle
)
