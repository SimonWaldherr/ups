package ups

type Variables struct {
	Head Head
	Data classAccessesMap `xml:"Data"`
}

type Head struct {
	Label   string
	Printer string
	Count   string
}

type classAccessesMap struct {
	Map map[string]string
}

type Device struct {
	Mandt string
	Name  string
	IP    string
	Info  string
	DPI   int
	Peel  bool
}

type Devices struct {
	Devs map[string]*Device
}

type LogMsg struct {
	Date    string
	Str     string
	Msgtype string
	Dst     string
	Ip      string
	Label   string
	Weight  string
}

type Connections struct {
	clients      map[chan LogMsg]bool
	addClient    chan chan LogMsg
	removeClient chan chan LogMsg
	messages     chan LogMsg
}
