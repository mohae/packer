package common

import (
	"fmt"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// This step cleans up forwarded ports and exports the VM to an OVF.
//
// Uses:
//
// Produces:
//   exportPath string - The path to the resulting export.
type StepExport struct {
	Format     string
	OutputDir  string
	ExportOpts []string
}

func (s *StepExport) Run(state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packer.Ui)
	vmName := state.Get("vmName").(string)

	// Wait a second to ensure VM is really shutdown.
	log.Println("1 second timeout to ensure VM is really shutdown")
	time.Sleep(1 * time.Second)

	// Clear out the Packer-created forwarding rule. We have to retry this
	// a number of times because sometimes things are slower than expected.
	// A brief sleep is added between retries to allow the machine to
	// finish termination.
	ui.Say("Preparing to export machine...")
	ui.Message(fmt.Sprintf(
		"Deleting forwarded port mapping for SSH (host port %d)",
		state.Get("sshHostPort")))
	command := []string{"modifyvm", vmName, "--natpf1", "delete", "packerssh"}
	
	// Setup the vars...should some of these be exposed?
	var err error
	var deleted bool
	var retryCount int
	retryMax := 5
	timeout := (200 * time.Millisecond)

	for retryCount < retryMax {

		err = driver.VBoxManage(command...)

		if err == nil {
			deleted = true
			break
		}

		retryCount++
		time.Sleep(timeout)
	}

	if ! deleted {
        	err = fmt.Errorf("Error deleting port forwarding rule: %s", err)
                state.Put("error", err)
                ui.Error(err.Error())
                return multistep.ActionHalt
	}

	// Export the VM to an OVF
	outputPath := filepath.Join(s.OutputDir, vmName+"."+s.Format)

	command = []string{
		"export",
		vmName,
		"--output",
		outputPath,
	}

	command = append(command, s.ExportOpts...)

	ui.Say("Exporting virtual machine...")
	ui.Message(fmt.Sprintf("Executing: %s", strings.Join(command, " ")))
	err = driver.VBoxManage(command...)
	if err != nil {
		err = fmt.Errorf("Error exporting virtual machine: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("exportPath", outputPath)

	return multistep.ActionContinue
}

func (s *StepExport) Cleanup(state multistep.StateBag) {}
