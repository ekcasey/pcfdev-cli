package vbox

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/address"
	"github.com/pivotal-cf/pcfdev-cli/config"
	. "github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/network"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vboxdriver"
	"os"
)

//go:generate mockgen -package mocks -destination mocks/driver.go github.com/pivotal-cf/pcfdev-cli/vbox Driver
type Driver interface {
	VBoxManage(arg ...string) (output []byte, err error)
	StartVM(vmName string) error
	VMExists(vmName string) (exists bool, err error)
	PowerOffVM(vmName string) error
	StopVM(vmName string) error
	SuspendVM(vmName string) error
	ResumeVM(vmName string) error
	DestroyVM(vmName string) error
	VMs() (vms []string, err error)
	Disks() (disks []string, err error)
	RunningVMs() (vms []string, err error)
	CreateHostOnlyInterface(ip string) (interfaceName string, err error)
	ConfigureHostOnlyInterface(interfaceName string, ip string) error
	AttachNetworkInterface(interfaceName string, vmName string) error
	ForwardPort(vmName string, ruleName string, hostPort string, guestPort string) error
	IsInterfaceInUse(interfaceName string) (bool, error)
	GetHostForwardPort(vmName string, ruleName string) (port string, err error)
	GetHostOnlyInterfaces() (interfaces []*network.Interface, err error)
	SetCPUs(vmName string, cpuNumber int) error
	SetMemory(vmName string, memory uint64) error
	CreateVM(vmName string, baseDirectory string) error
	AttachDisk(vmName string, diskPath string) error
	CloneDisk(src string, dest string) error
	DeleteDisk(diskPath string) error
	UseDNSProxy(vmName string) error
	GetMemory(vmName string) (uint64, error)
	VMState(vmName string) (string, error)
	Version() (version *vboxdriver.VBoxDriverVersion, err error)
}

//go:generate mockgen -package mocks -destination mocks/fs.go github.com/pivotal-cf/pcfdev-cli/vbox FS
type FS interface {
	Exists(path string) (exists bool, err error)
	Extract(archivePath string, destinationPath string, filename string) error
	Remove(path string) error
	Write(path string, contents io.Reader, append bool) error
	Read(path string) (contents []byte, err error)
	Chmod(path string, mode os.FileMode) error
}

//go:generate mockgen -package mocks -destination mocks/ssh.go github.com/pivotal-cf/pcfdev-cli/vbox SSH
type SSH interface {
	GenerateAddress() (host string, port string, err error)
	GenerateKeypair() (privateKey []byte, publicKey []byte, err error)
	RunSSHCommand(command string, addresses []ssh.SSHAddress, privateKey []byte, timeout time.Duration, stdout io.Writer, stderr io.Writer) error
}

//go:generate mockgen -package mocks -destination mocks/picker.go github.com/pivotal-cf/pcfdev-cli/vbox NetworkPicker
type NetworkPicker interface {
	SelectAvailableInterface(vboxnets []*network.Interface, vmConfig *config.VMConfig) (networkConfig *config.NetworkConfig, err error)
}

type VBox struct {
	Config *config.Config
	Driver Driver
	FS     FS
	Picker NetworkPicker
	SSH    SSH
}

type VMProperties struct {
	IPAddress string
}

type ProxyTypes struct {
	HTTPProxy  string
	HTTPSProxy string
	NOProxy    string
}

const (
	StatusRunning    = "Running"
	StatusSaved      = "Saved"
	StatusPaused     = "Paused"
	StatusStopped    = "Stopped"
	StatusNotCreated = "Not created"
	StatusUnknown    = "Unknown"
)

