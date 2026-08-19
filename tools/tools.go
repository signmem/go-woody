package tools

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

func ToString(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ToTrimString(filePath string) (string, error) {
	str, err := ToString(filePath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(str), nil
}

func RestartNamed() error {
	cmd := exec.Command("systemctl", "restart", "named")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("restart named failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}