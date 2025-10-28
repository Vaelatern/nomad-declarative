package submission

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type CmdReturn struct {
	ProgName string
	Err      error
}

func (c CmdReturn) Error() string {
	return fmt.Sprintf("%s: %v", c.ProgName, c.Err)
}

// ExecuteFilesAsync runs executable files in nested directories and collects errors
func ExecuteFilesAsync(compiledDir string) error {
	errChan := make(chan error, 100)

	var wg sync.WaitGroup

	err := filepath.Walk(compiledDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Mode()&0111 != 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cmd := exec.Command("./" + filepath.Base(path))
				cmd.Args[0] = filepath.Base(path)
				cmd.Dir = filepath.Dir(path)
				cmd.Env = os.Environ()
				fmt.Printf("Executing %s: %v\n", path, cmd)
				err := cmd.Run()
				errChan <- CmdReturn{ProgName: path, Err: err}
			}()
		}
		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		wg.Wait() // Should be a no-op...
		close(errChan)
	}()

	var finalError error
	for err := range errChan {
		if err.(CmdReturn).Err != nil {
			finalError = errors.Join(finalError, err)
		}
	}

	return finalError
}
