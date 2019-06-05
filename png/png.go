/*
   _____       __   __             _  __
  ╱ ____|     |  ╲/   |           | |/ /
 | |  __  ___ |  ╲ /  | __  _ _ __| ' /
 | | |_ |/ _ ╲| |╲ /| |/ _`  | '__|  <
 | |__| |  __/| |   | (  _|  | |  | . ╲
  ╲_____|╲___ |_|   |_|╲__,_ |_|  |_|╲_╲
 可爱飞行猪❤: golang83@outlook.com  💯💯💯
 Author Name: GeMarK.VK.Chow奥迪哥  🚗🔞🈲
 Creaet Time: 2019/06/04 - 18:32:33
 ProgramFile: png.go
 Description: PNG图片解析工具

*/

package png

import (
	"bufio"
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"strings"
)

const (
	// 数据长度的定义 Data length or size
	PNGHEADSIZE = 8
	CTLENGTH    = 4
	CIHDRLEN    = 13
	// 关键块 Critical chunks
	CIHDR = "IHDR" // IHDR必须是第一块(顺序的总共13个数据字节)
	CIDAT = "IDAT" // IDAT块包含实际图像数据，可以在多个IDAT块之间进行分割，它是压缩算法的输出流
	CIEND = "IEND" // 标志着图像结束。(内容是固定的，见变量定义的`ChunkIEND`)
	CPLTE = "PLTE" // PLTE 块是彩色类型3(基本索引颜色)
	// 辅助块 Ancillary chunks
	// CbKGD = "bKGD" // 给出默认的背景颜色
	// CcHRM = "cHRM" // 给出显示原色和白点的色度坐标
	// CdSIG = "dSIG" // 用于存储数字签名
	// CeXIf = "eXIf" // 存储Exif元数据
	// CgAMA = "gAMA" // 指定伽玛
	// ChIST = "hIST" // 可以存储直方图或图像中每种颜色的总量
	// CiCCP = "iCCP" // 是ICC颜色配置文件
	// CiTXt = "iTXt" // 包含关键字和UTF-8文本
	CpHYs = "pHYs" // 保持预期的像素大小（或像素长宽比）
	// CsBIT = "sBIT" // (有效位)表示源数据的颜色精度
	// CsPLT = "sPLT" // 如果全部颜色不可用，建议使用调色板
	// CsRGB = "sRGB" // 表示使用标准sRGB颜色空间
	// CsTER = "sTER" // 用于立体图像的立体图像指示器块
	// CtEXt = "tEXt" // 可以存储可以在ISO / IEC 8859-1中表示的文本
	// CtIME = "tIME" // 存储上次更改图像的时间
	// CtRNS = "tRNS" // 包含透明度信息
	// CzTXt = "zTXt" // 包含与tEXt具有相同限制的压缩文本（和压缩方法标记）
)

var (
	PNGHEAD = []byte{
		0x89, 0x50, 0x4E, 0x47, // 0x89 PNG
		0x0D, 0x0A, 0x1A, 0x0A,
	} // PNG 文件的头(固定大小固定内容)
	ChunkIEND = []byte{
		0x00, 0x00, 0x00, 0x00, // Length alawys 0
		0x49, 0x45, 0x4E, 0x44, // IEND string
		0xAE, 0x42, 0x60, 0x82, // CRC32 value
	} // PNG 文件的数据结尾(固定大小固定内容)
)

// 定义更清晰的类型 :)
// Well-defined type definition
type (
	PNGImage  pngStruct   // PNGImage为PNG图像结构
	Chunk     chunkStruct // Chunk为PNG图像的块结构
	Chunks    []Chunk
	ChunkData []byte      // 块数据
	CRC32     []byte      // 循环冗余检测数据
	Header    []byte      // 头数据
	ImageData []byte      // 图像数据
	IDATS     []ImageData // 图像数据(PNG可能会有多个IDAT块)
	PNGBODY   []byte      // 整个PNG文件的数据
)

// PNG 图像的二进制数据实际上是以文件头 file header 以及 chunk 块组合而成。
// 块数据的以大端序在组成，分别为：Length,ChunkType，Data，CRC四个元素组成。
// The binary data of a PNG image is actually a combination of
// a file header file header and a chunk block.
// The block data is composed of big endian, They are composed
// of four elements: Length, ChunkType, Data, and CRC.
type chunkStruct struct {
	Length    int       // 块数据长度 chunk data length
	ChunkType string    // 块数据类型 chunk type
	Data      ChunkData // 块数据 chunk data
	Crc       CRC32     // 块数据的CRC32验证数据 CRC32 of chunk data
}

