package compilers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/eris-ltd/common/go/common"
)

func init() {
	ClearCaches()
}

func copyFile(src, dst string) error {
	cmd := exec.Command("cp", src, dst)
	return cmd.Run()
}

func testCache(t *testing.T) {
	ClearCaches()

	resp := Compile("tests/test-inc1.lll", "")
	code := resp.Objects[0].Bytecode
	if resp.Error != "" {
		t.Fatal(fmt.Errorf(resp.Error))
	}
	fmt.Printf("%x\n", code)
	copyFile("tests/test-inc1.lll", path.Join(common.LllcScratchPath, "test-inc1.lll"))
	copyFile("tests/test-inc2.lll", path.Join(common.LllcScratchPath, "test-inc2.lll"))
	copyFile("tests/test-inc4.lll", path.Join(common.LllcScratchPath, "test-inc3.lll"))
	cur, _ := os.Getwd()
	os.Chdir(common.LllcScratchPath)
	resp = Compile(path.Join(common.LllcScratchPath, "test-inc1.lll"), "")
	code2 := resp.Objects[0].Bytecode
	if resp.Error != "" {
		t.Fatal(fmt.Errorf(resp.Error))
	}
	fmt.Printf("%x\n", code2)
	if bytes.Compare(code, code2) == 0 {
		t.Fatal("failed to update cached file")
	}
	os.Chdir(cur)
}

func TestCacheLocal(t *testing.T) {
	SetLanguageNet("lll", false)
	testCache(t)
}

// func TestCacheRemote(t *testing.T) {
// 	SetLanguageNet("lll", true)
// 	testCache(t)
// }

func TestSimple(t *testing.T) {
	ClearCaches()
	SetLanguageNet("lll", false)
	resp := Compile("tests/test-inc1.lll", "")
	code := resp.Objects[0].Bytecode
	if resp.Error != "" {
		t.Fatal(fmt.Errorf(resp.Error))
	}
	fmt.Printf("%x\n", code)
}
