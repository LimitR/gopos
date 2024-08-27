package gopos

import (
	"github.com/LimitR/gopos/pkg/text2img"
	"golang.org/x/sys/unix"
	"image"
	"image/color"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	commandRetractPaper = 0xA0 // Data: Number of steps to go back
	commandFeedPaper    = 0xA1 // Data: Number of steps to go forward
	commandDrawBitmap   = 0xA2 // Data: Line to draw. 0 bit -> don't draw pixel, 1 bit -> draw pixel
	commandDrawingMode  = 0xBE // Data: 1 for Text, 0 for Images
	commandSetEnergy    = 0xAF // Data: 1 - 0xFFFF
	commandSetQuality   = 0xA4 // Data: 1 - 5

	DefaultFont string = "./media/default.ttf"
)

type OptionPrint struct {
	FontSize int
	FontPath string
}

type OptionConnect struct {
	AddrPrinter   string
	DomainConnect int
	TypeConnect   int
	ProtoConnect  int

	PowerPrinter       byte // 0x01 - 0xFF
	QualityPrinter     byte // 0x01 - 0x05
	DrawingModePrinter byte // 0x01 for Text, 0x0 for Images
}

type Printer struct {
	debug    bool
	fd       int
	fontSize int
	fontPath string
}

func NewPrinter(option OptionConnect) (*Printer, error) {
	fd, err := unix.Socket(syscall.AF_BLUETOOTH, syscall.SOCK_STREAM, unix.BTPROTO_RFCOMM)

	if err != nil {
		return nil, err
	}

	addr := &unix.SockaddrRFCOMM{Addr: str2ba(option.AddrPrinter), Channel: 1}

	err = unix.Connect(fd, addr)

	if err != nil {
		return nil, err
	}

	p := &Printer{
		fd: fd,
	}
	p.sendCommand(commandSetEnergy, []byte{option.PowerPrinter})
	p.sendCommand(commandSetQuality, []byte{option.QualityPrinter})
	p.sendCommand(commandDrawingMode, []byte{option.DrawingModePrinter})
	return p, nil
}

func (p *Printer) PrintText(text string, option *OptionPrint) error {
	texts := strings.Split(text, "\n")

	for _, text := range texts {
		img, err := p.generateImageFromText(text, option)
		if err != nil {
			return err
		}

		bmps := p.generateBitMapFromImage(img)
		for _, bmp := range bmps {
			p.sendCommand(commandDrawBitmap, bmp)
			p.sendCommand(commandFeedPaper, []byte{0x00})
			time.Sleep(4 * time.Millisecond)
		}
	}
	p.sendCommand(commandFeedPaper, []byte{0x50})
	return nil
}

func (p *Printer) PrintImage(img *image.RGBA) error {
	bmps := p.generateBitMapFromImage(img)

	for _, bmp := range bmps {
		p.sendCommand(commandDrawBitmap, bmp)
		p.sendCommand(commandFeedPaper, []byte{0x00})
		time.Sleep(4 * time.Millisecond)
	}
	p.sendCommand(commandFeedPaper, []byte{0x50})
	return nil
}

func (p *Printer) generateImageFromText(text string, option *OptionPrint) (*image.RGBA, error) {
	d, err := text2img.NewDrawer(text2img.Params{
		FontPath:          option.FontPath,
		TextColor:         color.RGBA{R: 0, G: 0, B: 0, A: 255},
		BackgroundColor:   color.RGBA{R: 0, G: 0, B: 0, A: 0},
		FontSize:          float64(option.FontSize),
		Height:            option.FontSize,
		Width:             len(text) * 16,
		TextPosHorizontal: 0,
		TextPosVertical:   0,
	})
	if err != nil {
		return nil, err
	}

	img, err := d.Draw(text)
	if err != nil {
		return nil, err
	}

	return img, nil
}
func (p *Printer) generateBitMapFromImage(img *image.RGBA) [][]byte {
	result := make([][]byte, 0, img.Bounds().Dx())
	for y := 0; y < img.Bounds().Dy(); y++ {
		bmp := make([]byte, 0, img.Bounds().Dx())
		bit := 0
		for x := 0; x < img.Bounds().Dx(); x++ {
			if bit%8 == 0 {
				bmp = append(bmp, 0)
			}
			colorPint := img.RGBAAt(x, y)
			bmp[bit/8] >>= 1

			r, g, b, a := colorPint.RGBA()
			if r < 0x80 && g < 0x80 && b < 0x80 && a > 0x80 {
				bmp[bit/8] |= 0x80
			} else {
				bmp[bit/8] |= 0
			}
			bit++
		}
		result = append(result, bmp)
	}
	return result
}

