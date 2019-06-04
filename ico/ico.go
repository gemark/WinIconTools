/*
   _____       __   __             _  __
  ╱ ____|     |  ╲/   |           | |/ /
 | |  __  ___ |  ╲ /  | __  _ _ __| ' /
 | | |_ |/ _ ╲| |╲ /| |/ _`  | '__|  <
 | |__| |  __/| |   | (  _|  | |  | . ╲
  ╲_____|╲___ |_|   |_|╲__,_ |_|  |_|╲_╲
 可爱飞行猪❤: golang83@outlook.com  💯💯💯
 Author Name: GeMarK.VK.Chow奥迪哥  🚗🔞🈲
 Creaet Time: 2019/05/25 - 07:51:34
 ProgramFile: ico.go
 Description:
			  Windows系统的ico文件工具包
*/

package ico

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// 定义常量
// Constant definition
const (
	typeUKN           = iota // unknow type
	typeBMP                  // bmp file
	typePNG                  // png file
	fileHeaderSize    = 6    // 文件头的大小
	pngFileHeaderSize = 8    // png文件头大小
	headerSize        = 16   // icon图标的头结构大小
	bitmapHeaderSize  = 14   // 位图文件头
	dibHeaderSize     = 40   // dib结构头
)

// 定义变量
// Variable definitions
var (
	// 错误信息
	ErrIcoInvalid  = errors.New("ico: Invalid icon file")                   // 无效的ico文件
	ErrIcoReaders  = errors.New("ico: Reader type is not os.File pointer")  // LoadIconFile的io.Reader参数不是文件指针
	ErrIcoFileType = errors.New("ico: Reader is directory, not file")       // io.Reader的文件指针是目录，不是文件
	ErrIconsIndex  = errors.New("ico: Slice out of bounds")                 // 读取ico文件时，可能出现的切片越界错误
	PNGHEADER      = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG 文件头
	DIBHEADER      = []byte{0x28, 0, 0, 0}                                  // DIB 头
	BMPHEADERID    = []byte{0x42, 0x4d}
)

// 类型定义 type definition
// 定义icon图标数据的类型
// Define the type of icon data
type (
	ICONTYPE      int
	WinIconData   []byte
	WinIconStruct []winIconStruct
)

// 定义 Windows 系统的 Ico 文件结构
// Defining the Ico file structure of Windows system
type WinIcon struct {
	fileHeader *winIconFileHeader // 文件头
	icos       WinIconStruct      // icon 头结构
}

// ico文件头结构
// 参考维基百科：
// https://en.wikipedia.org/wiki/ICO_(file_format)
type winIconFileHeader struct {
	ReservedA  uint16 // 保留字段，始终为 '0x0000'
	FileType   uint16 // 图像类型：'0x0100' 为 ico，'0x0200' 为 cur
	ImageCount uint16 // 图像数量：至少为 '0x0100' 即 1个图标
}

// icon图标头结构
// 参考维基百科：
// https://en.wikipedia.org/wiki/ICO_(file_format)
type winIconStruct struct {
	Width         uint8       // 图像宽度
	Height        uint8       // 图像高度
	Palette       uint8       // 调色板颜色数，不使用调色版为 '0x00'
	ReservedB     uint8       // 保留字段，始终为 '0x00'
	ColorPlanes   uint16      // 在ico中，指定颜色平面，'0x0000' 或则 '0x0100'
	BitsPerPixel  uint16      // 在ico中，指定每像素的位数，如：'0x2000' 32bit
	ImageDataSize uint32      // 图像数据的大小，单位字节
	ImageOffset   uint32      // 图像数据的偏移量
	data          WinIconData // 该图标的图像数据
}

