package virtualbox

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type StatusType string

const (
	NotFound  StatusType = "notfound"
	Invalid              = "invalid"
	Stopped              = "stopped"
	Suspended            = "suspended"
	Running              = "running"
	Saved                = "saved"
	// TODO: more statuses
)

func VboxManageOutput(exe string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	log.Debugf("Executing VBoxManageOutput: %#v", args)
	cmd := exec.Command(exe, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	stderrString := strings.TrimSpace(stderr.String())

	if _, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("VBoxManageOutput error: %s", stderrString)
	}

	return stdout.String(), err
}

func VBoxManage(args ...string) (string, error) {
	return VboxManageOutput("vboxmanage", args...)
}

func Version() (string, error) {
	version, err := VBoxManage("--version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(version), nil
}

func FindSSHPort(vmName string) (string, error) {
	info, err := VBoxManage("showvminfo", vmName)
	portRe := regexp.MustCompile(`guestssh.*host port = (\d+)`)
	sshPort := portRe.FindStringSubmatch(info)
	return sshPort[1], err
}

func Exist(vmName string) bool {
	_, err := VBoxManage("showvminfo", vmName)
	if err != nil {
		return false
	}
	return true
}

func CreateOsVM(vmName string, templateName string) error {
	_, err := VBoxManage("clonevm", vmName, "--mode", "machine", "--name", templateName, "--register")
	return err
}

func FindNextPort(highport string, usedPorts [][]string) string {
	for _, port := range usedPorts {
		if highport == port[1] {
			var temp int
			temp, _ = strconv.Atoi(highport)
			temp = temp + 1
			highport = strconv.Itoa(temp)
			highport = FindNextPort(highport, usedPorts)
		}
	}
	return highport
}

func isPortUnassigned(highport string, usedPorts [][]string) bool {
	for _, port := range usedPorts {
		if highport == port[1] {
			return false
		}
	}
	return true
}

func getUsedVirtualboxPorts() ([][]string, error) {
	output, err := VBoxManage("list", "vms", "-l")
	if err != nil {
		return nil, err
	}
	allPortsRe := regexp.MustCompile(`host port = (\d+)`)
	usedPorts := allPortsRe.FindAllStringSubmatch(output, -1)
	log.Debugln(usedPorts)
	return usedPorts, nil
}

func ConfigureSSH(vmName string, vmSSHPort string) error {
	var localport string

	for {
		ln, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Debugln(err)
			return err
		}
		defer ln.Close()

		addressElements := strings.Split(ln.Addr().String(), ":")
		localport = addressElements[len(addressElements)-1]

		usedPorts, err := getUsedVirtualboxPorts()
		if err != nil {
			log.Debugln(err)
			return err
		}

		if isPortUnassigned(localport, usedPorts) {
			break
		}
		// We're going around the loop again, getting a new ln
		ln.Close()
	}

	rule := fmt.Sprintf("guestssh,tcp,127.0.0.1,%s,,%s", localport, vmSSHPort)
	_, err := VBoxManage("modifyvm", vmName, "--natpf1", rule)
	return err
}

func CreateSnapshot(vmName string, snapshotName string) error {
	_, err := VBoxManage("snapshot", vmName, "take", snapshotName)
	return err
}

func RevertToSnapshot(vmName string) error {
	_, err := VBoxManage("snapshot", vmName, "restorecurrent")
	return err
}

func Start(vmName string) error {
	_, err := VBoxManage("startvm", vmName, "--type", "headless")
	return err
}

func Stop(vmName string) error {
	_, err := VBoxManage("controlvm", vmName, "poweroff")
	return err
}

func Kill(vmName string) error {
	_, err := VBoxManage("controlvm", vmName, "acpipowerbutton")
	return err
}

func Delete(vmName string) error {
	_, err := VBoxManage("unregistervm", vmName, "--delete")
	return err
}

func Status(vmName string) (StatusType, error) {
	output, err := VBoxManage("showvminfo", vmName, "--machinereadable")
	statusRe := regexp.MustCompile(`VMState="(\w+)"`)
	status := statusRe.FindStringSubmatch(output)
	if err != nil {
		return NotFound, err
	}
	return StatusType(status[1]), nil
}

func WaitForStatus(vmName string, vmStatus StatusType, seconds int) error {
	var status StatusType
	var err error
	for i := 0; i < seconds; i++ {
		status, err = Status(vmName)
		if err != nil {
			return err
		}
		if status == vmStatus {
			return nil
		}
		time.Sleep(time.Second)
	}
	return errors.New("VM " + vmName + " is in " + string(status) + " where it should be in " + string(vmStatus))
}

func Unregister(vmName string) error {
	_, err := VBoxManage("unregistervm", vmName)
	return err
}
