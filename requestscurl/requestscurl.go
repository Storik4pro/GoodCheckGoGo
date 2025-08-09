package requestscurl

import (
	"fmt"
	"goodcheckgogo/checklist"
	"goodcheckgogo/options"
	"goodcheckgogo/strategy"
	"goodcheckgogo/utils"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func CheckConnectivityCurl() (bool, error) {

	keys := options.MyOptions.Curl.BasicKeys
	keys = append(keys, fmt.Sprintf("-m %d", options.MyOptions.ConnTimeout.Value))
	if strategy.IPV == 6 {
		keys = append(keys, "-6")
	} else {
		keys = append(keys, "-4")
	}
	// if strategy.Protocol == "UDP" {
	// 	keys = append(keys, "--http3-only")
	// }
	if options.MyOptions.SkipCertVerify.Value {
		keys = append(keys, "--insecure")
	}
	keys = append(keys, `-w "%{response_code}"`, "-o NUL", options.MyOptions.NetConnTestURL.Value)

	if !options.MyOptions.SkipCertVerify.Value {
		log.Printf("Making normal request to '%s' (Curl)\n", options.MyOptions.NetConnTestURL.Value)
	} else {
		log.Printf("Making insecure request to '%s' (Curl)\n", options.MyOptions.NetConnTestURL.Value)
	}

	cmd := exec.Command(options.MyOptions.Curl.ExecutableFullPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: utils.PrintStringArray(keys)}

	result, _ := cmd.Output()
	if len(result) == 0 {
		return false, fmt.Errorf("no response to read")
	}

	codeN, err := strconv.Atoi(string(result))
	if err != nil {
		return false, fmt.Errorf("can't convert response '%s' to integer: %v", result, err)
	}

	if codeN == 0 {
		log.Println("No connection: response code 0")
		return false, nil
	} else {
		log.Printf("Connection seems ok; response code: %d %s\n", codeN, http.StatusText(codeN))
		return true, nil
	}
}

func DnsLookupCurl(_resolver string, _addrToResolve string) string {
	_addrToResolve = utils.InsensitiveReplace(_addrToResolve, "https://", "")

	keys := options.MyOptions.Curl.BasicKeys
	//keys = append(keys, fmt.Sprintf("-m %d", options.MyOptions.ConnTimeout.Value))
	keys = append(keys, "-m 1")
	if strategy.IPV == 6 {
		keys = append(keys, "-6")
	} else {
		keys = append(keys, "-4")
	}
	if options.MyOptions.SkipCertVerify.Value {
		keys = append(keys, "--insecure")
	}
	if _resolver == "" {
		keys = append(keys, `-w "%{remote_ip}"`, "-o NUL", _addrToResolve)
	} else {
		keys = append(keys, `-w "%{remote_ip}"`, fmt.Sprintf("--doh-url %s", _resolver), "-Z", "-o NUL", _addrToResolve, "-o NUL", "0.0.0.0")
	}

	cmd := exec.Command(options.MyOptions.Curl.ExecutableFullPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: utils.PrintStringArray(keys)}

	result, _ := cmd.Output()
	if len(result) == 0 {
		return ""
	}
	return string(result)
}

func ExtractClusterCurl(mappingURL string) string {
	keys := options.MyOptions.Curl.BasicKeys
	keys = append(keys, fmt.Sprintf("-m %d", options.MyOptions.ConnTimeout.Value))
	if options.MyOptions.SkipCertVerify.Value {
		keys = append(keys, "--insecure")
	}
	keys = append(keys, "-4", mappingURL)

	cmd := exec.Command(options.MyOptions.Curl.ExecutableFullPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: utils.PrintStringArray(keys)}

	result, _ := cmd.Output()
	if len(result) == 0 {
		return ""
	}
	s := strings.Split(string(result), " ")[2]
	return s
}

func FormRequestsKeys(_resolver string, addresses []checklist.Website) []string {
	keys := options.MyOptions.Curl.BasicKeys
	keys = append(keys, fmt.Sprintf("-m %d", options.MyOptions.ConnTimeout.Value))
	if strategy.IPV == 6 {
		keys = append(keys, "-6")
	} else {
		keys = append(keys, "-4")
	}
	if strategy.Proxy != "noproxy" {
		keys = append(keys, fmt.Sprintf("--proxy %s", strategy.Proxy))
	}
	if strategy.Protocol == "UDP" {
		keys = append(keys, "--http3-only")
	}
	if options.MyOptions.SkipCertVerify.Value {
		keys = append(keys, "--insecure")
	}
	// if _resolver != "" {
	// 	keys = append(keys, fmt.Sprintf("--doh-url %s", _resolver))
	// }
	keys = append(keys, `-w "%{urlnum}$%{response_code}@"`)
	//keys = append(keys, `-w "%{urlnum}$%{response_code}$%{errormsg}@"`)
	if len(addresses) > 1 || _resolver != "" {
		keys = append(keys, "-Z", "--parallel-immediate", "--parallel-max 200")
	}
	for _, addr := range addresses {
		if strategy.Proxy == "noproxy" {
			keys = append(keys, addr.Address, "-o NUL", fmt.Sprintf("--resolve %s:443:%s", utils.InsensitiveReplace(addr.Address, "https://", ""), addr.IP))
		} else {
			keys = append(keys, addr.Address, "-o NUL")
		}
	}
	// if _resolver != "" && len(addresses) < 2 {
	// 	keys = append(keys, "0.0.0.0", "-o NUL")
	// }
	return keys
}

func SendRequestsAndParse(keys []string, addresses *[]checklist.Website) error {
	cmd := exec.Command(options.MyOptions.Curl.ExecutableFullPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: utils.PrintStringArray(keys)}

	result, _ := cmd.Output()
	if len(result) == 0 {
		return fmt.Errorf("curl returned no results")
	}

	resultString := string(result)
	//log.Println("resultString:", resultString)
	resultLines := strings.Split(resultString, "@")
	for _, line := range resultLines {
		if line == "$" || line == "" {
			break
		}
		v := strings.Split(line, "$")
		index, err := strconv.Atoi(v[0])
		if err != nil {
			return fmt.Errorf("can't convert urlnum '%s' to integer: %v", v[0], err)
		}
		if index >= len((*addresses)) {
			continue
		}
		code, err := strconv.Atoi(v[1])
		if err != nil {
			return fmt.Errorf("can't convert response code '%s' to integer: %v", v[1], err)
		}
		(*addresses)[index].LastResponseCode = code
		if code != 0 {
			(*addresses)[index].HasSuccesses = true
		}
	}
	log.Println("Responses was received and parsed")
	return nil
}