// PNG 的整体结构
// Overall structure
type pngStruct struct {
	FileHeader Header // 8 bytes
	Chunks     Chunks // At least 3 chunk: IHDR, IDAT, IEND or more chunk
	IDAT       IDATS  // IDAT datas
}

// New 创建一个PNGImage对象返回对象的指针
// create PNGImage Object Pointer
func New() *PNGImage {
	return new(PNGImage)
}

// NewChunk 创建一个Chunk对象返回对象的指针
// create Chunk object and return object pointer
func NewChunk(length int, chunkName string, data ChunkData, crc CRC32) *Chunk {
	return &Chunk{
		Length:    length,
		ChunkType: chunkName,
		Data:      data,
		Crc:       crc,
	}
}

// check 文件头的检测
// check file header 8bytes check
func (h Header) check() bool {
	if bytes.Compare(h, PNGHEAD) == 0 {
		return true
	}
	return false
}

// ParsePNGImage 解析PNG图像数据(块解析)
// Parse PNG Image chunk
func (pb PNGBODY) ParsePNGImage(img *PNGImage) error {
	l := list.New()
	// get IHDR chunk 获取IHDR块
	hdr, e := pb.getChunk(CIHDR)
	if e != nil {
		return e
	} else {
		l.PushBack(hdr)
	}
	// get PLTE chunk 获取调色板块
	// 这个可能有，可能没有，见前面定义的注释
	plt, e := pb.getChunk(CPLTE)
	if plt != nil {
		l.PushBack(plt)
	} else {
		if e != nil && strings.Contains(e.Error(), "crc") {
			return e
		}
	}
	// get IDAT and processing IDATS
	// 获取 IDAT 块，可能有多个
	e = pb.procIDATS(l, img)
	if e != nil {
		return e
	}
	// get IEND chunk
	// 获取PNG文件的结尾块
	end, e := pb.getChunk(CIEND)
	if e != nil {
		return e
	}
	l.PushBack(end)
	// 链表数据转换到 Chunks
	img.listToChunks(l)
	return nil
}

func (pb PNGBODY) procIDATS(l *list.List, img *PNGImage) error {
	dc := pb.searchIDATChunk()
	if len(dc) < 1 {
		return errors.New(CIDAT + " chunk not found")
	}
	idats, e := pb.getIDATChunk(dc)
	if e != nil {
		return e
	}
	img.IDAT = make(IDATS, len(idats))
	for i, v := range idats {
		img.IDAT[i] = ImageData((*v).Data)
		l.PushBack(*v)
	}
	return nil
}

func (img *PNGImage) listToChunks(l *list.List) {
	img.Chunks = make(Chunks, l.Len())
	for i := 0; i < l.Len(); i++ {
		elm := l.Front()
		obj, _ := (elm.Value).(Chunk)
		img.Chunks[i] = Chunk(obj)
	}
}

// getUint32 获取二进制数据中以大端序存放的Uint32类型数据
// Get Uint32 type data stored in big endian in binary data
func getUint32(buf PNGBODY) (v int, err error) {
	if len(buf) == 0 || len(buf) > 4 {
		return 0, errors.New("Invalid buffer")
	}
	return int(binary.BigEndian.Uint32(buf)), nil
}

// getChunk 获取PNG中的chunk块
// 成功返回 *Chunk
// 失败返回 error
func (pb PNGBODY) getChunk(ctn string) (*Chunk, error) {
	c := pb.searchChunk(ctn)
	if c == -1 {
		return nil, errors.New(ctn + " chunk not found")
	} else {
		l, e := getUint32(pb[c-CTLENGTH : c])
		if e != nil {
			return nil, e
		}
		i := c + CTLENGTH
		o := i + l
		c := pb[o : o+CTLENGTH]
		ch := NewChunk(l, ctn,
			ChunkData(pb[i:o]),
			CRC32(c),
		)
		if !ch.Crc.check(ch) {
			return nil, errors.New(ctn + " chunk crc error")
		}
		return ch, nil
	}
}