var (
	networkTemplate = `
auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp

auto eth1
iface eth1 inet static
address {{.IPAddress}}
netmask 255.255.255.0`

	proxyTemplate = `
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games
{{if .HTTPProxy}}HTTP_PROXY={{.HTTPProxy}}{{end}}
{{if .HTTPSProxy}}HTTPS_PROXY={{.HTTPSProxy}}{{end}}
NO_PROXY={{.NOProxy}}
{{if .HTTPProxy}}http_proxy={{.HTTPProxy}}{{end}}
{{if .HTTPSProxy}}https_proxy={{.HTTPSProxy}}{{end}}
no_proxy={{.NOProxy}}`
)

func (v *VBox) StartVM(vmConfig *config.VMConfig) error {
	if err := v.Driver.StartVM(vmConfig.Name); err != nil {
		return err
	}

	if err := v.insertSecureKeypair(vmConfig); err != nil {
		return err
	}

	if err := v.configureNetwork(vmConfig); err != nil {
		return err
	}
	if err := v.configureEnvironment(vmConfig); err != nil {
		return err
	}

	if err := v.Driver.StopVM(vmConfig.Name); err != nil {
		return err
	}

	return v.Driver.StartVM(vmConfig.Name)
}

func (v *VBox) insertSecureKeypair(vmConfig *config.VMConfig) error {
	exists, err := v.FS.Exists(v.Config.PrivateKeyPath)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	privateKey, publicKey, err := v.SSH.GenerateKeypair()
	if err != nil {
		return err
	}

	if err = v.SSH.RunSSHCommand(
		fmt.Sprintf(`echo -n "%s" > /home/vcap/.ssh/authorized_keys`, publicKey),
		[]ssh.SSHAddress{
			{
				IP:   "127.0.0.1",
				Port: vmConfig.SSHPort,
			},
			{
				IP:   vmConfig.IP,
				Port: "22",
			},
		},
		v.Config.InsecurePrivateKey,
		5*time.Minute,
		ioutil.Discard,
		ioutil.Discard,
	); err != nil {
		return err
	}

	return v.writePrivateKey(privateKey)
}

func (v *VBox) writePrivateKey(privateKey []byte) error {
	if err := v.FS.Write(v.Config.PrivateKeyPath, bytes.NewReader(privateKey), false); err != nil {
		return err
	}
	return v.FS.Chmod(v.Config.PrivateKeyPath, 0600)
}

func (v *VBox) configureNetwork(vmConfig *config.VMConfig) error {
	privateKeyBytes, err := v.FS.Read(v.Config.PrivateKeyPath)
	if err != nil {
		return err
	}

	t, err := template.New("properties template").Parse(networkTemplate)
	if err != nil {
		return err
	}

	var sshCommand bytes.Buffer
	if err = t.Execute(&sshCommand, VMProperties{IPAddress: vmConfig.IP}); err != nil {
		return err
	}

	return v.SSH.RunSSHCommand(
		fmt.Sprintf("echo -e '%s' | sudo tee /etc/network/interfaces", sshCommand.String()),
		[]ssh.SSHAddress{
			{
				IP:   "127.0.0.1",
				Port: vmConfig.SSHPort,
			},
			{
				IP:   vmConfig.IP,
				Port: "22",
			},
		},
		privateKeyBytes,
		5*time.Minute,
		ioutil.Discard,
		ioutil.Discard,
	)
}

func (v *VBox) configureEnvironment(vmConfig *config.VMConfig) error {
	proxySettings, err := v.proxySettings(vmConfig)
	if err != nil {
		return err
	}

	privateKeyBytes, err := v.FS.Read(v.Config.PrivateKeyPath)
	if err != nil {
		return err
	}

	return v.SSH.RunSSHCommand(
		fmt.Sprintf("echo -e '%s' | sudo tee /etc/environment", proxySettings),
		[]ssh.SSHAddress{
			{
				IP:   "127.0.0.1",
				Port: vmConfig.SSHPort,
			},
			{
				IP:   vmConfig.IP,
				Port: "22",
			},
		},
		privateKeyBytes,
		5*time.Minute,
		ioutil.Discard,
		ioutil.Discard,
	)
}

