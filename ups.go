package ups

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	_ "expvar"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"simonwaldherr.de/go/golibs/as"
	"simonwaldherr.de/go/golibs/cache"
	"simonwaldherr.de/go/golibs/file"
	"simonwaldherr.de/go/golibs/regex"
	"simonwaldherr.de/go/zplgfa"
	"strings"
	"time"
)

var cacheXML *cache.Cache
var cacheMAT *cache.Cache
var cacheTXT *cache.Cache

var portWaage string = ":56429"

var Hub = &Connections{
	clients:      make(map[chan LogMsg]bool),
	addClient:    make(chan (chan LogMsg)),
	removeClient: make(chan (chan LogMsg)),
	messages:     make(chan LogMsg),
}

func (hub *Connections) Init() {
	go func() {
		for {
			select {
			case s := <-hub.addClient:
				hub.clients[s] = true
				Info.Println("Added new client")

			case s := <-hub.removeClient:
				delete(hub.clients, s)
				Info.Println("Removed client")

			case msg := <-hub.messages:
				for s := range hub.clients {
					s <- msg
				}
			}
		}
	}()
}

func normalizeLabelName(name string) string {
	name = strings.ToLower(name)
	name, _ = regex.ReplaceAllString(name, "[ _-]", "")
	name = strings.TrimSpace(name)
	return name
}