// bitmap 的 DIB 头结构
// DIB header (bitmap information header)
// 参考维基百科：
// https://en.wikipedia.org/wiki/BMP_file_format
type dibHeader struct {
	dibSize        uint32 // 0x28 00 00 00 -> 40bytes (DIB Header Size)
	bitmapWidth    uint32 // Width of the bitmap in pixels -> left to right order
	bitmapHeight   uint32 // Heigth of the bitmap in pixels -> bottom to top order
	colorPlanes    uint16 // Number of color planes being used
	BitsPerPixel   uint16 // Number of bits per pixel
	markerBI_RGB   uint32 // BI_BITFIELDS, no pixel array compression used
	originalSize   uint32 // Size of the raw bitmap data (including padding)
	printResH      uint32 // 0x13 0b 00 00 or 0x12 0b 00 00-> 72 DPI x 39.3701
	printResV      uint32 // 0x13 0b 00 00 or 0x12 0b 00 00-> 72 DPI x 39.3701
	Palette        uint32 // 0 or 255 -> 0x00 00 00 00 or 0x00 01 00 00
	importantColor uint32 // 0 or 255 -> 0x00 00 00 00 or 0x00 01 00 00
}

// bitmap 的 BITMAP 头结构
// BITMAPFILEHEADER(14bytes)
// 参考维基百科：
// https://en.wikipedia.org/wiki/BMP_file_format
type bitmapHeader struct {
	bitmapID         uint16 // 0x42 0x4d "BM"
	fileSize         uint32 // BMP头与DIB的大小
	unusedA          uint16 // 0x00 00
	unusedB          uint16 // 0x00 00
	bitmapDataOffset uint32 // Bitmap Data 偏移量
}

// createBitmapHeader 创建位图文件头结构
// Create a bitmap file header structure
func createBitmapHeader(datasize int) *bitmapHeader {
	return &bitmapHeader{
		bitmapID:         binary.LittleEndian.Uint16([]byte{0x42, 0x4d}),
		fileSize:         uint32(datasize + bitmapHeaderSize),
		unusedA:          0,
		unusedB:          0,
		bitmapDataOffset: uint32(bitmapHeaderSize + dibHeaderSize),
	}
}

// GetIconType 获取icon的数据类型
// Get the image type of the icon
func GetIconType(d []byte) ICONTYPE {
	if checkDIBHeader(d) {
		return typeBMP
	}
	if checkPNGHeader(d) {
		return typePNG
	} else {
		return typeUKN
	}
}

// checkPNGHeader 检测是否是png ico数据
// Check if it is png ico data
func checkPNGHeader(d []byte) bool {
	if len(d) < len(PNGHEADER) {
		return false
	}
	if bytes.Compare(d[0:8], PNGHEADER) != 0 {
		return false
	}
	return true
}

// headerToBytes 将bitmapHeader位图头结构转换为字节切片
// Convert BITMAPFILEHEADER bitmap header structure to byte slice
func (bmh *bitmapHeader) headerToBytes() []byte {
	d := make([]byte, bitmapHeaderSize)
	binary.LittleEndian.PutUint16(d[0:2], bmh.bitmapID)
	binary.LittleEndian.PutUint32(d[2:6], bmh.fileSize)
	binary.LittleEndian.PutUint16(d[6:8], bmh.unusedA)
	binary.LittleEndian.PutUint16(d[8:10], bmh.unusedB)
	binary.LittleEndian.PutUint32(d[10:14], bmh.bitmapDataOffset)
	return d
}

// JoinHeader 将bitmapfileheader链接到含有dib头的位图数据前
// Link BITMAPFILEHEADER to the front of the bitmap
// data containing the dib header.
func (bmh *bitmapHeader) JoinHeader(d []byte) []byte {
	h := bmh.headerToBytes()
	j := [][]byte{h, d}
	return bytes.Join(j, nil)
}

// checkDIBHeader 检测是否是bmp ico数据
// Check if it is bmp ico data
func checkDIBHeader(d []byte) bool {
	if len(d) < dibHeaderSize {
		return false
	}
	if bytes.Compare(d[0:4], DIBHEADER) != 0 {
		return false
	}
	return true
}

// getPerm 更具操作系统定义写入文件时的FileMode
func getPerm() os.FileMode {
	if runtime.GOOS == "windows" {
		return 0
	} else {
		return 0666
	}
}

