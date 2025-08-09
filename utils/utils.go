package utils

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func IsCommented(line string, symbol string) bool {
	if line != "" {
		return string(line[0]) == symbol
	}
	return true
}

func InsensitiveReplace(s, old, new string) string { //code by Jan Tungli
	if old == new || old == "" {
		return s // avoid allocation
	}
	t := strings.ToLower(s)
	o := strings.ToLower(old)

	// Compute number of replacements.
	n := strings.Count(t, o)
	if n == 0 {
		return s // avoid allocation
	}
	// Apply replacements to buffer.
	var b strings.Builder
	b.Grow(len(s) + n*(len(new)-len(old)))
	start := 0
	for i := 0; i < n; i++ {
		j := start
		if len(old) == 0 {
			if i > 0 {
				_, wid := utf8.DecodeRuneInString(s[start:])
				j += wid
			}
		} else {
			j += strings.Index(t[start:], o)
		}
		b.WriteString(s[start:j])
		b.WriteString(new)
		start = j + len(old)
	}
	b.WriteString(s[start:])
	return b.String()
}

func RemoveFromArrayString(slice []string, num int) []string {
	// if num == len(slice)-1 {
	// 	return slice[:num]
	// }
	return append(slice[:num], slice[num+1:]...)
}

func CreateLog(file string, silentConsole bool) error {
	// removing previous logfile if finded
	if _, err := os.Stat(file); err == nil {
		err = os.Remove(file)
		if err != nil {
			return fmt.Errorf("can't remove a file '%s': %v", file, err)
		}
	}
	// creating log folder if needed
	d := filepath.Dir(file)
	if d != "." {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			err = os.Mkdir(d, 0200)
			if err != nil {
				return fmt.Errorf("can't create a folder '%s': %v", d, err)
			}
		}
	}
	// creating logfile
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0200)
	if err != nil {
		return fmt.Errorf("can't create a file '%s': %v", file, err)
	}
	// setting output
	if silentConsole {
		log.SetOutput(f)
	} else {
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}
	//mw := io.MultiWriter(os.Stdout, f)
	//defer f.Close()
	//log = log.New(l, "", 0)
	log.SetFlags(0)
	//log.SetOutput(mw)
	log.Println("Log created at", time.Now())

	return nil
}

func CLS() error {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("can't call cmd: %v", err)
	}
	return nil
}

func ConvertSecondsToMinutesSeconds(secondsRaw int) string {
	var minutes int = secondsRaw / 60
	var seconds int = secondsRaw % 60
	t := strconv.Itoa(minutes) + " min, " + strconv.Itoa(seconds) + " sec"
	return t
}

func ConvertMillisecondsSecondsToMinutesSeconds(millisecondsRaw int) string {
	var minutes int = millisecondsRaw / 60000
	var seconds int = (millisecondsRaw % 60000) / 1000
	t := strconv.Itoa(minutes) + " min, " + strconv.Itoa(seconds) + " sec"
	return t
}

func Is64bit() bool {
	arch := runtime.GOARCH
	if arch == "amd64" || arch == "arm64" || arch == "arm64be" || arch == "loong64" || arch == "mips64" || arch == "mips64le" || arch == "ppc64" || arch == "ppc64le" || arch == "riscv64" || arch == "s390x" || arch == "sparc64" || arch == "wasm" {
		return true
	}
	return false
}

func UnwrapErrCompletely(err error) error {
	if err == nil {
		return nil
	}
	var errUnwrapped error
	for err != nil {
		errUnwrapped = err
		err = errors.Unwrap(err)
	}
	return errUnwrapped
}

func ReturnArchitecture() string {
	arch := runtime.GOARCH
	return arch
}

func ReturnWindowsVersion() string {
	cmd := exec.Command("cmd", "/C", "ver")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("Unknown: can't read cmd output: %s", err.Error())
	}
	return strings.Split(string(out), "\n")[1]
}

func PrintStringArray(str []string) string {
	if len(str) == 0 {
		return ""
	}
	line := str[0]
	for i := 1; i < len(str); i++ {
		line = line + " " + str[i]
	}
	return line
}

func SetTitle(t string) error {
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: "/C title " + t}
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("can't properly execute call to cmd.exe: %v", err)
	}
	return nil
}

func AmAdmin(quiet bool) bool { //code by jerblack
	if !quiet {
		log.Println("Checking privilegies")
	}
	elevated := windows.GetCurrentProcessToken().IsElevated()
	if !quiet {
		log.Printf("Admin rights: %t\n", elevated)
	}
	return elevated
}