// check CRC32 循环冗余检测
// 将chunk中的crc32数据与我们自己生成的crc32数据进行比对
// cyclic redundancy check(32bit)
// Compare the crc32 data in the chunk
// with our own generated crc32 data.
func (c CRC32) check(ck *Chunk) bool {
	a, e := getUint32(PNGBODY(ck.Crc))
	if e != nil {
		return false
	}
	b := int(crc32.ChecksumIEEE(bytes.Join([][]byte{
		[]byte(ck.ChunkType),
		ck.Data},
		[]byte("")),
	))
	if a == b {
		return true
	} else {
		return false
	}
}

// searchChunk 根据chunk块的chunkTypeCode，即chunk的
// 名字搜索数据中chunk块
// 如果找到了，返回数据中的偏移量，如果找不到，则返回 -1
// Search for the chunk block in the data according to
// the chunkTypeCode of the chunk block, that is, the
// name of the chunk If found, returns the offset in
// the data, or -1 if not found.
func (pb PNGBODY) searchChunk(c string) int {
	return bytes.Index(pb, []byte(c))
}

// searchIDATChunk 获取IDAT块的偏移量(可能有多个)
// Get the offset of the IDAT block (possibly multiple)
func (pb PNGBODY) searchIDATChunk() []int {
	offset := make([]int, 256)
	start := 0
	end := len(pb)
	var j int
	for j = 0; j < len(offset); j++ {
		i := bytes.Index(pb[start:end], []byte(CIDAT))
		if i == -1 {
			break
		}
		l := int(binary.BigEndian.Uint32(pb[i-CTLENGTH : i]))
		offset[j] = i
		start = l
	}
	return offset[:j]
}

// getIDATChunk 获取IDAT块的偏移量(可能有多个IDAT块)
// Get the offset of the IDAT chunk (possibly multiple)
func (pb PNGBODY) getIDATChunk(offset []int) ([]*Chunk, error) {
	cs := make([]*Chunk, len(offset))
	for j, v := range offset {
		l, e := getUint32(pb[v-CTLENGTH : v])
		if e != nil {
			return nil, e
		}
		i := v + CTLENGTH
		o := i + l
		c := pb[o : o+CTLENGTH]
		ch := NewChunk(l, CIDAT,
			ChunkData(pb[i:o]),
			CRC32(c),
		)
		if !ch.Crc.check(ch) {
			return nil, errors.New(CIDAT + " chunk crc error")
		}
		cs[j] = ch
	}
	return cs, nil
}

// GetPNGSize 获取已得到的文件数据大小
// 可用于和io.Reader转换为*os.File后，
// 得到的FileInfo对象的文件大小进行比对
// Get the size of the obtained file data
// Used to compare the file size of the
// resulting FileInfo object after converting
// it to *os.File with io.Reader
func (pb PNGBODY) GetPNGSzie() int {
	return len(pb)
}

// LoadPNGFile 载入 PNG 文件的数据(包含解析)
// load png file data, and parse chunk data.
func (img *PNGImage) LoadPNGFile(rd io.Reader) error {
	r := bufio.NewReader(rd)
	hb, e := r.Peek(PNGHEADSIZE)
	if e != nil {
		return e
	}
	img.FileHeader = hb
	if !img.FileHeader.check() {
		return errors.New("Invalid header data")
	}
	s, e := img.getReaderSize(rd)
	if e != nil {
		return e
	}
	b := img.loadAllBytes(r, s)
	if b == nil || b.GetPNGSzie() != s {
		return errors.New("Nil slice or load data error")
	}
	if e := b.ParsePNGImage(img); e != nil {
		return e
	}
	return nil
}

// getReaderSize 获取 io.Reader -> *os.File 的大小
// io.Reader 实际上是一个 *os.File 指针，通过该指针
// 我们可以获取实际的文件大小
// io.Reader is actually a *os.File pointer,
// through which we can get the actual file size
func (img *PNGImage) getReaderSize(rd io.Reader) (int, error) {
	fs, is := rd.(*os.File)
	if is {
		fi, e := fs.Stat()
		if e != nil {
			return -1, e
		}
		return int(fi.Size()), nil
	} else {
		return -1, errors.New("not file pointer")
	}
}

// loadAllBytes 载入png文件的所有数据，返回 PNGBODY
// load png file all data(return PNGBODY or nil)
func (img *PNGImage) loadAllBytes(rd *bufio.Reader, size int) PNGBODY {
	var e error
	p := make(PNGBODY, size)
	for i := 0; i < size; i++ {
		p[i], e = rd.ReadByte()
		if e != nil {
			break
		}
	}
	return p
}