var (
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func LogInit(
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

var infoserver string
var homedir string
var goos string
var testdrucker string

const timeout int64 = 32
const retrytime = 510 * time.Millisecond

var sonderzeichen = map[string]string{
	"Ö": "\\99",
	"ö": "\\94",
	"Ü": "\\9A",
	"ü": "\\81",
	"Ä": "\\8E",
	"ä": "\\84",
	"ß": "\\E1",
	"Ø": "\\9D",
	"µ": "\\E6",
	"~": "\\7E",
}

type msgch struct {
	msg    string
	ip     string
	port   string
	intime int64
}

var msgchan = make(chan msgch, 64)
var Labels []string
var Ltemplate = make(map[string]string)
var Printer *Devices

func toUtf8(iso8859 []byte) string {
	buf := make([]rune, len(iso8859))
	for i, b := range iso8859 {
		buf[i] = rune(b)
	}
	return string(buf)
}

func sendLabelToZebra(ip, port, printertype, zpl string, retry int) bool {
	var servAddr string
	servAddr = ip + ":" + port

	Info.Println(servAddr)

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	conn, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err == nil {
		defer conn.Close()
		payloadBytes := []byte(fmt.Sprintf("%s\r\n\r\n", zpl))
		if _, err = conn.Write(payloadBytes); err != nil {
			Info.Println(err)

			if retry > 0 {
				Warning.Printf("pos: 1 ip: %v retry: %v err: %v", ip, retry, err)

				time.Sleep(retrytime)
				return sendLabelToZebra(ip, port, printertype, zpl, retry-1)
			}
		}
		return true
	}
	Warning.Println(err)

	if retry > 0 {
		Warning.Printf("pos: 2 ip: %v retry: %v err: %v", ip, retry, err)

		time.Sleep(retrytime)
		return sendLabelToZebra(ip, port, printertype, zpl, retry-1)
	}
	return false
}

func sendDataToZebra(ip, port, printertype, str string) bool {
	var servAddr string

	servAddr = ip + ":" + port

	Info.Println(servAddr)

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	conn, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err == nil {
		defer conn.Close()

		payloadBytes := []byte(fmt.Sprintf("%s\r\n\r\n", str))
		if _, err = conn.Write(payloadBytes); err != nil {
			Info.Println(err)
		}
		return true
	}
	Warning.Println(err)
	return false
}

func sendFeedCmdToZebra(ip, port, printertype string) bool {
	return sendDataToZebra(ip, port, printertype, "^xa^aa^fd ^fs^xz")
}

func sendCalibCmdToZebra(ip, port, printertype string) bool {
	return sendDataToZebra(ip, port, printertype, "~jc^xa^jus^xz")
}

func sendCmdToZebra(data, printername string) bool {
	if _, ok := Printer.Devs[printername]; ok {
		printertype := PrinterType(printername)
		return sendDataToZebra(Printer.Devs[printername].IP, Printer.Devs[printername].Port, printertype, data)
	}
	return false
}

func getInfoFromZebra(ip, port string, retry int) string {
	zpl := "~HS"
	servAddr := ip + ":" + port
	Info.Println(servAddr)

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	conn, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err == nil {
		defer conn.Close()

		payloadBytes := []byte(fmt.Sprintf("%s\r\n\r\n", zpl))
		if _, err = conn.Write(payloadBytes); err != nil {
			Warning.Println(err)

			if retry > 0 {
				Warning.Printf("pos: 1 ip: %v retry: %v err: %v", ip, retry, err)

				time.Sleep(retrytime)
				return getInfoFromZebra(ip, port, retry-1)
			}
		}
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return ""
		}
		return message
	}
	Warning.Println(err)

	if retry > 0 {
		Warning.Printf("pos: 2 ip: %v retry: %v err: %v", ip, retry, err)

		time.Sleep(retrytime)
		return getInfoFromZebra(ip, port, retry-1)
	}
	return ""
}

func toISODate(str string) string {
	if str == "00000000" {
		return ""
	}
	if len(str) == 8 {
		return str[0:4] + "-" + str[4:6] + "-" + str[6:8]
	}
	return str
}

func handleTCPConnection(c net.Conn) {
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
	buf := make([]byte, 4096)
	for {
		n, err := c.Read(buf)
		if (err != nil) || (n == 0) {
			c.Close()
			break
		}
		Info.Println("handleTCPConnection")

		if n > 20 {
			msgchan <- msgch{
				msg:    fmt.Sprintf("%v<LOADBALANCER>%v", c.RemoteAddr(), string(buf[0:n])),
				ip:     strings.Split(as.String(c.RemoteAddr()), ":")[0],
				port:   strings.Split(as.String(c.RemoteAddr()), ":")[1],
				intime: time.Now().Unix(),
			}
		}
	}
	time.Sleep(150 * time.Millisecond)
	c.Close()
	Info.Printf("Connection from %v closed.\n", c.RemoteAddr())
}

func handleHTTPConnection(rw http.ResponseWriter, req *http.Request) {
	postdata, err := ioutil.ReadAll(req.Body)
	if err == nil {
		str := strings.TrimSpace(string(postdata))
		if str != "" {
			if strings.Contains(str, "<Printer>") {
				msgchan <- msgch{
					msg:    fmt.Sprintf("%v<LOADBALANCER>%v", req.RemoteAddr, str),
					ip:     strings.Split(as.String(req.RemoteAddr), ":")[0],
					port:   strings.Split(as.String(req.RemoteAddr), ":")[1],
					intime: time.Now().Unix(),
				}
				return
			}
		}
	}
	f, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming not supported!", http.StatusInternalServerError)
		return
	}

	if req.URL.Path == "/" {
		rw.Header().Set("Server", "NicerWatch 0.998")
		rw.Header().Set("Content-Type", "text/html; charset=UTF-8")
		rw.WriteHeader(200)
		str, _ := file.Read(filepath.Join(homedir, "index.html"))
		fmt.Fprintf(rw, str)

		f.Flush()
		return
	} else if req.URL.Path == "/reloadPrinter" {
		Printer = LoadPrinter("drucker.txt")
		rw.Header().Set("Server", "NicerWatch 0.998")
		rw.Header().Set("Content-Type", "text/html; charset=UTF-8")
		rw.WriteHeader(200)
		fmt.Fprintf(rw, fmt.Sprintf("%#v", Printer))

		f.Flush()
		return
	} else if req.URL.Path == "/reloadLabels" {
		Labels, Ltemplate = ParseLabels("labels")
		rw.Header().Set("Server", "NicerWatch 0.998")
		rw.Header().Set("Content-Type", "text/html; charset=UTF-8")
		rw.WriteHeader(200)
		fmt.Fprintf(rw, fmt.Sprintf("%#v", Labels))

		f.Flush()
		return
	} else if len(req.URL.Path) > 6 && req.URL.Path[0:6] == "/send/" {
		path := strings.Split(strings.Replace(req.URL.Path, "/send/", "", 1), "/")
		if path[0] == "calibrate" {
			sendCmdToZebra("~jc^xa^jus^xz", path[1])
		} else if path[0] == "feed" {
			sendCmdToZebra("^xa^aa^fd ^fs^xz", path[1])
		}
		rw.Header().Set("Server", "NicerWatch 0.998")
		rw.Header().Set("Content-Type", "text/html; charset=UTF-8")
		rw.WriteHeader(200)

		f.Flush()
		return
	}

	messageChannel := make(chan LogMsg)
	Hub.addClient <- messageChannel
	notify := rw.(http.CloseNotifier).CloseNotify()

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")

	tickerDuration := time.Second * 60
	ticker := time.NewTimer(tickerDuration)
	for i := 0; i < 1440; {
		fmt.Println("foo", i)
		ticker.Reset(tickerDuration)
		select {
		case msg := <-messageChannel:
			ticker.Stop()
			jsonData, _ := json.Marshal(msg)
			str := string(jsonData)
			str = fmt.Sprintf("data: {\"str\": %s,\"time\": \"%v\",\"x\": \"\"}\n\n", str, time.Now())
			if req.URL.Path == "/events/sse" {
				fmt.Fprint(rw, str)
			} else if req.URL.Path == "/events/lp" {
				fmt.Fprint(rw, str)
			}
			f.Flush()
		case <-ticker.C:
			if req.URL.Path == "/events/sse" {
				fmt.Fprintf(rw, "data: {\"str\": \"No Data\"}\n\n")
			} else if req.URL.Path == "/events/lp" {
				fmt.Fprintf(rw, "{\"str\": \"No Data\"}")
			}
			f.Flush()
			i++
		case <-notify:
			ticker.Stop()
			f.Flush()
			i = 1440
			Hub.removeClient <- messageChannel
		}
	}
}