// 将ico文件的数据载入到内存
// Load data from ico file into memory
// rd io.Reader must *os.File object pointer.
// Successfully return WinIcon pointer.
// Failed to return error object
func LoadIconFile(rd io.Reader) (icon *WinIcon, err error) {
	// 类型断言
	v, t := rd.(*os.File)
	if !t {
		return nil, ErrIcoReaders
	}
	// 声明与定义变量
	var (
		fileSize int64
		ico      *WinIcon
	)

	// 获取文件信息及判断是否是文件，而不是目录
	fi, err := v.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, ErrIcoFileType
	}
	fileSize = fi.Size()

	// 创建缓冲IO的Reader对象窥视6个字节的文件头
	reader := bufio.NewReader(rd)
	p, err := reader.Peek(fileHeaderSize)
	if err != nil {
		return nil, err
	}

	// 检测文件头及获取头结构
	icoHeader, err := getIconFileHeader(p)
	if err != nil {
		return nil, err
	}

	// 获取ico文件的所有数据
	data, err := getFileAll(reader, fileSize)
	if err != nil {
		return nil, err
	}

	// 创建一个 winIconStruct 数组切片
	icos := make(WinIconStruct, int(icoHeader.ImageCount))
	// 根据文件头中表示的icon图标文件的数量进行循环
	structOffset := fileHeaderSize
	for i := 0; i < int(icoHeader.ImageCount); i++ {
		wis := getIconStruct(data, structOffset, headerSize)
		icodata := wis.getImageData(data, wis.getIconOffset(), wis.getIconLength())
		structOffset += headerSize
		icos[i] = *wis
		icos[i].data = icodata
	}

	// 创建 WinIcon 对象
	ico = &WinIcon{
		fileHeader: icoHeader,
		icos:       icos,
	}
	return ico, nil
}

// getFileAll 获取ico文件所有数据(不包括文件头的6个字节)
// rd *bufio.Reader: 对象
// size int64: 文件大小（我们需要读取的总数量）
// fb []byte: 文件的所有数据，如果成功读取的话
// err error: 如果读取出现错误，返回错误
// Get all data of ico or other image file(exclude first 6bytes of file)
// first 6 bytes its ico file header.
func getFileAll(rd *bufio.Reader, size int64) (fb []byte, err error) {
	data := make([]byte, size)
	for i := int64(0); i < size; i++ {
		b, err := rd.ReadByte()
		if err != nil {
			return nil, err
		}
		data[i] = b
	}
	return data, nil
}

// getIconFileHeader 获取文件头结构
// b []byte: 读取的数据来自这个字节切片
// wih *winIconFileHeader: 如果获取成功返回 winIconFileHeader对象指针
// err error: 如果读取发生错误，则返回错误信息
// Get structure header of ico file.
func getIconFileHeader(b []byte) (wih *winIconFileHeader, err error) {
	if len(b) != fileHeaderSize {
		return nil, ErrIcoInvalid
	}
	reserved := binary.LittleEndian.Uint16(b[0:2])
	filetype := binary.LittleEndian.Uint16(b[2:4])
	imagecount := binary.LittleEndian.Uint16(b[4:6])
	if reserved != 0 || filetype != 1 || imagecount == 0 {
		return nil, ErrIcoInvalid
	}
	header := &winIconFileHeader{
		ReservedA:  reserved,
		FileType:   filetype,
		ImageCount: imagecount,
	}
	return header, nil
}

// getIconStruct 根据 offset, length 来获取icon图标结构
// b []byte: 文件数据的字节切片
// offset int: 偏移量
// length int: 数据长度
// Get icon image structure according to offset, length arguments
func getIconStruct(b []byte, o, l int) (wis *winIconStruct) {
	var s []byte
	s = make([]byte, headerSize)
	j := 0
	for i := o; i < o+l; i++ {
		s[j] = b[i]
		j++
	}
	is := &winIconStruct{
		Width:         s[0],
		Height:        s[1],
		Palette:       s[2],
		ReservedB:     s[3],
		ColorPlanes:   binary.LittleEndian.Uint16(s[4:6]),
		BitsPerPixel:  binary.LittleEndian.Uint16(s[6:8]),
		ImageDataSize: binary.LittleEndian.Uint32(s[8:12]),
		ImageOffset:   binary.LittleEndian.Uint32(s[12:]),
	}
	return is
}

