package ico

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadIconFile(t *testing.T) {
	var fs *os.File
	defer func() {
		if fs != nil {
			fs.Close()
		}
		if err := recover(); err != nil {
			log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
			log.Printf("LoadIconFile() = Error:%v\r\n", err)
			os.Exit(1)
		}
	}()
	path := "../testico/"
	file := "ICON16_1.ico"
	filePath := filepath.Join(path, file)
	fs, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	wi, err := LoadIconFile(fs)
	if err != nil {
		panic(err)
	}
	if err := wi.ExtractIconToFile("test", "../testico/"); err != nil {
		panic(err)
	}
}

// 测试-提取单个icon图标数据到ico文件
func TestWinIcon_IconToIcoFile(t *testing.T) {
	var fs *os.File
	defer func() {
		if fs != nil {
			fs.Close()
		}
		if err := recover(); err != nil {
			log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
			log.Printf("LoadIconFile() = Error:%v\r\n", err)
			os.Exit(1)
		}
	}()
	path := "../testico/"
	file := "ICON16_1.ico"
	filePath := filepath.Join(path, file)
	fs, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	wi, err := LoadIconFile(fs)
	if err != nil {
		panic(err)
	}
	if err := wi.IconToIcoFile("../testico/vk.ico", 1); err != nil {
		panic(err)
	}
}

func Test_getPerm(t *testing.T) {
	tests := []struct {
		name string
		want os.FileMode
	}{
		{"Test File Mode in Windows", 0},
		{"Test File Mode in Linux", 0666},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "windows" && i == 0 {
				got := getPerm()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("getPerm() = %v, want %v", got, tt.want)
				} else {
					t.Logf("getPerm() == %v, want %v", got, tt.want)
				}
			}
			if runtime.GOOS == "linux" && i == 1 {
				got := getPerm()
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("getPerm() = %v, want %v", got, tt.want)
				} else {
					t.Logf("getPerm() == %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_loadImageData(t *testing.T) {
	var fs *os.File
	defer func() {
		if fs != nil {
			fs.Close()
		}
		if err := recover(); err != nil {
			log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
			log.Printf("LoadIconFile() = Error:%v\r\n", err)
			os.Exit(1)
		}
	}()
	path := "../testico/"
	file := "vkico64x64@32bit.bmp"
	filePath := filepath.Join(path, file)
	fs, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	imgdata, tp, e := loadImageData(fs)
	if e != io.EOF && e != nil {
		t.Logf("type error:%v\r\n", e)
	}
	t.Logf("image type:%v\r\n", tp)
	t.Logf("data size:%v\r\n", len(imgdata))
}

func TestCreateWinIcon(t *testing.T) {
	type args struct {
		filePath []string
	}
	tests := []struct {
		name    string
		args    args
		want    *WinIcon
		wantErr bool
	}{
		{
			"Test Create Win Icon",
			args{[]string{
				"D:\\go_project\\src\\WinIconTools\\testico\\vkico16x16@32bit.bmp",
				"D:\\go_project\\src\\WinIconTools\\testico\\vkico20x20@32bit.bmp",
				"D:\\go_project\\src\\WinIconTools\\testico\\vkico24x24@32bit.bmp",
				"D:\\go_project\\src\\WinIconTools\\testico\\vkico32x32@32bit.bmp",
				"D:\\go_project\\src\\WinIconTools\\testico\\vkico40x40@32bit.bmp",
				"D:\\go_project\\src\\WinIconTools\\testico\\vkico64x64@32bit.bmp",
				"D:\\go_project\\src\\WinIconTools\\testico\\vkico256x256@32bit.bmp",
			}},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateWinIcon(tt.args.filePath)
			if err != nil {
				t.Errorf("CreateWinIcon() = %v", err)
			}
			for i, _ := range got.icos {
				t.Logf("width:%v, heigth:%v, bits:%v, offset:%v, length:%v\r\n",
					got.icos[i].getIconWidth(),
					got.icos[i].getIconHeight(),
					got.icos[i].getIconBitsPerPixel(),
					got.icos[i].getIconOffset(),
					got.icos[i].getIconLength(),
				)
			}
			Time := func() string {
				tb, _ := time.Now().MarshalText()
				return fmt.Sprintf("%s%s",
					strings.Join(strings.Split(string(tb[0:10]), "-"), ""),
					strings.Join(strings.Split(string(tb[11:19]), ":"), ""),
				)
			}()
			got.WriteIcoFile("../", Time+".ico")
		})
	}
}

func TestCRC(t *testing.T) {
	b, e := ioutil.ReadFile("../testico/vkico256x256@32bit.png")
	if e != nil {
		fmt.Println(e)
		return
	}
	i := bytes.Index(b, []byte("IDAT"))
	l := binary.BigEndian.Uint32(b[i-4 : i])
	crc := crc32.ChecksumIEEE(b[i : int(l)+i+4])
	fmt.Println("Offset:", i)
	fmt.Println("Length", l)
	cb := make([]byte, 4)
	binary.BigEndian.PutUint32(cb, crc)
	fmt.Printf("[%0x %0x %0x %0x]\r\n", cb[0], cb[1], cb[2], cb[3])
	o := int(l) + i + 4
	fmt.Printf("[%0x %0x %0x %0x]\r\n", b[o], b[o+1], b[o+2], b[o+3])
}
