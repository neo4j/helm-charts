package helpers

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"strings"
)

// RunCommand runs the command and returns its standard
// output and standard error.
func RunCommand(c *exec.Cmd) ([]byte, []byte, error) {
	log.Printf("Command := %s", strings.Join(c.Args, " "))
	if c.Stdout != nil {
		return nil, nil, errors.New("exec: Stdout already set")
	}
	if c.Stderr != nil {
		return nil, nil, errors.New("exec: Stderr already set")
	}
	var a bytes.Buffer
	var b bytes.Buffer
	c.Stdout = &a
	c.Stderr = &b
	err := c.Run()
	return a.Bytes(), b.Bytes(), err
}