func (v *VBox) proxySettings(vmConfig *config.VMConfig) (settings string, err error) {
	subnet, err := address.SubnetForIP(vmConfig.IP)
	if err != nil {
		return "", err
	}

	httpProxy := strings.Replace(v.Config.HTTPProxy, "127.0.0.1", subnet, -1)
	httpsProxy := strings.Replace(v.Config.HTTPSProxy, "127.0.0.1", subnet, -1)
	noProxy := strings.Join([]string{
		"localhost",
		"127.0.0.1",
		subnet,
		vmConfig.IP,
		vmConfig.Domain,
		"." + vmConfig.Domain,
	}, ",")
	if v.Config.NoProxy != "" {
		noProxy = strings.Join([]string{noProxy, v.Config.NoProxy}, ",")
	}

	t, err := template.New("proxy template").Parse(proxyTemplate)
	if err != nil {
		return "", err
	}

	var proxySettings bytes.Buffer
	if err = t.Execute(&proxySettings, ProxyTypes{HTTPProxy: httpProxy, HTTPSProxy: httpsProxy, NOProxy: noProxy}); err != nil {
		return "", err
	}

	return proxySettings.String(), nil
}

func (v *VBox) ImportVM(vmConfig *config.VMConfig) error {
	if err := v.Driver.CreateVM(vmConfig.Name, v.Config.VMDir); err != nil {
		return err
	}

	compressedDisk := filepath.Join(v.Config.VMDir, vmConfig.Name+"-disk1.vmdk") + ".compressed"
	uncompressedDisk := filepath.Join(v.Config.VMDir, vmConfig.Name, vmConfig.Name+"-disk1.vmdk")
	if err := v.FS.Extract(vmConfig.OVAPath, compressedDisk, `\w+\.vmdk`); err != nil {
		return err
	}

	if err := v.Driver.CloneDisk(compressedDisk, uncompressedDisk); err != nil {
		return err
	}

	if err := v.Driver.DeleteDisk(compressedDisk); err != nil {
		return err
	}

	if err := v.Driver.AttachDisk(vmConfig.Name, uncompressedDisk); err != nil {
		return err
	}

	vboxInterfaces, err := v.Driver.GetHostOnlyInterfaces()
	if err != nil {
		return err
	}

	networkConfig, err := v.Picker.SelectAvailableInterface(vboxInterfaces, vmConfig)
	if err != nil {
		return err
	}

	if networkConfig.Interface.Exists {
		if err := v.Driver.ConfigureHostOnlyInterface(networkConfig.Interface.Name, networkConfig.Interface.IP); err != nil {
			return err
		}
	} else {
		interfaceName, err := v.Driver.CreateHostOnlyInterface(networkConfig.Interface.IP)
		if err != nil {
			return err
		}
		networkConfig.Interface.Name = interfaceName
	}

	if err := v.Driver.AttachNetworkInterface(networkConfig.Interface.Name, vmConfig.Name); err != nil {
		return err
	}

	if err := v.FS.Write(
		filepath.Join(v.Config.VMDir, "vm_config"),
		strings.NewReader(fmt.Sprintf(`{"ip":"%s","domain":"%s"}`, networkConfig.VMIP, networkConfig.VMDomain)),
		false,
	); err != nil {
		return err
	}

	if err := v.Driver.UseDNSProxy(vmConfig.Name); err != nil {
		return err
	}

	_, sshPort, err := v.SSH.GenerateAddress()
	if err != nil {
		return err
	}

	if err := v.Driver.ForwardPort(vmConfig.Name, "ssh", sshPort, "22"); err != nil {
		return err
	}

	if err := v.Driver.SetCPUs(vmConfig.Name, vmConfig.CPUs); err != nil {
		return err
	}

	if err := v.Driver.SetMemory(vmConfig.Name, vmConfig.Memory); err != nil {
		return err
	}

	return nil
}

