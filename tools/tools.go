package tools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func ToString(filePath string) (string, error) {
	// 修复: ioutil.ReadFile 自 go1.16 起已 deprecated
	b, err := os.ReadFile(filePath)
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

// RestartNamed 强制重启 named (中断解析服务, 仅作为回退手段)
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

// ReloadNamed 优先使用 rndc reconfig 热加载 zone 配置变更 (不中断解析服务),
// rndc 不可用或失败时回退为 systemctl restart named。
// zone 条目增删属于配置变更, reconfig 即可生效。
func ReloadNamed() error {
	cmd := exec.Command("rndc", "reconfig")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		return nil
	}

	// rndc 失败, 回退到 restart
	return RestartNamed()
}
