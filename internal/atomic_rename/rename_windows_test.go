// +build windows

package atomic_rename

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const TEST_FILE_COUNT = 500

func TestConcurrentRenames(t *testing.T) {
	var wg sync.WaitGroup

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	trigger := make(chan struct{})
	testDir := filepath.Join(os.TempDir(), fmt.Sprintf("nsqd_TestConcurrentRenames_%d", r.Int()))

	err := os.MkdirAll(testDir, 644)
	if err != nil {
		t.Error(err)
	}

	fis, err := ioutil.ReadDir(testDir)
	if err != nil {
		t.Error(err)
	} else if len(fis) > 0 {
		t.Errorf("Test directory %s unexpectedly has %d items in it!", testDir, len(fis))
		t.FailNow()
	}

	// create a bunch of source files and attempt to concurrently rename them all
	for i := 1; i <= TEST_FILE_COUNT; i++ {
		//First rename doesn't overwrite/replace; no target present
		sourcePath1 := filepath.Join(testDir, fmt.Sprintf("source1_%d.txt", i))
		//Second rename will replace
		sourcePath2 := filepath.Join(testDir, fmt.Sprintf("source2_%d.txt", i))
		targetPath := filepath.Join(testDir, fmt.Sprintf("target_%d.txt", i))
		err = ioutil.WriteFile(sourcePath1, []byte(sourcePath1), 0644)
		if err != nil {
			t.Error(err)
		}
		err = ioutil.WriteFile(sourcePath2, []byte(sourcePath2), 0644)
		if err != nil {
			t.Error(err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = <-trigger
			err := Rename(sourcePath1, targetPath)
			if err != nil {
				t.Error(err)
			}
			err = Rename(sourcePath2, targetPath)
			if err != nil {
				t.Error(err)
			}
		}()
	}

	// start.. they're off to the races!
	close(trigger)

	// wait for completion...
	wg.Wait()

	// no source files should exist any longer; we should just have 500 target files
	fis, err = ioutil.ReadDir(testDir)
	if err != nil {
		t.Error(err)
	} else if len(fis) != TEST_FILE_COUNT {
		t.Errorf("Test directory %s unexpectedly has %d items in it!", testDir, len(fis))
	} else {
		for _, fi := range fis {
			if !strings.HasPrefix(fi.Name(), "target_") {
				t.Errorf("Test directory file %s is not expected target file!", fi.Name())
			}
		}
	}

	// clean up the test directory
	err = os.RemoveAll(testDir)
	if err != nil {
		t.Error(err)
	}
}