// getImageData 根据 offset, length 参数获取图标图像数据
// index int: 索引
// return []byte: 返回获取的数据字节切片
// Get icon image data according to index arguments
func (wi *WinIcon) getImageData(index int) []byte {
	return wi.icos[index].data
}

// getImageData 根据 offset, length 参数获取图标图像数据
// data []byte: 图像数据的字节切片
// offset int: 图像数据的偏移量
// length int: 图像数据的长度
// return []byte: 返回获取的数据字节切片
// Get icon image data according to offset, length arguments
func (wis winIconStruct) getImageData(b []byte, o, s int) []byte {
	var d = make([]byte, s)
	for i, j := o, 0; i < o+s; i++ {
		d[j] = b[i]
		j++
	}
	return d
}

// ExtractIconToFile 提取 ico 数据到文件
// filePrefix string: 为前缀，如果传如空字符串，则没有前缀，使用数字和分辨率作为文件名
// filePath string: 提取的数据写入的路径，空字符串则将文件保存到当前目录
// 舍弃：--count int: 提取文件的数量，0: 为所有，> 0 则根据已保存的map对象来提取
// 对应数量内容，指定数量超出实际数量则全部提取--
// 该函数不检测路径的有效性，使用者自己把控，如果路径有问题，会返回error对象
// ExtractIconToFile function its extract ico data to file(.ico)
// The prefix, if passed as an empty string, there is no prefix,
// using the number and resolution as the file name.
// This function does not detect the validity of the path.
// the user controls it. if there is a problem with the
// path, it will return an error object.
func (wi *WinIcon) ExtractIconToFile(filePrefix, filePath string) error {
	var ext string
	for i, v := range wi.icos {
		w := v.getIconWidth()
		h := v.getIconHeight()
		b := v.getIconBitsPerPixel()
		d, _ := wi.GetImageData(i)
		if GetIconType(d) == typeBMP {
			ext = "bmp"
		} else {
			ext = "png"
			if w == 0 && h == 0 {
				w = 256
				h = 256
			}
		}
		fn := v.generateFileNameFormat(filePrefix, ext, w, h, b)
		if err := wi.IconToFile(filePath, fn, i); err != nil {
			return err
		}
	}
	return nil
}

// GetImageData 获取ico图标的图像数据
// index int: 下标索引，0序
// 如果越界或读取数据错误，返回 error 对象
// Returns an error object if it is out of
// bounds or reads data incorrectly.
func (wi *WinIcon) GetImageData(index int) (d []byte, err error) {
	if index >= wi.getIconsHeaderCount() || index < 0 {
		return nil, ErrIconsIndex
	}
	return wi.getImageData(index), nil
}

// IconToFile 将图标写入文件
// path string: 文件写入的路径
// name string: 文件名
// error 如果写入发生错误，则返回错误信息
// IconToFile 并不会检测路径是否有效
// write icon to file.
// path argument its file path(no check legality)
func (wi *WinIcon) IconToFile(path, name string, index int) error {
	if index >= wi.getIconsHeaderCount() || index < 0 {
		return ErrIconsIndex
	}
	wis := wi.icos[index]
	p := filepath.Join(path, name)
	d, e := wi.GetImageData(index)
	if e != nil {
		return e
	}
	// 处理bitmap头结构
	if GetIconType(d) == typeBMP {
		w := wis.getIconWidth()
		h := wis.getIconHeight()
		b := wis.getIconBitsPerPixel()
		s := len(d) - dibHeaderSize
		dib := createDIBHeader(w, h, b, s, 0, 0)
		err := dib.EditDIBHeader(d)
		if err != nil {
			return err
		}
		bmh := createBitmapHeader(len(d))
		d = bmh.JoinHeader(d)
	}
	if e := wis.IconToFile(p, d); e != nil {
		return e
	} else {
		return nil
	}
}

