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
	"fmt"
	"os"
	"testing"
)

func TestPNGImage_LoadPNGFile(t *testing.T) {
	p := "../img/vkico256x256@32bit.png"
	f, e := os.Open(p)
	if e != nil {
		panic(e)
	}
	defer f.Close()
	ipng := New()
	e = ipng.LoadPNGFile(f)
	if e != nil {
		t.Logf("TestPNGImage_LoadPNGFile() = %v", e)
	}
	fmt.Println(len(ipng.IDAT))
	fmt.Println(len(ipng.Chunks))
}