func (p *Printer) sendCommand(command byte, data []byte) {
	commandMessage := p.formatMessage(command, data)
	_, err := unix.Write(p.fd, commandMessage)
	if err != nil {
		return
	}
}

var crc8_table = []byte{0x00, 0x07, 0x0e, 0x09, 0x1c, 0x1b, 0x12, 0x15, 0x38, 0x3f, 0x36, 0x31,
	0x24, 0x23, 0x2a, 0x2d, 0x70, 0x77, 0x7e, 0x79, 0x6c, 0x6b, 0x62, 0x65,
	0x48, 0x4f, 0x46, 0x41, 0x54, 0x53, 0x5a, 0x5d, 0xe0, 0xe7, 0xee, 0xe9,
	0xfc, 0xfb, 0xf2, 0xf5, 0xd8, 0xdf, 0xd6, 0xd1, 0xc4, 0xc3, 0xca, 0xcd,
	0x90, 0x97, 0x9e, 0x99, 0x8c, 0x8b, 0x82, 0x85, 0xa8, 0xaf, 0xa6, 0xa1,
	0xb4, 0xb3, 0xba, 0xbd, 0xc7, 0xc0, 0xc9, 0xce, 0xdb, 0xdc, 0xd5, 0xd2,
	0xff, 0xf8, 0xf1, 0xf6, 0xe3, 0xe4, 0xed, 0xea, 0xb7, 0xb0, 0xb9, 0xbe,
	0xab, 0xac, 0xa5, 0xa2, 0x8f, 0x88, 0x81, 0x86, 0x93, 0x94, 0x9d, 0x9a,
	0x27, 0x20, 0x29, 0x2e, 0x3b, 0x3c, 0x35, 0x32, 0x1f, 0x18, 0x11, 0x16,
	0x03, 0x04, 0x0d, 0x0a, 0x57, 0x50, 0x59, 0x5e, 0x4b, 0x4c, 0x45, 0x42,
	0x6f, 0x68, 0x61, 0x66, 0x73, 0x74, 0x7d, 0x7a, 0x89, 0x8e, 0x87, 0x80,
	0x95, 0x92, 0x9b, 0x9c, 0xb1, 0xb6, 0xbf, 0xb8, 0xad, 0xaa, 0xa3, 0xa4,
	0xf9, 0xfe, 0xf7, 0xf0, 0xe5, 0xe2, 0xeb, 0xec, 0xc1, 0xc6, 0xcf, 0xc8,
	0xdd, 0xda, 0xd3, 0xd4, 0x69, 0x6e, 0x67, 0x60, 0x75, 0x72, 0x7b, 0x7c,
	0x51, 0x56, 0x5f, 0x58, 0x4d, 0x4a, 0x43, 0x44, 0x19, 0x1e, 0x17, 0x10,
	0x05, 0x02, 0x0b, 0x0c, 0x21, 0x26, 0x2f, 0x28, 0x3d, 0x3a, 0x33, 0x34,
	0x4e, 0x49, 0x40, 0x47, 0x52, 0x55, 0x5c, 0x5b, 0x76, 0x71, 0x78, 0x7f,
	0x6a, 0x6d, 0x64, 0x63, 0x3e, 0x39, 0x30, 0x37, 0x22, 0x25, 0x2c, 0x2b,
	0x06, 0x01, 0x08, 0x0f, 0x1a, 0x1d, 0x14, 0x13, 0xae, 0xa9, 0xa0, 0xa7,
	0xb2, 0xb5, 0xbc, 0xbb, 0x96, 0x91, 0x98, 0x9f, 0x8a, 0x8d, 0x84, 0x83,
	0xde, 0xd9, 0xd0, 0xd7, 0xc2, 0xc5, 0xcc, 0xcb, 0xe6, 0xe1, 0xe8, 0xef,
	0xfa, 0xfd, 0xf4, 0xf3}

func (p *Printer) crc8(data []byte) byte {
	crc := byte(0)
	for _, b := range data {
		crc = crc8_table[(crc^b)&0xFF]
	}
	return crc & 0xFF
}

func (p *Printer) formatMessage(command byte, data []byte) []byte {
	data2 := []byte{0x51, 0x78, command, 0x00, byte(len(data)), 0x00}
	for _, b := range data {
		data2 = append(data2, b)
	}
	data2 = append(data2, p.crc8(data))
	data2 = append(data2, 0xFF)
	return data2
}

func str2ba(addr string) [6]byte {
	a := strings.Split(addr, ":")
	var b [6]byte
	for i, tmp := range a {
		u, _ := strconv.ParseUint(tmp, 16, 8)
		b[len(b)-1-i] = byte(u)
	}
	return b
}