// IconToIcoFile 将ico文件中的指定icon图标数据写入ico文件
// path string: 路径（不检查合法性）
// index int: icon图标的索引
// error: 如果发生错误返回error对象
// Write the specified icon data in the ico file to the ico file.
// path argument its path of file write(no check legality)
func (wi *WinIcon) IconToIcoFile(path string, index int) error {
	if index < 0 || index >= len(wi.icos) {
		return ErrIconsIndex
	}
	d := wi.icos[index].data
	wis := wi.icos[index]
	wis.ImageOffset = fileHeaderSize + headerSize
	wis.ImageDataSize = uint32(len(d))
	d = wis.joinHeader(d)
	if e := ioutil.WriteFile(path, d, getPerm()); e != nil {
		return e
	}
	return nil
}

// getIconsHeaderCount 获取 icons 图标的结构数量-可能和头结构的ico数量不一致，只是可能
// 返回值为数量，类型 int
// Get the number of structures of the icons icon - may not match the number of
// icos in the header structure, just possible
// return count of icon image
func (wi *WinIcon) getIconsHeaderCount() int {
	return len(wi.icos)
}

// generateFileNameFormat 产生文件名
// Generate a formatted file name (customPrefix_icon64x64@24bit.extname)
func (wis winIconStruct) generateFileNameFormat(prefix, ext string, width, height, bit int) string {
	return fmt.Sprintf("%s_icon%dx%d@%dbit.%s", prefix, width, height, bit, ext)
}

// iconToFile 将ico图像数据写入磁盘文件
// Write ico image data to disk file
func (wis winIconStruct) IconToFile(path string, data []byte) error {
	if err := ioutil.WriteFile(path, data, getPerm()); err != nil {
		return err
	}
	return nil
}

// headerToBytes 将头结构数据转换为[]byte字节切片
// Convert header structure data to byte slice
func (wis winIconStruct) headerToBytes(both bool) []byte {
	var d []byte
	if both {
		d = make([]byte, fileHeaderSize+headerSize)
		binary.LittleEndian.PutUint16(d[0:2], 0)
		binary.LittleEndian.PutUint16(d[2:4], 1)
		binary.LittleEndian.PutUint16(d[4:6], 1)
		d[6] = uint8(wis.getIconWidth())
		d[7] = uint8(wis.getIconHeight())
		d[8] = wis.Palette
		d[9] = wis.ReservedB
		binary.LittleEndian.PutUint16(d[10:12], wis.ColorPlanes)
		binary.LittleEndian.PutUint16(d[12:14], wis.BitsPerPixel)
		binary.LittleEndian.PutUint32(d[14:18], wis.ImageDataSize)
		binary.LittleEndian.PutUint32(d[18:22], wis.ImageOffset)
	} else {
		d = make([]byte, headerSize)
		d[0] = uint8(wis.getIconWidth())
		d[1] = uint8(wis.getIconHeight())
		d[2] = wis.Palette
		d[3] = wis.ReservedB
		binary.LittleEndian.PutUint16(d[4:6], wis.ColorPlanes)
		binary.LittleEndian.PutUint16(d[6:8], wis.BitsPerPixel)
		binary.LittleEndian.PutUint32(d[8:12], wis.ImageDataSize)
		binary.LittleEndian.PutUint32(d[12:16], wis.ImageOffset)
	}
	return d
}

// joinHeader 对bmp的icon数据添加头结构
// bmp格式的icon图标，是没有BITMAPFILEHEADER的，所以导出的时候，我们给其添加一个头结构
// The icon in bmp format does not have BITMAPFILEHEADER,
// so when exporting, we add a header structure to it.
func (wis winIconStruct) joinHeader(d []byte) []byte {
	h := wis.headerToBytes(true)
	j := [][]byte{h, d}
	return bytes.Join(j, nil)
}

// getIconOffset 获取icon图像数据的偏移量
// 返回偏移量数据
// get offset of icon image data, return offset
func (wis winIconStruct) getIconOffset() int {
	return int(wis.ImageOffset)
}