func isZPLprintable(label string) bool {
	if _, ok := Ltemplate[label]; ok {
		return true
	}
	return false
}

func cdatafy(xml string, ele ...string) string {
	for _, element := range ele {
		xml = strings.Replace(xml, "</"+element+">", "]]></"+element+">", -1)
		xml = strings.Replace(xml, "<"+element+">", "<"+element+"><![CDATA[", -1)
	}
	return xml
}

func base64decode(s string) string {
	data, err := base64.StdEncoding.DecodeString(s)
	if err == nil {
		return string(data)
	}
	return ""
}

func isPicture(s string) bool {
	fileExtension := strings.ToLower(filepath.Ext(s))
	if fileExtension == ".jpg" || fileExtension == ".jpeg" || fileExtension == ".png" {
		return true
	}
	return false
}

func convertPictureToZPL(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Warning: could not open the file: %s\n", err)
		return "", err
	}

	defer file.Close()

	// load image head information
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		log.Printf("Warning: image not compatible, format: %s, config: %v, error: %s\n", format, config, err)
		return "", err
	}

	// reset file pointer to the beginning of the file
	file.Seek(0, 0)

	// load and decode image
	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Warning: could not decode the file, %s\n", err)
		return "", err
	}

	// flatten image
	flat := zplgfa.FlattenImage(img)

	// convert image to zpl compatible type
	gfimg := zplgfa.ConvertToZPL(flat, zplgfa.CompressedASCII)
	return gfimg, nil
}

