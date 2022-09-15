package script

import (
	"fmt"
	"os"
	"os/exec"

	"zabbix_agent_installer/mylog"
)

// 启动zabbix agent
func StartAgent(scriptAbsPath string) {
	cmd := exec.Command("sh", scriptAbsPath, "restart")
	_, err := cmd.Output()
	if err != nil {
		mylog.Logger("ERROR", "start agent failed "+err.Error())
		os.Exit(1)
	}
	mylog.Logger("INFO", "start agent successful")
}

// 检查进程
func ShowAgentProcess() {
	c2 := exec.Command("sh", "-c", "ps -ef|grep -E 'UID|zabbix' |grep -Ev 'installer|grep'")
	out, err := c2.Output()
	if err != nil {
		mylog.Logger("ERROR", "run ps failed "+err.Error())
		return
	}
	mylog.Logger("INFO", "run ps successful")
	fmt.Print(string(out))
}
