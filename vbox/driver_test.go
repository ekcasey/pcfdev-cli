package vbox_test

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/helpers"
	"github.com/pivotal-cf/pcfdev-cli/ssh"
	"github.com/pivotal-cf/pcfdev-cli/vbox"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var vBoxManagePath string

var _ = BeforeSuite(func() {
	var err error
	vBoxManagePath, err = helpers.VBoxManagePath()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("driver", func() {
	var driver *vbox.VBoxDriver
	var vmName string

	BeforeEach(func() {
		driver = &vbox.VBoxDriver{}

		var err error
		vmName, err = helpers.ImportSnappy()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		driver.VBoxManage("controlvm", vmName, "poweroff")
		driver.VBoxManage("unregistervm", vmName, "--delete")
	})

	Describe("#VBoxManage", func() {
		It("should execute VBoxManage with given args", func() {
			output, err := driver.VBoxManage("help")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("Oracle VM VirtualBox Command Line Management Interface"))
		})

		It("should return any errors with their output", func() {
			output, err := driver.VBoxManage("some-bad-command")
			Expect(err).To(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("Syntax error: Invalid command 'some-bad-command'"))
		})
	})

	Describe("#GetVMIP", func() {
		Context("when interface exists", func() {
			var interfaceName string

			BeforeEach(func() {
				var err error
				interfaceName, err = driver.CreateHostOnlyInterface("192.168.88.1")
				Expect(err).NotTo(HaveOccurred())

				err = driver.AttachNetworkInterface(interfaceName, vmName)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				command := exec.Command(vBoxManagePath, "hostonlyif", "remove", interfaceName)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			})

			It("should return the ip of the vm", func() {
				ip, err := driver.GetVMIP(vmName)
				Expect(err).NotTo(HaveOccurred())
				Expect(ip).To(Equal("192.168.88.11"))
			})
		})

		Context("when interface does not exist", func() {
			It("should return an error message", func() {
				_, err := driver.GetVMIP(vmName)
				Expect(err).To(MatchError("there is no attached hostonlyif for " + vmName))
			})
		})

		Context("when VBoxManage command fails", func() {
			It("should return the output of the failed command", func() {
				_, err := driver.GetVMIP("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage showvminfo some-bad-vm-name --machinereadable': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#GetMemory", func() {
		BeforeEach(func() {
			err := exec.Command(vBoxManagePath, "modifyvm", vmName, "--memory", "4567").Run()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return the vm memory", func() {
			memory, err := driver.GetMemory(vmName)
			Expect(err).NotTo(HaveOccurred())
			Expect(memory).To(Equal(uint64(4567)))
		})

		Context("when VBoxManage command fails", func() {
			It("should return the output of the failed command", func() {
				_, err := driver.GetMemory("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage showvminfo some-bad-vm-name --machinereadable': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("when starting and stopping and suspending and resuming and destroying the VM", func() {
		It("should start, stop, suspend, resume, and then destroy a VBox VM", func() {
			sshClient := &ssh.SSH{}
			_, port, err := sshClient.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())

			err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
			Expect(err).NotTo(HaveOccurred())

			err = driver.StartVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Expect(driver.VMState(vmName)).To(Equal(vbox.StateRunning))

			stdout := gbytes.NewBuffer()
			err = sshClient.RunSSHCommand("hostname", port, 5*time.Minute, stdout, ioutil.Discard)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stdout.Contents())).To(ContainSubstring("ubuntu-core-stable-15"))

			err = driver.StopVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() (string, error) { return driver.VMState(vmName) }, 120*time.Second).Should(Equal(vbox.StateStopped))

			Expect(driver.StartVM(vmName)).To(Succeed())
			Expect(driver.VMState(vmName)).To(Equal(vbox.StateRunning))

			err = driver.SuspendVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() (string, error) { return driver.VMState(vmName) }, 120*time.Second).Should(Equal(vbox.StateSaved))

			err = driver.ResumeVM(vmName)
			Expect(err).NotTo(HaveOccurred())
			Expect(driver.VMState(vmName)).To(Equal(vbox.StateRunning))

			Expect(driver.PowerOffVM(vmName)).To(Succeed())
			Expect(driver.VMState(vmName)).To(Equal(vbox.StateStopped))

			err = driver.DestroyVM(vmName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				exists, err := driver.VMExists(vmName)
				Expect(err).NotTo(HaveOccurred())
				return exists
			}, 120*time.Second).Should(BeFalse())
		})
	})

	Describe("#StartVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.StartVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage startvm some-bad-vm-name --type headless': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#CreateVM", func() {
		var createdVMName string

		BeforeEach(func() {
			createdVMName = "some-created-vm"
		})

		AfterEach(func() {
			exec.Command(vBoxManagePath, "controlvm", createdVMName, "poweroff").Run()
			exec.Command(vBoxManagePath, "unregistervm", createdVMName, "--delete").Run()
		})

		It("should create VM", func() {
			basedir := os.TempDir()
			err := driver.CreateVM(createdVMName, basedir)
			command := exec.Command(vBoxManagePath, "list", "vms")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(createdVMName))
		})
	})

	Describe("#VMExists", func() {
		Context("when VM exists", func() {
			It("should return true", func() {
				exists, err := driver.VMExists(vmName)
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})

		Context("when VM does not exist", func() {
			It("should return false", func() {
				exists, err := driver.VMExists("does-not-exist")
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})
	})

	Describe("#VMState", func() {
		Context("when the VM is running", func() {
			It("should return StateRunning", func() {
				sshClient := &ssh.SSH{}

				_, port, err := sshClient.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
				Expect(err).NotTo(HaveOccurred())

				err = driver.StartVM(vmName)
				Expect(err).NotTo(HaveOccurred())
				state, err := driver.VMState(vmName)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(vbox.StateRunning))
			})
		})

		Context("when the VM is saved", func() {
			It("should return StateSaved", func() {
				sshClient := &ssh.SSH{}

				_, port, err := sshClient.GenerateAddress()
				Expect(err).NotTo(HaveOccurred())

				err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
				Expect(err).NotTo(HaveOccurred())

				err = driver.StartVM(vmName)
				Expect(err).NotTo(HaveOccurred())
				err = driver.SuspendVM(vmName)

				Expect(driver.VMState(vmName)).To(Equal(vbox.StateSaved))
			})
		})

		Context("when the VM is stopped", func() {
			It("should return StateStopped", func() {
				Expect(driver.VMState(vmName)).To(Equal(vbox.StateStopped))
			})
		})

		Context("when VBoxManage command fails", func() {
			It("should return the output of the failed command", func() {
				_, err := driver.VMState("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage showvminfo some-bad-vm-name --machinereadable': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#StopVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.StopVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage controlvm some-bad-vm-name acpipowerbutton': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#SuspendVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.SuspendVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage controlvm some-bad-vm-name savestate': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#ResumeVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.ResumeVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage startvm some-bad-vm-name --type headless': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#PowerOffVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.PowerOffVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage controlvm some-bad-vm-name poweroff': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#DestroyVM", func() {
		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				err := driver.DestroyVM("some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage unregistervm some-bad-vm-name --delete': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#AttachDisk", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "pcfdev-vbox-driver")
			Expect(err).NotTo(HaveOccurred())

			Expect(exec.Command(
				vBoxManagePath, "createvm", "--name", "some-vm", "--ostype", "Ubuntu_64", "--basefolder", tmpDir, "--register").Run(),
			).To(Succeed())
			Expect(exec.Command(
				vBoxManagePath, "createmedium", "disk", "--filename", filepath.Join(tmpDir, "some-disk.vmdk"), "--size", "1", "--format", "VMDK").Run(),
			).To(Succeed())
		})

		AfterEach(func() {
			exec.Command(vBoxManagePath, "unregistervm", "some-vm", "--delete").Run()
			exec.Command(vBoxManagePath, "closemedium", "disk", filepath.Join(tmpDir, "some-disk.vmdk")).Run()
			os.RemoveAll(tmpDir)
		})

		It("should attach disk", func() {
			Expect(driver.AttachDisk("some-vm", filepath.Join(tmpDir, "some-disk.vmdk"))).To(Succeed())

			command := exec.Command(vBoxManagePath, "showvminfo", "some-vm", "--machinereadable")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`storagecontrollername0="SATA"`))
			Expect(session).To(gbytes.Say(fmt.Sprintf(`"SATA-0-0"="%s"`, filepath.Join(tmpDir, "some-disk.vmdk"))))
		})

		Context("when adding the storage controller fails", func() {
			It("should return an error", func() {
				Expect(driver.AttachDisk("some-bad-vm", "some-disk.vmdk")).To(
					MatchError(ContainSubstring("failed to execute 'VBoxManage storagectl some-bad-vm --name SATA --add sata':")))
			})
		})

		Context("when attaching the storage fails", func() {
			It("should return an error", func() {
				Expect(driver.AttachDisk("some-vm", "some-bad-disk")).To(
					MatchError(ContainSubstring("failed to execute 'VBoxManage storageattach some-vm --storagectl SATA --medium some-bad-disk --type hdd --port 0 --device 0':")))
			})
		})
	})

	Describe("#CreateHostOnlyInterface", func() {
		var interfaceName string

		AfterEach(func() {
			command := exec.Command(vBoxManagePath, "hostonlyif", "remove", interfaceName)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should create a hostonlyif", func() {
			var err error
			interfaceName, err = driver.CreateHostOnlyInterface("192.168.77.1")
			Expect(err).NotTo(HaveOccurred())

			var name string
			var ipAddress string
			var netMask string
			var output []byte
			output, err = exec.Command(vBoxManagePath, "list", "hostonlyifs").Output()
			Expect(err).NotTo(HaveOccurred())

			nameRegex := regexp.MustCompile(`(?m:^Name:\s+(.*))`)
			nameMatches := nameRegex.FindAllStringSubmatch(string(output), -1)

			ipRegex := regexp.MustCompile(`(?m:^IPAddress:\s+(.*))`)
			ipMatches := ipRegex.FindAllStringSubmatch(string(output), -1)

			netMaskRegex := regexp.MustCompile(`(?m:^NetworkMask:\s+(.*))`)
			netMaskRegexMatches := netMaskRegex.FindAllStringSubmatch(string(output), -1)

			for i := 0; i < len(nameMatches); i++ {
				if strings.TrimSpace(nameMatches[i][1]) == interfaceName {
					name = strings.TrimSpace(nameMatches[i][1])
					ipAddress = strings.TrimSpace(ipMatches[i][1])
					netMask = strings.TrimSpace(netMaskRegexMatches[i][1])
				}
			}

			Expect(name).To(Equal(interfaceName))
			Expect(ipAddress).To(Equal("192.168.77.1"))
			Expect(netMask).To(Equal("255.255.255.0"))
		})
	})

	Describe("#GetHostOnlyInterfaces", func() {
		var interfaceName string
		var expectedIP string

		BeforeEach(func() {
			expectedIP = "192.168.55.55"
			output, err := exec.Command(vBoxManagePath, "hostonlyif", "create").Output()
			Expect(err).NotTo(HaveOccurred())
			regex := regexp.MustCompile(`Interface '(.*)' was successfully created`)
			matches := regex.FindStringSubmatch(string(output))
			interfaceName = matches[1]
			assignIP := exec.Command(vBoxManagePath, "hostonlyif", "ipconfig", interfaceName, "--ip", expectedIP, "--netmask", "255.255.255.0")
			session, err := gexec.Start(assignIP, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			command := exec.Command(vBoxManagePath, "hostonlyif", "remove", interfaceName)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should return a slice of network.Interfaces representing the list of VBox nets", func() {
			interfaces, err := driver.GetHostOnlyInterfaces()
			Expect(err).NotTo(HaveOccurred())

			for _, iface := range interfaces {
				if iface.Name == interfaceName {
					Expect(iface.IP).To(Equal(expectedIP))
					return
				}
			}
			Fail(fmt.Sprintf("did not create interface with expected name %s", interfaceName))
		})
	})

	Describe("#AttachInterface", func() {
		var interfaceName string

		BeforeEach(func() {
			var err error
			interfaceName, err = driver.CreateHostOnlyInterface("192.168.77.1")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			command := exec.Command(vBoxManagePath, "hostonlyif", "remove", interfaceName)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should attach a hostonlyif to the vm", func() {
			err := driver.AttachNetworkInterface(interfaceName, vmName)
			Expect(err).NotTo(HaveOccurred())

			showvmInfoCommand := exec.Command(vBoxManagePath, "showvminfo", vmName, "--machinereadable")
			session, err := gexec.Start(showvmInfoCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`hostonlyadapter2="` + interfaceName + `"`))
			Expect(session).To(gbytes.Say(`nic2="hostonly"`))
		})

		Context("when attaching a hostonlyif fails", func() {
			It("should return an error", func() {
				err := driver.AttachNetworkInterface("some-interface-name", "some-bad-vm-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage modifyvm some-bad-vm-name --nic2 hostonly --hostonlyadapter2 some-interface-name': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#ForwardPort", func() {
		It("should forward guest port to the given host port", func() {
			sshClient := &ssh.SSH{}
			_, port, err := sshClient.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())

			err = driver.ForwardPort(vmName, "some-rule-name", port, "22")
			Expect(err).NotTo(HaveOccurred())
			err = driver.StartVM(vmName)
			Expect(err).NotTo(HaveOccurred())

			stdout := gbytes.NewBuffer()
			err = sshClient.RunSSHCommand("hostname", port, 5*time.Minute, stdout, ioutil.Discard)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stdout.Contents())).To(ContainSubstring("ubuntu-core-stable-15"))
		})

		Context("when forwarding a port fails", func() {
			It("should return an error", func() {
				err := driver.ForwardPort("some-bad-vm-name", "some-rule-name", "some-host-port", "some-guest-port")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage modifyvm some-bad-vm-name --natpf1 some-rule-name,tcp,127.0.0.1,some-host-port,,some-guest-port': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#GetHostForwardPort", func() {
		It("should return the forwarded port on the host", func() {
			sshClient := &ssh.SSH{}
			_, expectedPort, err := sshClient.GenerateAddress()
			Expect(err).NotTo(HaveOccurred())

			err = driver.ForwardPort(vmName, "some-rule-name", expectedPort, "22")
			Expect(err).NotTo(HaveOccurred())

			port, err := driver.GetHostForwardPort(vmName, "some-rule-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(port).To(Equal(expectedPort))
		})

		Context("when no port is forwarded", func() {
			It("should return an error", func() {
				_, err := driver.GetHostForwardPort(vmName, "some-bad-rule-name")
				Expect(err).To(MatchError("could not find forwarded port"))
			})
		})

		Context("when VM with the given name does not exist", func() {
			It("should return an error", func() {
				_, err := driver.GetHostForwardPort("some-bad-vm-name", "some-rule-name")
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage showvminfo some-bad-vm-name --machinereadable': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#SetMemory", func() {
		It("should set vm memory in mb", func() {
			Expect(driver.SetMemory(vmName, uint64(2048))).To(Succeed())

			showvmInfoCommand := exec.Command(vBoxManagePath, "showvminfo", vmName, "--machinereadable")
			session, err := gexec.Start(showvmInfoCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`memory=2048`))
		})

		Context("when setting memory fails", func() {
			It("should return an error", func() {
				err := driver.SetMemory("some-bad-vm-name", uint64(0))
				Expect(err).To(MatchError(ContainSubstring("failed to execute 'VBoxManage modifyvm some-bad-vm-name --memory 0': exit status 1")))
				Expect(err).To(MatchError(ContainSubstring("Could not find a registered machine named 'some-bad-vm-name'")))
			})
		})
	})

	Describe("#SetCPUs", func() {
		It("should set vm cpus", func() {
			Expect(driver.SetCPUs(vmName, 2)).To(Succeed())

			showvmInfoCommand := exec.Command(vBoxManagePath, "showvminfo", vmName, "--machinereadable")
			session, err := gexec.Start(showvmInfoCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`cpus=2`))
		})

		Context("when setting memory fails", func() {
			It("should return an error", func() {
				Expect(driver.SetCPUs("some-bad-vm-name", 1)).To(MatchError(ContainSubstring("failed to execute 'VBoxManage modifyvm some-bad-vm-name --cpus 1'")))
			})
		})
	})

	Describe("#VMs", func() {
		It("should return a list of VMs", func() {
			Expect(driver.VMs()).To(ContainElement(vmName))

			Expect(driver.StartVM(vmName)).To(Succeed())

			Expect(driver.VMs()).To(ContainElement(vmName))
		})
	})

	Describe("#RunningVMs", func() {
		It("should return a list of running VMs", func() {
			Expect(driver.RunningVMs()).NotTo(ContainElement(vmName))

			Expect(driver.StartVM(vmName)).To(Succeed())

			Expect(driver.RunningVMs()).To(ContainElement(vmName))
		})
	})

	Describe("#IsInterfaceInUse", func() {
		Context("when there is a VM assigned to the given hostonlyifs", func() {
			var interfaceName string

			BeforeEach(func() {
				var err error
				interfaceName, err = driver.CreateHostOnlyInterface("192.168.88.1")
				Expect(err).NotTo(HaveOccurred())

				err = driver.AttachNetworkInterface(interfaceName, vmName)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				command := exec.Command(vBoxManagePath, "hostonlyif", "remove", interfaceName)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			})

			It("should return true", func() {
				Expect(driver.IsInterfaceInUse(interfaceName)).To(BeTrue())
			})
		})

		Context("when there is not a VM assigned to the given hostonlyifs", func() {
			It("should return false", func() {
				Expect(driver.IsInterfaceInUse("some-bad-interface")).To(BeFalse())
			})
		})
	})

	Describe("#CloneDisk", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "pcfdev-vbox-driver")
			Expect(err).NotTo(HaveOccurred())

			archive, err := os.Open(filepath.Join("..", "assets", "snappy.ova"))
			Expect(err).NotTo(HaveOccurred())

			reader := tar.NewReader(archive)
			for {
				header, err := reader.Next()
				if err == io.EOF {
					break
				}
				Expect(err).NotTo(HaveOccurred())

				if header.Name == "Snappy-disk1.vmdk" {
					file, err := os.OpenFile(filepath.Join(tmpDir, "compressed-Snappy-disk1.vmdk"), os.O_WRONLY|os.O_CREATE, 0644)
					Expect(err).NotTo(HaveOccurred())
					defer file.Close()

					_, err = io.Copy(file, reader)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})

		AfterEach(func() {
			exec.Command(vBoxManagePath, "closemedium", "disk", filepath.Join(tmpDir, "cloned-Snappy-disk1.vmdk")).Run()
			os.RemoveAll(tmpDir)
		})

		It("should clone a disk", func() {
			Expect(driver.CloneDisk(filepath.Join(tmpDir, "compressed-Snappy-disk1.vmdk"), filepath.Join(tmpDir, "cloned-Snappy-disk1.vmdk"))).To(Succeed())

			command := exec.Command(vBoxManagePath, "showmediuminfo", "disk", filepath.Join(tmpDir, "cloned-Snappy-disk1.vmdk"))
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say(`Storage format: VMDK`))

			command = exec.Command(vBoxManagePath, "list", "hdds")
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session).NotTo(gbytes.Say("compressed-Snappy-disk1.vmdk"))
		})

		Context("when cloning fails", func() {
			It("should return an error", func() {
				Expect(driver.CloneDisk("some-bad-src", "cloned-Snappy-disk1.vmdk")).To(
					MatchError(ContainSubstring("failed to execute 'VBoxManage clonemedium disk some-bad-src cloned-Snappy-disk1.vmdk':")))
			})
		})
	})
})
