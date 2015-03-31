package parallels

// Large part of this source is taken from
// https://github.com/mitchellh/packer/blob/master/builder/parallels/common

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"os/exec"
	"regexp"
	"runtime"
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
	// TODO: more statuses
)

const (
	prlctlPath = "prlctl"
	dhcpLeases = "/Library/Preferences/Parallels/parallels_dhcp_leases"
)

func PrlctlOutput(args ...string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf(
			"Parallels works only on \"darwin\" platform!")
	}

	var stdout, stderr bytes.Buffer

	logrus.Debugf("Executing PrlctlOutput: %#v", args)
	cmd := exec.Command(prlctlPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	stderrString := strings.TrimSpace(stderr.String())

	if _, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("PrlctlOutput error: %s", stderrString)
	}

	return stdout.String(), err
}

func Prlctl(args ...string) error {
	_, err := PrlctlOutput(args...)
	return err
}

func Exec(vmName string, args ...string) (string, error) {
	args2 := append([]string{"exec", vmName}, args...)
	return PrlctlOutput(args2...)
}

func Version() (string, error) {
	out, err := PrlctlOutput("--version")
	if err != nil {
		return "", err
	}

	versionRe := regexp.MustCompile(`prlctl version (\d+\.\d+.\d+)`)
	matches := versionRe.FindStringSubmatch(string(out))
	if matches == nil {
		return "", fmt.Errorf(
			"Could not find Parallels Desktop version in output:\n%s", string(out))
	}

	version := matches[1]
	logrus.Debugf("Parallels Desktop version: %s", version)
	return version, nil
}

func Exist(name string) bool {
	err := Prlctl("list", name, "--no-header", "--output", "status")
	if err != nil {
		return false
	}
	return true
}

func CreateTemplate(vmName, templateName string) error {
	return Prlctl("clone", vmName, "--name", templateName, "--template", "--linked")
}

func CreateOsVM(vmName, templateName string) error {
	return Prlctl("create", vmName, "--ostemplate", templateName)
}

func CreateSnapshot(vmName, snapshotName string) error {
	return Prlctl("snapshot", vmName, "--name", snapshotName)
}

func GetDefaultSnapshot(vmName string) (string, error) {
	output, err := PrlctlOutput("snapshot-list", vmName)
	if err != nil {
		return "", err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		pos := strings.Index(line, " *")
		if pos >= 0 {
			snapshot := line[pos+2:]
			snapshot = strings.TrimSpace(snapshot)
			if len(snapshot) > 0 { // It uses UUID so it should be 38
				return snapshot, nil
			}
		}
	}

	return "", errors.New("No snapshot")
}

func RevertToSnapshot(vmName, snapshotID string) error {
	return Prlctl("snapshot-switch", vmName, "--id", snapshotID)
}

func Start(vmName string) error {
	return Prlctl("start", vmName)
}

func Status(vmName string) (StatusType, error) {
	output, err := PrlctlOutput("list", vmName, "--no-header", "--output", "status")
	if err != nil {
		return NotFound, err
	}
	return StatusType(strings.TrimSpace(output)), nil
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

func TryExec(vmName string, seconds int, cmd ...string) error {
	var err error
	for i := 0; i < seconds; i++ {
		_, err = Exec(vmName, cmd...)
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return err
}

func Kill(vmName string) error {
	return Prlctl("stop", vmName, "--kill")
}

func Stop(vmName string) error {
	return Prlctl("stop", vmName)
}

func Delete(vmName string) error {
	return Prlctl("delete", vmName)
}

func Unregister(vmName string) error {
	return Prlctl("unregister", vmName)
}

func Mac(vmName string) (string, error) {
	output, err := PrlctlOutput("list", "-i", vmName)
	if err != nil {
		return "", err
	}

	stdoutString := strings.TrimSpace(output)
	re := regexp.MustCompile("net0.* mac=([0-9A-F]{12}) card=.*")
	macMatch := re.FindAllStringSubmatch(stdoutString, 1)

	if len(macMatch) != 1 {
		return "", fmt.Errorf("MAC address for NIC: nic0 on Virtual Machine: %s not found!\n", vmName)
	}

	mac := macMatch[0][1]
	logrus.Debugf("Found MAC address for NIC: net0 - %s\n", mac)
	return mac, nil
}

// IPAddress finds the IP address of a VM connected that uses DHCP by its MAC address
//
// Parses the file /Library/Preferences/Parallels/parallels_dhcp_leases
// file contain a list of DHCP leases given by Parallels Desktop
// Example line:
// 10.211.55.181="1418921112,1800,001c42f593fb,ff42f593fb000100011c25b9ff001c42f593fb"
// IP Address   ="Lease expiry, Lease time, MAC, MAC or DUID"
func IPAddress(mac string) (string, error) {
	if len(mac) != 12 {
		return "", fmt.Errorf("Not a valid MAC address: %s. It should be exactly 12 digits.", mac)
	}

	leases, err := ioutil.ReadFile(dhcpLeases)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("(.*)=\"(.*),(.*)," + strings.ToLower(mac) + ",.*\"")
	mostRecentIP := ""
	mostRecentLease := uint64(0)
	for _, l := range re.FindAllStringSubmatch(string(leases), -1) {
		ip := l[1]
		expiry, _ := strconv.ParseUint(l[2], 10, 64)
		leaseTime, _ := strconv.ParseUint(l[3], 10, 32)
		logrus.Debugf("Found lease: %s for MAC: %s, expiring at %d, leased for %d s.\n", ip, mac, expiry, leaseTime)
		if mostRecentLease <= expiry-leaseTime {
			mostRecentIP = ip
			mostRecentLease = expiry - leaseTime
		}
	}

	if len(mostRecentIP) == 0 {
		return "", fmt.Errorf("IP lease not found for MAC address %s in: %s\n", mac, dhcpLeases)
	}

	logrus.Debugf("Found IP lease: %s for MAC address %s\n", mostRecentIP, mac)
	return mostRecentIP, nil
}