// setIconOffset 设置icon图像数据的偏移量
// set offset of icon image structure
func (wis winIconStruct) setIconOffset(o int) {
	wis.ImageOffset = uint32(o)
}

// getIconLength 获取icon图像数据的长度
// 返回长度数据
// get data length of icon image, return size(length)
func (wis winIconStruct) getIconLength() int {
	return int(wis.ImageDataSize)
}

// setIconLength 设置icon图标数据的大小
// set size of icon image data
func (wis winIconStruct) setIconLength(l int) {
	wis.ImageDataSize = uint32(l)
}

// getIconWidth 获取icon图像数据的宽度
// return width of icon image
func (wis winIconStruct) getIconWidth() int {
	if wis.Width == 0 {
		return 256
	}
	return int(wis.Width)
}

// setIconWidth 设置icon图标数据的高度
// set width of icon image data
func (wis winIconStruct) setIconWidth(w int) {
	wis.Width = uint8(w)
}

// getIconHeight 获取icon图像数据的高度
// return height of icon image
func (wis winIconStruct) getIconHeight() int {
	if wis.Height == 0 {
		return 256
	}
	return int(wis.Height)
}

// setIconHeight 设置icon图标的高度
// set height of icon image data
func (wis winIconStruct) setIconHeight(h int) {
	wis.Height = uint8(h)
}

// getIconBitsPerPixel 获取icon图像数据的颜色位数
// return image pixel color bits (8bit, 24bit, 32bit)
func (wis winIconStruct) getIconBitsPerPixel() int {
	return int(wis.BitsPerPixel)
}

func (wis winIconStruct) setIconBitsPerPixel(b int) {
	wis.BitsPerPixel = uint16(b)
}

// createDIBHeader创建DIB头结构
// create DIB header structure
func createDIBHeader(width, height, bit, size, p, i int) *dibHeader {
	return &dibHeader{
		dibSize:        dibHeaderSize,
		bitmapWidth:    uint32(width),
		bitmapHeight:   uint32(height),
		colorPlanes:    1,
		BitsPerPixel:   uint16(bit),
		markerBI_RGB:   0,
		originalSize:   uint32(size),
		printResH:      2835,
		printResV:      2835,
		Palette:        uint32(p),
		importantColor: uint32(i),
	}
}

// HeaderToBytes 将DIB的头结构转换为字节切片
// Convert the DIB header structure to a byte slice
func (dh *dibHeader) HeaderToBytes() []byte {
	d := make([]byte, dibHeaderSize)
	binary.LittleEndian.PutUint32(d[0:4], dh.dibSize)
	binary.LittleEndian.PutUint32(d[4:8], dh.bitmapWidth)
	binary.LittleEndian.PutUint32(d[8:12], dh.bitmapHeight)
	binary.LittleEndian.PutUint16(d[12:14], dh.colorPlanes)
	binary.LittleEndian.PutUint16(d[14:16], dh.BitsPerPixel)
	binary.LittleEndian.PutUint32(d[16:20], dh.markerBI_RGB)
	binary.LittleEndian.PutUint32(d[20:24], dh.originalSize)
	binary.LittleEndian.PutUint32(d[24:28], dh.printResH)
	binary.LittleEndian.PutUint32(d[28:32], dh.printResV)
	binary.LittleEndian.PutUint32(d[32:36], dh.Palette)
	binary.LittleEndian.PutUint32(d[36:40], dh.importantColor)
	return d
}

// EditDIBHeader 修改DIB头
// EditDIBHeader edit DIB header
// Its actually the data of the replacement header :)
func (dh *dibHeader) EditDIBHeader(b []byte) error {
	if len(b) <= dibHeaderSize {
		return ErrIconsIndex
	}
	h := dh.HeaderToBytes()
	for i := 0; i < dibHeaderSize; i++ {
		b[i] = h[i]
	}
	return nil
}

