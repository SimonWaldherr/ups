package ups

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"simonwaldherr.de/go/golibs/as"
	"simonwaldherr.de/go/golibs/csv"
	"simonwaldherr.de/go/golibs/file"
	"strings"
)

func CreateDeviceMap() *Devices {
	dev := make(map[string]*Device)
	return &Devices{
		Devs: dev,
	}
}

func (devices *Devices) Set(Mandant, Name, IP, Port, Info string, DPI int, PeelOff bool) {
	devices.Devs[Name] = &Device{
		Mandt: Mandant,
		Name:  Name,
		IP:    IP,
		Port:  Port,
		Info:  Info,
		DPI:   DPI,
		Peel:  PeelOff,
	}
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
	csvdata, k := csv.LoadCSVfromFile(filename)
	fmt.Println("###### Printer ######")

	for _, data := range csvdata {
		mndt := as.String(data[k["mndt"]])
		name := as.String(data[k["name"]])
		ip := as.String(data[k["ip"]])
		port := as.String(data[k["port"]])
		info := as.String(data[k["info"]])
		dpi := int(as.Int(data[k["dpi"]]))
		peel := as.Bool(data[k["peel"]])
		fmt.Printf("* %v\n", name)
		dev.Set(mndt, name, ip, port, info, dpi, peel)
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
