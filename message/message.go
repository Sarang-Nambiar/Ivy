package message

type Message struct {
	Type       string
	ID         int
	IP         string // IP address of the sender of the request
	PageID     int
	Permission string
	AvgReadPerNode float64
	AvgWritePerNode float64
}