// loadImageData 载入图像数据
// rd io.Reader 必须为*os.File对象作为参数
// 成功返回 data 字节切片和 format icon类型
// （类型参考const定义,0为不清楚类型，1为BMP，2为PNG）
// 失败返回 error 对象
// rd io.Reader argument must be a *os.File object as a argument
// Successfully returned byte slice and image type(ICONTYPE uknow or bmp, png).
// Failed to return error object
func loadImageData(rd io.Reader) (data []byte, format ICONTYPE, err error) {
	v, it := rd.(*os.File)
	if !it {
		return nil, typeUKN, ErrIcoReaders
	}
	f, e := v.Stat()
	if e != nil {
		return nil, typeUKN, e
	}
	if f.IsDir() {
		return nil, typeUKN, ErrIcoFileType
	}
	r := bufio.NewReader(rd)
	p, err := r.Peek(pngFileHeaderSize)
	if err != nil {
		return nil, typeUKN, err
	}
	if bytes.Compare(p[0:], PNGHEADER) == 0 {
		d, e := getFileAll(r, f.Size())
		if e != nil {
			return nil, typeUKN, e
		}
		return d, typePNG, nil
	} else {
		b, e := r.Peek(bitmapHeaderSize + dibHeaderSize)
		if e != nil {
			return nil, typeUKN, e
		}
		if bytes.Compare(b[0:2], BMPHEADERID) != 0 {
			return nil, typeUKN, ErrIcoInvalid
		}
		bs := int64(binary.LittleEndian.Uint32(b[2:6]))
		ds := int32(binary.LittleEndian.Uint32(b[14:18]))
		if bs != f.Size() {
			return nil, typeUKN, ErrIcoInvalid
		}
		if ds != dibHeaderSize {
			return nil, typeUKN, ErrIcoInvalid
		}
		d, e := getFileAll(r, f.Size())
		if e != nil {
			return nil, typeUKN, e
		}
		return d, typeBMP, nil
	}
}

// CreateWinIcon 可以将N个BMP和PNG图像打包为一
// 个windows系统的ico文件所需要的结构
// BMP和PNG目前不支持压缩过及带调色板的索引图像
// filePath []string: 文件的路径
// 成功返回 WinIcon 对象的指针
// 失败返回 error 对象
// The structure required to package n's BMP
// and PNG images into a ico file of windows system
// filePath []string: bmp or png file path string array(slice)
// a pointer to a WinIcon object that successfully returns.
// Failed to return error object
func CreateWinIcon(filePath []string) (*WinIcon, error) {
	var (
		fs  []*os.File
		err error
	)
	defer func() { // 可能遇到错误，出错的时候把文件指针关闭
		if fs != nil {
			for _, v := range fs {
				if v != nil {
					v.Close()
				}
			}
		}
	}()
	fs = make([]*os.File, len(filePath))
	for i, v := range filePath {
		fs[i], err = os.OpenFile(v, os.O_RDONLY, getPerm())
		if err != nil {
			return nil, err
		}
	}
	icos := make(WinIconStruct, len(fs))
	for i, v := range fs {
		d, t, e := loadImageData(v)
		if e != nil && e != io.EOF {
			return nil, e
		}
		switch t {
		case typeBMP:
			icos[i] = bmpToIcon(d)
			icos[i].data = d[bitmapHeaderSize:]
		case typePNG:
			d := pngToIconPNG(d)
			if d != nil {
				icos[i] = pngToIcon(d)
				icos[i].data = d
			} else {
				return nil, ErrIcoInvalid
			}
		default:
			return nil, ErrIcoInvalid
		}
	}
	// 根据icon图标的width排个序（升序）
	// Ascending of sort with icon image width
	sort.Sort(icos)
	wi := &WinIcon{
		icos: icos,
	}
	wi.generateOffset()
	return wi, nil
}

// generateOffset 产生对应的数据偏移量
func (wi *WinIcon) generateOffset() {
	c := len(wi.icos)
	c = c*headerSize + fileHeaderSize
	for i, _ := range wi.icos {
		l := wi.icos[i].getIconLength()
		wi.icos[i].ImageOffset = uint32(c)
		c += l
	}
}