func PrintMessages() {
	for {
		func() {
			defer func() {
				recover()
			}()

			msgStrctIn := <-msgchan
			msgStrctIn.msg = strings.TrimSpace(msgStrctIn.msg)

			go func(msgstrct msgch) {
				arr := strings.Split(msgstrct.msg, "<?xml ver")
				for i, val := range arr {
					arr[i] = strings.Trim(val, " ")
					if len(arr[i]) != 0 {
						msgstrct.msg = "<?xml ver" + arr[i]
						msgstrct.msg = cdatafy(msgstrct.msg, "BSTKD", "VENDOR_TEXT", "SHIP_FROM_TEXT", "MTTEXT")
						q := ParseDocumentXML(msgstrct.msg)

						if len(q.Head.Printer) != 0 {
							if q.Head.Count == "0" || q.Head.Count == "" {
								q.Head.Count = "1"
							}

							fmt.Printf("printing %s labels\n", q.Head.Count)

							if len(q.Head.Printer) != 0 {
								var base64decoded string
								printertype := PrinterType(q.Head.Printer)
								zplPrintable := isZPLprintable(q.Head.Label)
								labelIsPicture := isPicture(q.Head.Label)
								printerUnknown := true
								peelOff := false

								if !zplPrintable {
									base64decoded = base64decode(q.Head.Label)
								}

								if _, ok := Printer.Devs[q.Head.Printer]; ok {
									printerUnknown = false
									peelOff = Printer.Devs[q.Head.Printer].Peel
								} else {
									Warning.Printf("couldn't found printer %#v in printer table.\n", q.Head.Printer)
								}

								fmt.Printf("printerUnknown: %v, isZPLprintable: %v\n", printerUnknown, zplPrintable)

								if !printerUnknown && (zplPrintable || labelIsPicture || base64decoded != "") {
									var tmpl string

									if zplPrintable {
										tmpl = Ltemplate[q.Head.Label]
									} else if base64decoded != "" {
										tmpl = base64decoded
									}

									if tmpl != "" {
										for fieldName, fieldValue := range q.Data.Map {

											fieldValue = strings.Replace(fieldValue, "\\", "\\1F", -1)

											for key, encoded := range sonderzeichen {
												fieldValue = strings.Replace(fieldValue, key, encoded, -1)
											}

											tmpl = strings.Replace(tmpl, "$"+fieldName+"$", fieldValue, -1)
										}

										tmpl = strings.Replace(tmpl, "$DATE$", time.Now().Format("2006-01-02"), -1)
										tmpl = strings.Replace(tmpl, "$TIME$", time.Now().Format("15:04:05"), -1)
										tmpl = strings.Replace(tmpl, "$PRINTER$", q.Head.Printer, -1)
										tmpl = strings.Replace(tmpl, "^MTT", "^MTD", -1)

										if peelOff {
											tmpl = strings.Replace(tmpl, "^MMT", "^MMK", -1)
										} else {
											tmpl = strings.Replace(tmpl, "^MMP", "^MMT", -1)
											tmpl = strings.Replace(tmpl, "^MMK", "^MMT", -1)
										}
										tmpl, _ = regex.ReplaceAllString(tmpl, "\\^PR\\d+,\\d+", "^PR12,12")
									} else if labelIsPicture {
										tmpl, _ = convertPictureToZPL(q.Head.Label)
									}

									if tmpl != "" {
										if q.Head.Count == "1" {
											go sendLabelToZebra(Printer.Devs[q.Head.Printer].IP, Printer.Devs[q.Head.Printer].Port, printertype, tmpl, 3)
										} else {
											var ic int64
											ici := as.Int(q.Head.Count)
											for ic = 0; ic < ici; ic++ {
												go sendLabelToZebra(Printer.Devs[q.Head.Printer].IP, Printer.Devs[q.Head.Printer].Port, printertype, tmpl, 3)
											}
										}
									}
								}

								if printerUnknown {
									msga := "Etikett für " + q.Head.Printer + " (IP UNKNOWN " + q.Head.Printer + ") im Format " + q.Head.Label
									Hub.messages <- LogMsg{Date: as.String(time.Now()), Str: msga, Msgtype: "label", Dst: q.Head.Printer, Ip: "unknown", Label: q.Head.Label, Weight: ""}
								} else {
									msga := "Etikett für " + q.Head.Printer + " (" + Printer.Devs[q.Head.Printer].IP + ") im Format " + q.Head.Label
									Hub.messages <- LogMsg{Date: as.String(time.Now()), Str: msga, Msgtype: "label", Dst: q.Head.Printer, Ip: Printer.Devs[q.Head.Printer].IP, Label: q.Head.Label, Weight: ""}
								}
							}
						}
					}
				}
			}(msgStrctIn)
		}()
	}
}

func InitTelnet() {
	flag.Parse()
	port := ":" + flag.Arg(0)
	if port == ":" {
		port = ":30000"
	}

	ln, err := net.Listen("tcp", port)
	if err != nil {
		Warning.Println("Listen TCP: ", err)
		os.Exit(1)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			Warning.Println("handleTCPConnection: ", err)
			continue
		}
		go handleTCPConnection(conn)
	}
}

func InitHTTP() {
	flag.Parse()
	port := ":" + flag.Arg(1)

	if port == ":" {
		port = ":56425"
	}

	http.HandleFunc("/", handleHTTPConnection)
	log.Fatal(http.ListenAndServe(port, nil))

}

func PrinterType(str string) string {
	printerType, _ := regex.ReplaceAllString(str, "[0-9 ]", "")
	return printerType
}
