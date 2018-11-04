package ups

import (
	"bufio"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"simonwaldherr.de/go/golibs/as"
	"simonwaldherr.de/go/golibs/file"
	"strings"
)

func CreateDeviceMap() *Devices {
	dev := make(map[string]*Device)
	return &Devices{
		Devs: dev,
	}
}

func (devices *Devices) Set(Mandant, Name, IP, Info string, DPI int, PeelOff bool) {
	devices.Devs[Name] = &Device{
		Mandt: Mandant,
		Name:  Name,
		IP:    IP,
		Info:  Info,
		DPI:   DPI,
		Peel:  PeelOff,
	}
}

func LoadCSVfromFile(filename string) (map[int][]string, map[string]int) {
	fp, _ := os.Open(filename)
	return loadCSV(bufio.NewReader(fp))
}

func LoadCSVfromString(csv string) (map[int][]string, map[string]int) {
	fp := strings.NewReader(csv)
	return loadCSV(fp)
}

func loadCSV(reader io.Reader) (map[int][]string, map[string]int) {
	var row int
	var head = map[int][]string{}
	var data = map[int][]string{}

	csvReader := csv.NewReader(reader)
	csvReader.Comma = ';'
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		if row == 0 {
			head[row] = record
		} else {
			data[row] = record
		}
		row++
	}
	return data, GetHead(head)
}

func GetHead(data map[int][]string) map[string]int {
	head := make(map[string]int, len(data[0]))
	for pos, name := range data[0] {
		head[name] = pos
	}
	return head
}

func ParseLabels(labeldir string) ([]string, map[string]string) {
	var labelsLocal []string
	var ltemplateLocal = make(map[string]string)
	labelsLocal, _ = file.ReadDir(filepath.Join(homedir, labeldir))
	fmt.Println("###### Labels ######")
	for _, name := range labelsLocal {
		if strings.Contains(name, ".200zpl") || strings.Contains(name, ".300zpl") || strings.Contains(name, ".zpl") {
			fmt.Printf("* %v\n", name)
			str, _ := file.Read(filepath.Join(homedir, labeldir, name))
			name = normalizeLabelName(name)
			ltemplateLocal[name] = str
		}
	}
	fmt.Println()
	return labelsLocal, ltemplateLocal
}

func LoadPrinter(filename string) *Devices {
	dev := CreateDeviceMap()
	csv, k := LoadCSVfromFile(filename)
	fmt.Println("###### Printer ######")

	for _, data := range csv {
		mndt := as.String(data[k["mndt"]])
		name := as.String(data[k["name"]])
		ip := as.String(data[k["ip"]])
		info := as.String(data[k["info"]])
		dpi := int(as.Int(data[k["dpi"]]))
		peel := as.Bool(data[k["peel"]])
		fmt.Printf("* %v\n", name)
		dev.Set(mndt, name, ip, info, dpi, peel)
	}

	fmt.Println()

	return dev
}

func (c *classAccessesMap) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	c.Map = map[string]string{}
	val := ""

	for {
		t, _ := d.Token()
		switch tt := t.(type) {

		case xml.StartElement:

		case xml.EndElement:
			if tt.Name == start.Name {
				return nil
			}
			c.Map[tt.Name.Local] = val
		default:
			val = strings.TrimSpace(fmt.Sprintf("%s", tt))
		}
	}
}

func ParseDocumentXML(xmlString string) Variables {
	m := Variables{}
	xml.Unmarshal([]byte(xmlString), &m)
	return m
}