func (v *VBox) DestroyVM(vmConfig *config.VMConfig) error {
	return v.Driver.DestroyVM(vmConfig.Name)
}

func (v *VBox) PowerOffVM(vmConfig *config.VMConfig) error {
	return v.Driver.PowerOffVM(vmConfig.Name)
}

func (v *VBox) GetVMName() (name string, err error) {
	vms, err := v.Driver.VMs()
	if err != nil {
		return "", err
	}
	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") {
			if name == "" {
				name = vm
			} else {
				return "", errors.New("multiple PCF Dev VMs found")
			}
		}
	}
	return name, nil
}

func (v *VBox) StopVM(vmConfig *config.VMConfig) error {
	return v.Driver.StopVM(vmConfig.Name)
}

func (v *VBox) SuspendVM(vmConfig *config.VMConfig) error {
	return v.Driver.SuspendVM(vmConfig.Name)
}

func (v *VBox) ResumePausedVM(vmConfig *config.VMConfig) error {
	return v.Driver.ResumeVM(vmConfig.Name)
}

func (v *VBox) ResumeSavedVM(vmConfig *config.VMConfig) error {
	return v.Driver.StartVM(vmConfig.Name)
}

func (v *VBox) DestroyPCFDevVMs() error {
	vms, err := v.Driver.VMs()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") {
			IgnoreErrorFrom(v.Driver.PowerOffVM(vm))
			IgnoreErrorFrom(v.Driver.DestroyVM(vm))
		}
	}

	vms, err = v.Driver.VMs()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		if strings.HasPrefix(vm, "pcfdev-") {
			return errors.New("failed to destroy all pcfdev vms")
		}
	}

	disks, err := v.Driver.Disks()
	if err != nil {
		return err
	}

	for _, disk := range disks {
		filename := filepath.Base(disk)
		if strings.HasPrefix(filename, "pcfdev-") {
			IgnoreErrorFrom(v.Driver.DeleteDisk(disk))
		}
	}

	disks, err = v.Driver.Disks()
	if err != nil {
		return err
	}

	for _, disk := range disks {
		filename := filepath.Base(disk)
		if strings.HasPrefix(filename, "pcfdev-") {
			return errors.New("failed to destroy all pcfdev disks")
		}
	}
	return nil
}

func (v *VBox) VMConfig(vmName string) (*config.VMConfig, error) {
	memory, err := v.Driver.GetMemory(vmName)
	if err != nil {
		return nil, err
	}
	port, err := v.Driver.GetHostForwardPort(vmName, "ssh")
	if err != nil {
		return nil, err
	}
	vmConfigBytes, err := v.FS.Read(filepath.Join(v.Config.VMDir, "vm_config"))
	if err != nil {
		return nil, err
	}

	vmConfig := &config.VMConfig{
		Memory:   memory,
		Name:     vmName,
		SSHPort:  port,
		Provider: "virtualbox",
	}
	if err := json.Unmarshal(vmConfigBytes, &vmConfig); err != nil {
		return nil, err
	}

	return vmConfig, nil
}

func (v *VBox) VMStatus(vmName string) (status string, err error) {
	exists, err := v.Driver.VMExists(vmName)
	if err != nil {
		return "", err
	}

	if !exists {
		return StatusNotCreated, nil
	}

	state, err := v.Driver.VMState(vmName)
	if err != nil {
		return "", err
	}

	switch state {
	case vboxdriver.StateRunning:
		return StatusRunning, nil
	case vboxdriver.StateStopped, vboxdriver.StateAborted:
		return StatusStopped, nil
	case vboxdriver.StateSaved:
		return StatusSaved, nil
	case vboxdriver.StatePaused:
		return StatusPaused, nil
	default:
		return StatusUnknown, nil
	}
}

func (v *VBox) Version() (version *vboxdriver.VBoxDriverVersion, err error) {
	return v.Driver.Version()
}