func RunMeElevated(quiet bool) error { //code inspired by jerblack
	if !quiet {
		log.Println("Relaunching with elevation")
	}
	verb := "runas"
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("can't return the path to the process' executable: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't return the path to the current directory: %v", err)
	}

	var argsQuoted []string
	for i := 1; i < len(os.Args); i++ {
		argsQuoted = append(argsQuoted, fmt.Sprintf("%q", os.Args[i]))
	}
	args := strings.Join(argsQuoted, " ")

	verbPtr, err := syscall.UTF16PtrFromString(verb)
	if err != nil {
		return fmt.Errorf("can't form pointer: %v", err)
	}
	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return fmt.Errorf("can't form pointer: %v", err)
	}
	cwdPtr, err := syscall.UTF16PtrFromString(cwd)
	if err != nil {
		return fmt.Errorf("can't form pointer: %v", err)
	}
	argPtr, err := syscall.UTF16PtrFromString(args)
	if err != nil {
		return fmt.Errorf("can't form pointer: %v", err)
	}

	var showCmd int32 = 1 //SW_NORMAL

	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		return fmt.Errorf("can't execute shell command: %v", err)
	}
	return nil
}

func StopAndDeleteServices(snames ...string) error {
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("can't establish connection to the service manager: %v", err)
	} else {
		log.Println("Established connection to service manager")
	}
	defer manager.Disconnect()

	for _, sname := range snames {
		service, err := manager.OpenService(sname)
		if err != nil {
			log.Printf("Can't access the '%s' service, skipping: %v\n", sname, err)
			continue
		} else {
			log.Printf("Service '%s' is opened for interaction\n", sname)
		}
		defer service.Close()

		status, err := service.Query()
		if err != nil {
			return fmt.Errorf("can't query the state of '%s' service: %v", sname, err)
		}
		if status.State == svc.Running {
			_, err := service.Control(svc.Stop)
			if err != nil {
				return fmt.Errorf("can't stop the '%s' service: %v", sname, err)
			} else {
				log.Printf("Service '%s' was received a stop signal\n", sname)
			}
		} else {
			log.Printf("Service '%s' isn't currently running\n", sname)
		}

		err = service.Delete()
		if err != nil && err.Error() != "The specified service has been marked for deletion." {
			return fmt.Errorf("could not delete the service: %v", err)
		} else {
			log.Printf("Service '%s' was deleted\n", sname)
		}
	}

	return nil
}

func TaskKill(tasks ...string) error {
	for _, t := range tasks {
		log.Printf("Terminating process '%s'\n", t)
		cmd := exec.Command("taskkill", "/IM", t, "/T", "/F")
		err := cmd.Run()
		if err != nil && err.Error() == "exit status 128" {
			log.Printf("Process '%s' not found\n", t)
		} else if err == nil {
			log.Printf("Process '%s' was terminated\n", t)
		} else {
			return fmt.Errorf("can't terminate process '%s': %v", t, err)
		}
	}
	return nil
}

func StartProgramWithArguments(prog string, args []string) (*exec.Cmd, error) {
	// var args2 []string
	// for _, a := range args {
	// 	args2 = append(args2, fmt.Sprintf("\"%s\"", a))
	// }

	exe := exec.Command(prog)
	exe.SysProcAttr = &syscall.SysProcAttr{CmdLine: PrintStringArray(args)}
	err := exe.Start()
	if err != nil {
		return exe, fmt.Errorf("program didn't start properly: %v", err)
	}
	return exe, nil
}

func StopProgram(exe *exec.Cmd) error {
	exist, err := PidExists(int32(exe.Process.Pid))
	if err != nil {
		return fmt.Errorf("failed to check if process with PID do exist: %v", err)
	}
	if exist {
		err := exe.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to terminate process: %v", err)
		}
	} else {
		log.Println("Can't find process: either it wasn't properly started, has exited already, has crushed or something arlready terminated it")
	}
	return nil
}

func PidExists(pid int32) (bool, error) { // code by shirou
	if pid == 0 { // special case for pid 0 System Idle Process
		return true, nil
	}
	if pid < 0 {
		return false, fmt.Errorf("invalid pid %v", pid)
	}
	if pid%4 != 0 {
		// OpenProcess will succeed even on non-existing pid here https://devblogs.microsoft.com/oldnewthing/20080606-00/?p=22043
		return false, fmt.Errorf("pid %v incorrect: it should be a multiplier of 4", pid)
	}
	const STILL_ACTIVE = 259 // https://docs.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-getexitcodeprocess
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err == windows.ERROR_ACCESS_DENIED {
		return true, nil
	}
	if err == windows.ERROR_INVALID_PARAMETER {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer syscall.CloseHandle(syscall.Handle(h))
	var exitCode uint32
	err = windows.GetExitCodeProcess(h, &exitCode)
	return exitCode == STILL_ACTIVE, err
}