// WriteIcoFile 将icon图标打包数据写入磁盘文件
func (wi *WinIcon) WriteIcoFile(filePath, fileName string) {
	var (
		fs  *os.File
		err error
	)
	fp := filepath.Join(filePath, fileName)
	fs, err = os.OpenFile(fp, os.O_CREATE|os.O_RDWR, getPerm())
	if err != nil {
		panic(err)
	}
	defer fs.Close()
	ih := make([]byte, fileHeaderSize)
	binary.LittleEndian.PutUint16(ih[0:2], 0)
	binary.LittleEndian.PutUint16(ih[2:4], 1)
	binary.LittleEndian.PutUint16(ih[4:6], uint16(wi.getIconsHeaderCount()))
	if _, err := fs.Write(ih); err != nil {
		panic(err)
	}
	for _, v := range wi.icos {
		b := v.headerToBytes(false)
		if _, err := fs.Write(b); err != nil {
			panic(err)
		}
	}
	for _, v := range wi.icos {
		w := binary.LittleEndian.Uint32(v.data[8:12])
		binary.LittleEndian.PutUint32(v.data[8:12], w*2)
		if _, err := fs.Write(v.data); err != nil {
			panic(err)
		}
	}
}

// bmpToIcon bmp图像转换到 winIconStruct 对象
// PNG image converted to winIconStruct object
// Currently only supports uncompressed image data,
// and image data without index color has no color
// palette, temporarily set the palette to 0.
func bmpToIcon(b []byte) winIconStruct {
	b = b[bitmapHeaderSize:]
	wis := winIconStruct{
		Width:         uint8(binary.LittleEndian.Uint32(b[4:8])),
		Height:        uint8(binary.LittleEndian.Uint32(b[8:12])),
		Palette:       uint8(0),
		ReservedB:     uint8(0),
		ColorPlanes:   uint16(1),
		BitsPerPixel:  uint16(binary.LittleEndian.Uint16(b[14:16])),
		ImageDataSize: uint32(len(b)),
		ImageOffset:   uint32(0),
	}
	return wis
}

// pngToIcon png图像转换到 winIconStruct 对象
// 目前仅支持非压缩的图像数据，和不含有索引色的图像数据
// 没有调色版，暂时统一设置palette调色版为0，不使用它
// PNG image converted to winIconStruct object
// Currently only supports uncompressed image data,
// and image data without index color has no color
// palette, temporarily set the palette to 0, dont fix it.
// this feature may be improved in the future.
func pngToIcon(b []byte) winIconStruct {
	wis := winIconStruct{
		Width:         uint8(binary.BigEndian.Uint32(b[16:20])),
		Height:        uint8(binary.BigEndian.Uint32(b[20:24])),
		Palette:       uint8(0),
		ReservedB:     uint8(0),
		ColorPlanes:   uint16(1),
		BitsPerPixel:  uint16(32), //b[24]
		ImageDataSize: uint32(len(b)),
		ImageOffset:   uint32(0),
	}
	return wis
}

func pngToIconPNG(b []byte) []byte {
	rd := bytes.NewReader(b)
	i, _ := png.Decode(rd)
	buf := new(bytes.Buffer)
	enc := new(png.Encoder)
	enc.CompressionLevel = png.BestCompression
	e := enc.Encode(buf, i)
	if e != nil {
		return nil
	} else {
		return buf.Bytes()
	}
}

// Len 实现go语言的排序算法接口中Len方法
// Implementing the Go language sorting
// algorithm interface in the Len method
func (w WinIconStruct) Len() int {
	return len(w)
}

// Less 实现go语言的排序算法接口中Less方法
// Implementing the go language sorting
// algorithm interface in the Less method
func (w WinIconStruct) Less(i, j int) bool {
	return w[j].getIconWidth() < w[i].getIconWidth()
}

// Swap 实现go语言的排序算法接口中的Swap方法
// Implementing the go language sorting
// algorithm interface in the Swap method
func (w WinIconStruct) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}
