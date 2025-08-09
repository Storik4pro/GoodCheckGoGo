package options

import (
	"bufio"
	"fmt"
	"goodcheckgogo/utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Options struct {
	ConnTimeout       optionInt
	InternalTimeoutMs optionInt
	AutoGGC           optionBool
	NetConnTest       optionBool
	NetConnTestURL    optionString
	SkipCertVerify    optionBool

	MappingURLs optionStringArray

	FakeSNI          optionFake
	FakeHexStreamTCP optionFake
	FakeHexStreamUDP optionFake
	FakeHexBytesTCP  optionFake
	FakeHexBytesUDP  optionFake
	PayloadTCP       optionFake
	PayloadUDP       optionFake

	Gdpi   OptionFoolingProgram
	Zapret OptionFoolingProgram
	Ciadpi OptionFoolingProgram

	Curl OptionCurl

	WinDivert optionStringArray

	UseDoH                optionBool
	DoHResolvers          optionStringArray
	ResolverNativeTimeout optionInt
	ResolverNativeRetries optionInt
}

type OptionFoolingProgram struct {
	// real values
	ProgramName    string
	Folder         string
	ExecutableName string
	ServiceNames   []string
	WorksAsProxy   bool
	// names for options in config
	folderInConfig         string
	executableNameInConfig string
	serviceNamesInConfig   string
	// dynamically calculated
	ExecutableFullPath string
	IsExist            bool
}

type OptionCurl struct {
	// real values
	ProgramName    string
	Folder         string
	ExecutableName string
	BasicKeys      []string
	// names for options in config
	folderInConfig         string
	executableNameInConfig string
	basicKeysInConfig      string
	// dynamically calculated
	ExecutableFullPath string
	IsExist            bool
}

type optionFake struct {
	nameInConfig string
	Value        string
	isCustom     bool
	Mask         string
}

type optionStringArray struct {
	nameInConfig string
	Value        []string
	isCustom     bool
}

type optionString struct {
	nameInConfig string
	Value        string
	isCustom     bool
}

type optionInt struct {
	nameInConfig string
	Value        int
	isCustom     bool
}

type optionBool struct {
	nameInConfig string
	Value        bool
	isCustom     bool
}

func initOptionFoolingProgram(_folderInConfig string, _executableNameInConfig string, _serviceNamesInConfig string, _programName string, _executableName string, _serviceNames []string, _worksAsProxy bool) OptionFoolingProgram {
	p := OptionFoolingProgram{
		ProgramName:            _programName,
		Folder:                 "",
		ExecutableName:         _executableName,
		ServiceNames:           _serviceNames,
		WorksAsProxy:           _worksAsProxy,
		folderInConfig:         _folderInConfig,
		executableNameInConfig: _executableNameInConfig,
		serviceNamesInConfig:   _serviceNamesInConfig,
		ExecutableFullPath:     "unknown",
		IsExist:                false,
	}
	return p
}

func initOptionCurl() OptionCurl {
	c := OptionCurl{
		ProgramName:            "Curl",
		Folder:                 "Curl",
		ExecutableName:         "curl.exe",
		BasicKeys:              []string{"-s"},
		folderInConfig:         "CurlFolder",
		executableNameInConfig: "CurlExecutableName",
		basicKeysInConfig:      "CurlCustomKeys",
		ExecutableFullPath:     "unknown",
		IsExist:                false,
	}
	return c
}

func initOptionFake(_nameInConfig string, _value string, _mask string) optionFake {
	o := optionFake{
		nameInConfig: _nameInConfig,
		Value:        _value,
		isCustom:     false,
		Mask:         _mask,
	}
	return o
}

func initOptionStringArray(_nameInConfig string, _value []string) optionStringArray {
	o := optionStringArray{
		nameInConfig: _nameInConfig,
		Value:        _value,
		isCustom:     false,
	}
	return o
}

func initOptionString(_nameInConfig string, _value string) optionString {
	o := optionString{
		nameInConfig: _nameInConfig,
		Value:        _value,
		isCustom:     false,
	}
	return o
}

func initOptionInt(_nameInConfig string, _value int) optionInt {
	o := optionInt{
		nameInConfig: _nameInConfig,
		Value:        _value,
		isCustom:     false,
	}
	return o
}

func initOptionBool(_nameInConfig string, _value bool) optionBool {
	o := optionBool{
		nameInConfig: _nameInConfig,
		Value:        _value,
		isCustom:     false,
	}
	return o
}

var MyOptions = Options{
	ConnTimeout:       initOptionInt("ConnectionTimeout", 2),
	InternalTimeoutMs: initOptionInt("InternalTimeoutMs", 100),
	AutoGGC:           initOptionBool("AutomaticGoogleCacheTest", true),
	NetConnTest:       initOptionBool("AutomaticConnectivityTest", true),
	NetConnTestURL:    initOptionString("ConnectivityTestURL", "https://www.w3.org"),
	SkipCertVerify:    initOptionBool("SkipCertVerify", false),

	MappingURLs: initOptionStringArray("GoogleCacheMappingURLs", []string{"https://redirector.gvt1.com/report_mapping?di=no", "https://redirector.googlevideo.com/report_mapping?di=no"}),

	FakeHexStreamTCP: initOptionFake("FakeHexStreamTCP", "1603030135010001310303424143facf5c983ac8ff20b819cfd634cbf5143c0005b2b8b142a6cd335012c220008969b6b387683dedb4114d466ca90be3212b2bde0c4f56261a9801", "FAKEHEXSTREAMTCP"),
	FakeHexStreamUDP: initOptionFake("FakeHexStreamUDP", "c200000001142ee3e35f6bbb23a8e65da97821cfc2724c8fc45e14232336e9c50386557b4c7aa7e19f321903124424008000047c0dfcfa1dcd73ba2a9093b3eef743c585daff453c02305bbae9437cc89b07f6bcf4dd447ab0c6903c0049ef59a8e418f8e091d371f7257b180f85d878484e63ea2306f35e445701d95ae90c70bb372f25d683efa453f174105f07", "FAKEHEXSTREAMUDP"),
	FakeHexBytesTCP:  initOptionFake("FakeHexBytesTCP", `:\x16\x03\x03\x01\x3b\x01\x00\x01\x37\x03\x03\xad\xe3\x84\x4a\xd1\x64\xc3\x78\xdd\xe2\x42\xb6\x7a\x17\x74\xe6\x4b\xc0\x2a\xcb\x4a\x2f\x74\x74\x23\xf0\x43\x8d\x61\x2a\x7b\x10\x20\x43\x7d\xae\x47\x24\xba\x27\xfe\x70\x27\x80\x75\xd1\xa5\x33\x60\x29\x78\xb2\xca\xe7\x3e\x19\xb6\x87\x8d\xe5\x38\xdf\xab\x1b\x2b\x00\x5c\x13\x02\x13\x03\x13\x01\xc0\x30\xc0\x2c\xc0\x28\xc0\x24\xc0\x14\xc0\x0a\x00\x9f\x00\x6b\x00\x39\xcc\xa9\xcc\xa8\xcc\xaa\x00\xc4\x00\x88\x00\x9d\x00\x3d\x00\x35\x00\xc0\x00\x84\xc0\x2f\xc0\x2b\xc0\x27\xc0\x23\xc0\x13\xc0\x09\x00\x9e\x00\x67\x00\x33\x00\xbe\x00\x45\x00\x9c\x00\x3c\x00\x2f\x00\xba\x00\x41\xc0\x11\xc0\x07\x00\x05\xc0\x12\xc0\x08\x00\x16\x00\x0a\x00\xff\x01\x00\x00\x92\x00\x0a\x00\x0a\x00\x08\x00\x1d\x00\x17\x00\x18\x00\x19\x00\x00\x00\x19\x00\x17\x00\x00\x14\x74\x72\x61\x6e\x73\x6c\x61\x74\x65\x2e\x67\x6f\x6f\x67\x6c\x65\x2e\x63\x6f\x6d\x00\x0b\x00\x02\x01\x00\x00\x10\x00\x0e\x00\x0c\x02\x68\x32\x08\x68\x74\x74\x70\x2f\x31\x2e\x31\x00\x0d\x00\x18\x00\x16\x08\x06\x06\x01\x06\x03\x08\x05\x05\x01\x05\x03\x08\x04\x04\x01\x04\x03\x02\x01\x02\x03\x00\x2b\x00\x05\x04\x03\x04\x03\x03\x00\x33\x00\x26\x00\x24\x00\x1d\x00\x20\x43\xc9\xea\x84\x67\x5a\x9f\xcb\x6f\x02\xb9\x78\x44\x1e\xa9\x07\x77\xbd\xcb\x62\xdc\x87\x23\x3b\x1c\xae\x71\x19\xa3\xa6\x80\x0d`, "FAKEHEXBYTESTCP"),
	FakeHexBytesUDP:  initOptionFake("FakeHexBytesUDP", `:\xc2\x00\x00\x00\x01\x14\x2e\xe3\xe3\x5f\x6b\xbb\x23\xa8\xe6\x5d\xa9\x78\x21\xcf\xc2\x72\x4c\x8f\xc4\x5e\x14\x23\x23\x36\xe9\xc5\x03\x86\x55\x7b\x4c\x7a\xa7\xe1\x9f\x32\x19\x03\x12\x44\x24\x00\x80\x00\x04\x7c\x0d\xfc\xfa\x1d\xcd\x73\xba\x2a\x90\x93\xb3\xee\xf7\x43\xc5\x85\xda\xff\x45\x3c\x02\x30\x5b\xba\xe9\x43\x7c\xc8\x9b\x07\xf6\xbc\xf4\xdd\x44\x7a\xb0\xc6\x90\x3c\x00\x49\xef\x59\xa8\xe4\x18\xf8\xe0\x91\xd3\x71\xf7\x25\x7b\x18\x0f\x85\xd8\x78\x48\x4e\x63\xea\x23\x06\xf3\x5e\x44\x57\x01\xd9\x5a\xe9\x0c\x70\xbb\x37\x2f\x25\xd6\x83\xef\xa4\x53\xf1\x74\x10\x5f\x07`, "FAKEHEXBYTESUDP"),
	PayloadTCP:       initOptionFake("PayloadTCP", `Payloads\default_tcp.bin`, "PAYLOADTCP"),
	PayloadUDP:       initOptionFake("PayloadUDP", `Payloads\default_udp.bin`, "PAYLOADUDP"),
	FakeSNI:          initOptionFake("FakeSNI", "www.google.com", "FAKESNI"),

	Curl: initOptionCurl(),

	Gdpi:   initOptionFoolingProgram("GoodbyeDPIFolder", "GoodbyeDPIExecutableName", "GoodbyeDPIServiceNames", "GoodbyeDPI", "goodbyedpi.exe", []string{"GoodbyeDPI", "goodbyedpi"}, false),
	Zapret: initOptionFoolingProgram("ZapretFolder", "ZapretExecutableName", "ZapretServiceNames", "Zapret", "winws.exe", []string{"winws", "winws1", "winws2", "Zapret", "zapret"}, false),
	Ciadpi: initOptionFoolingProgram("ByeDPIFolder", "ByeDPIExecutableName", "ByeDPIServiceNames", "ByeDPI", "ciadpi.exe", []string{"ciadpi", "ByeDPI", "byedpi"}, true),

	WinDivert: initOptionStringArray("WinDivertServiceNames", []string{"WinDivert", "WinDivert14"}),

	UseDoH:                initOptionBool("UseDoH", true),
	DoHResolvers:          initOptionStringArray("DoHResolvers", []string{"https://dns.comss.one/dns-query", "https://one.one.one.one/dns-query", "https://1.1.1.2/dns-query", "https://dns.google/dns-query", "https://mozilla.cloudflare-dns.com/dns-query", "https://dns10.quad9.net/dns-query", "https://dns.controld.com/comss", "https://freedns.controld.com/p0"}),
	ResolverNativeTimeout: initOptionInt("ResolverNativeTimeout", 2),
	ResolverNativeRetries: initOptionInt("ResolverNativeRetries", 2),
}

var configFile string

func ParseConfig(configfile string) error {
	configFile = configfile

	// setting options
	readConfigInt(&MyOptions.ConnTimeout)
	readConfigInt(&MyOptions.InternalTimeoutMs)
	readConfigBool(&MyOptions.NetConnTest)
	readConfigString(&MyOptions.NetConnTestURL)
	readConfigBool(&MyOptions.SkipCertVerify)

	readConfigBool(&MyOptions.AutoGGC)
	readConfigStringArray(&MyOptions.MappingURLs)

	readConfigBool(&MyOptions.UseDoH)
	readConfigStringArray(&MyOptions.DoHResolvers)
	readConfigInt(&MyOptions.ResolverNativeTimeout)
	readConfigInt(&MyOptions.ResolverNativeRetries)

	readConfigFake(&MyOptions.FakeSNI)
	readConfigFake(&MyOptions.FakeHexStreamTCP)
	readConfigFake(&MyOptions.FakeHexStreamUDP)
	readConfigFake(&MyOptions.FakeHexBytesTCP)
	readConfigFake(&MyOptions.FakeHexBytesUDP)

	readConfigFake(&MyOptions.PayloadTCP)
	if _, err := os.Stat(MyOptions.PayloadTCP.Value); err != nil {
		return fmt.Errorf("can't read payload file '%s': %v", MyOptions.PayloadTCP.Value, err)
	} else {
		log.Printf("Payload '%s' is in place\n", MyOptions.PayloadTCP.Value)
	}
	readConfigFake(&MyOptions.PayloadUDP)
	if _, err := os.Stat(MyOptions.PayloadUDP.Value); err != nil {
		return fmt.Errorf("can't read payload file '%s': %v", MyOptions.PayloadUDP.Value, err)
	} else {
		log.Printf("Payload '%s' is in place\n", MyOptions.PayloadTCP.Value)
	}

	err := setCurl()
	if err != nil {
		return fmt.Errorf("can't set up '%s': %v", MyOptions.Curl.ProgramName, err)
	}

	err = setFoolingPrograms()
	if err != nil {
		return fmt.Errorf("can't set up fooling programs: %v", err)
	}

	readConfigStringArray(&MyOptions.WinDivert)

	return nil
}

func openConfig() (*os.File, error) {
	var err error
	c, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("can't open a file: %v", err)
	}
	return c, nil
}

func readConfigFake(_optionFake *optionFake) error {
	c, err := openConfig()
	if err != nil {
		return fmt.Errorf("can't open config file: %v", err)
	}
	defer c.Close()
	scan := bufio.NewScanner(c)
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "[") && !utils.IsCommented(scan.Text(), "/") {
			param := strings.SplitN(scan.Text(), `=`, 2)
			if param[0] == _optionFake.nameInConfig {
				if param[1] != "" {
					_optionFake.isCustom = true
					_optionFake.Value = param[1]
					log.Printf("Set option '%s': '%s'\n", _optionFake.nameInConfig, _optionFake.Value)
					return nil
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", _optionFake.nameInConfig, _optionFake.Value)
					return nil
				}
			}
		}
	}
	log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", _optionFake.nameInConfig, _optionFake.Value)
	return nil
}

func readConfigString(_optionString *optionString) error {
	c, err := openConfig()
	if err != nil {
		return fmt.Errorf("can't open config file: %v", err)
	}
	defer c.Close()
	scan := bufio.NewScanner(c)
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "[") && !utils.IsCommented(scan.Text(), "/") {
			param := strings.SplitN(scan.Text(), `=`, 2)
			if param[0] == _optionString.nameInConfig {
				if param[1] != "" {
					_optionString.isCustom = true
					_optionString.Value = param[1]
					log.Printf("Set option '%s': '%s'\n", _optionString.nameInConfig, _optionString.Value)
					return nil
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", _optionString.nameInConfig, _optionString.Value)
					return nil
				}
			}
		}
	}
	log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", _optionString.nameInConfig, _optionString.Value)
	return nil
}

func readConfigInt(_optionInt *optionInt) error {
	c, err := openConfig()
	if err != nil {
		return fmt.Errorf("can't open config file: %v", err)
	}
	defer c.Close()
	scan := bufio.NewScanner(c)
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "[") && !utils.IsCommented(scan.Text(), "/") {
			param := strings.SplitN(scan.Text(), `=`, 2)
			if param[0] == _optionInt.nameInConfig {
				if param[1] != "" {
					v, err := strconv.Atoi(param[1])
					if err != nil {
						log.Printf("Can't set option '%s': can't convert to integer: %v; using defaults: '%d'\n", _optionInt.nameInConfig, err, _optionInt.Value)
						return nil
					}
					if v < 0 {
						log.Printf("Can't set option '%s': shouldn't be lesser than 0; using defaults: '%d'\n", _optionInt.nameInConfig, _optionInt.Value)
						return nil
					}
					_optionInt.isCustom = true
					_optionInt.Value = v
					log.Printf("Set option '%s': '%d'\n", _optionInt.nameInConfig, _optionInt.Value)
					return nil
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%d'\n", _optionInt.nameInConfig, _optionInt.Value)
					return nil
				}
			}
		}
	}
	log.Printf("Can't set option '%s': not found in config; using defaults: '%d'\n", _optionInt.nameInConfig, _optionInt.Value)
	return nil
}

func readConfigBool(_optionBool *optionBool) error {
	c, err := openConfig()
	if err != nil {
		return fmt.Errorf("can't open config file: %v", err)
	}
	defer c.Close()
	scan := bufio.NewScanner(c)
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "[") && !utils.IsCommented(scan.Text(), "/") {
			param := strings.SplitN(scan.Text(), `=`, 2)
			if param[0] == _optionBool.nameInConfig {
				if param[1] != "" {
					v, err := strconv.ParseBool(param[1])
					if err != nil {
						log.Printf("Can't set option '%s': can't convert to boolean: %v; using defaults: '%t'\n", _optionBool.nameInConfig, err, _optionBool.Value)
						return nil
					}
					_optionBool.isCustom = true
					_optionBool.Value = v
					log.Printf("Set option '%s': '%t'\n", _optionBool.nameInConfig, _optionBool.Value)
					return nil
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%t'\n", _optionBool.nameInConfig, _optionBool.Value)
					return nil
				}
			}
		}
	}
	log.Printf("Can't set option '%s': not found in config; using defaults: '%t'\n", _optionBool.nameInConfig, _optionBool.Value)
	return nil
}

func setFoolingPrograms() error {
	currentDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't return the path to the current directory: %v", err)
	}

	//gdpi
	gdpiSubfolder := "x86"
	if utils.Is64bit() {
		gdpiSubfolder = "x86_64"
	}
	err = readConfigFoolingProgram(&MyOptions.Gdpi)
	if err != nil {
		return fmt.Errorf("can't set fooling program '%s': %v", MyOptions.Gdpi.ProgramName, err)
	}
	lookForFoolingProgram(&MyOptions.Gdpi, MyOptions.Gdpi.Folder)
	if !MyOptions.Gdpi.IsExist {
		lookForFoolingProgram(&MyOptions.Gdpi, filepath.Join(MyOptions.Gdpi.Folder, gdpiSubfolder))
	}
	if !MyOptions.Gdpi.IsExist {
		lookForFoolingProgram(&MyOptions.Gdpi, currentDirectory)
	}
	if !MyOptions.Gdpi.IsExist {
		lookForFoolingProgram(&MyOptions.Gdpi, filepath.Join(currentDirectory, gdpiSubfolder))
	}
	if !MyOptions.Gdpi.IsExist {
		log.Printf("Can't find '%s' anywhere\n", MyOptions.Gdpi.ProgramName)
	}

	//zapret
	zapretSubfolder := "zapret-winws"
	err = readConfigFoolingProgram(&MyOptions.Zapret)
	if err != nil {
		return fmt.Errorf("can't set fooling program '%s': %v", MyOptions.Zapret.ProgramName, err)
	}
	lookForFoolingProgram(&MyOptions.Zapret, MyOptions.Zapret.Folder)
	if !MyOptions.Zapret.IsExist {
		lookForFoolingProgram(&MyOptions.Zapret, filepath.Join(MyOptions.Zapret.Folder, zapretSubfolder))
	}
	if !MyOptions.Zapret.IsExist {
		lookForFoolingProgram(&MyOptions.Zapret, currentDirectory)
	}
	if !MyOptions.Zapret.IsExist {
		lookForFoolingProgram(&MyOptions.Zapret, filepath.Join(currentDirectory, zapretSubfolder))
	}
	if !MyOptions.Zapret.IsExist {
		log.Printf("Can't find '%s' anywhere\n", MyOptions.Zapret.ProgramName)
	}

	//ciadpi
	err = readConfigFoolingProgram(&MyOptions.Ciadpi)
	if err != nil {
		return fmt.Errorf("can't set fooling program '%s': %v", MyOptions.Ciadpi.ProgramName, err)
	}
	lookForFoolingProgram(&MyOptions.Ciadpi, MyOptions.Ciadpi.Folder)
	if !MyOptions.Ciadpi.IsExist {
		lookForFoolingProgram(&MyOptions.Ciadpi, currentDirectory)
	}
	if !MyOptions.Ciadpi.IsExist {
		log.Printf("Can't find '%s' anywhere\n", MyOptions.Ciadpi.ProgramName)
	}
	if MyOptions.Ciadpi.IsExist && !MyOptions.Curl.IsExist {
		MyOptions.Ciadpi.IsExist = false
		log.Printf("Fooling program '%s' working as a proxy, but '%s' which is required for testing in proxy-mode aren't found; '%s' will be disabled\n", MyOptions.Ciadpi.ProgramName, MyOptions.Curl.ProgramName, MyOptions.Ciadpi.ProgramName)
	}

	if !MyOptions.Gdpi.IsExist && !MyOptions.Zapret.IsExist && !MyOptions.Ciadpi.IsExist {
		return fmt.Errorf("can't find a single fooling program")
	}

	return nil
}

func readConfigFoolingProgram(_optionFoolingProgram *OptionFoolingProgram) error {
	c, err := openConfig()
	if err != nil {
		return fmt.Errorf("can't open config file: %v", err)
	}
	defer c.Close()
	scan := bufio.NewScanner(c)
	folderIsSet, exeIsSet, snameIsSet := false, false, false
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "[") && !utils.IsCommented(scan.Text(), "/") {
			param := strings.SplitN(scan.Text(), `=`, 2)
			switch param[0] {
			case _optionFoolingProgram.folderInConfig:
				if param[1] != "" {
					_optionFoolingProgram.Folder = param[1]
					folderIsSet = true
					log.Printf("Set option '%s': '%s'\n", _optionFoolingProgram.folderInConfig, _optionFoolingProgram.Folder)
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", _optionFoolingProgram.folderInConfig, _optionFoolingProgram.Folder)
				}
			case _optionFoolingProgram.executableNameInConfig:
				if param[1] != "" {
					_optionFoolingProgram.ExecutableName = param[1]
					exeIsSet = true
					log.Printf("Set option '%s': '%s'\n", _optionFoolingProgram.executableNameInConfig, _optionFoolingProgram.ExecutableName)
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", _optionFoolingProgram.executableNameInConfig, _optionFoolingProgram.ExecutableName)
				}
			case _optionFoolingProgram.serviceNamesInConfig:
				if param[1] != "" {
					_optionFoolingProgram.ServiceNames = strings.Split(param[1], ";")
					snameIsSet = true
					log.Printf("Set option '%s': '%s'\n", _optionFoolingProgram.serviceNamesInConfig, _optionFoolingProgram.ServiceNames)
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", _optionFoolingProgram.serviceNamesInConfig, _optionFoolingProgram.ServiceNames)
				}
			}
		}
	}
	if !folderIsSet {
		log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", _optionFoolingProgram.folderInConfig, _optionFoolingProgram.Folder)
	}
	if !exeIsSet {
		log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", _optionFoolingProgram.executableNameInConfig, _optionFoolingProgram.ExecutableName)
	}
	if !snameIsSet {
		log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", _optionFoolingProgram.serviceNamesInConfig, _optionFoolingProgram.ServiceNames)
	}
	return nil
}

func lookForFoolingProgram(_program *OptionFoolingProgram, _folder string) {
	fullPath := filepath.Join(_folder, _program.ExecutableName)
	if _, err := os.Stat(fullPath); err == nil {
		_program.IsExist = true
		_program.ExecutableFullPath = fullPath
		log.Printf("'%s' executable '%s' found at '%s'\n", _program.ProgramName, _program.ExecutableName, _folder)
	} else {
		log.Printf("Can't find '%s' executable '%s' at '%s'\n", _program.ProgramName, _program.ExecutableName, _folder)
	}
}

func setCurl() error {
	c, err := openConfig()
	if err != nil {
		return fmt.Errorf("can't open config file: %v", err)
	}
	defer c.Close()
	folderIsSet, exeIsSet, keysIsSet := false, false, false
	scan := bufio.NewScanner(c)
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "[") && !utils.IsCommented(scan.Text(), "/") {
			param := strings.SplitN(scan.Text(), `=`, 2)
			switch param[0] {
			case MyOptions.Curl.folderInConfig:
				if param[1] != "" {
					folderIsSet = true
					MyOptions.Curl.Folder = param[1]
					log.Printf("Set option '%s': '%s'\n", MyOptions.Curl.folderInConfig, MyOptions.Curl.Folder)
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", MyOptions.Curl.folderInConfig, MyOptions.Curl.Folder)
				}
			case MyOptions.Curl.executableNameInConfig:
				if param[1] != "" {
					exeIsSet = true
					MyOptions.Curl.ExecutableName = param[1]
					log.Printf("Set option '%s': '%s'\n", MyOptions.Curl.executableNameInConfig, MyOptions.Curl.ExecutableName)
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", MyOptions.Curl.executableNameInConfig, MyOptions.Curl.ExecutableName)
				}
			case MyOptions.Curl.basicKeysInConfig:
				if param[1] != "" {
					keysIsSet = true
					MyOptions.Curl.BasicKeys[0] = param[1]
					log.Printf("Set option '%s': '%s'\n", MyOptions.Curl.basicKeysInConfig, MyOptions.Curl.BasicKeys[0])
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", MyOptions.Curl.basicKeysInConfig, MyOptions.Curl.BasicKeys[0])
				}
			}
		}
	}
	if !folderIsSet {
		log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", MyOptions.Curl.folderInConfig, MyOptions.Curl.Folder)
	}
	if !exeIsSet {
		log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", MyOptions.Curl.executableNameInConfig, MyOptions.Curl.ExecutableName)
	}
	if !keysIsSet {
		log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", MyOptions.Curl.basicKeysInConfig, MyOptions.Curl.BasicKeys[0])
	}

	currentDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't return the path to the current directory: %v", err)
	}
	curlSubfolder := "x86"
	if utils.Is64bit() {
		curlSubfolder = "x86_64"
	}
	lookForCurl(MyOptions.Curl.Folder)
	if !MyOptions.Curl.IsExist {
		lookForCurl(filepath.Join(MyOptions.Curl.Folder, curlSubfolder))
	}
	if !MyOptions.Curl.IsExist {
		lookForCurl(currentDirectory)
	}
	if !MyOptions.Curl.IsExist {
		lookForCurl(filepath.Join(currentDirectory, curlSubfolder))
	}
	if !MyOptions.Curl.IsExist {
		curlFromEnv := ""
		envPaths := strings.Split(os.Getenv("Path"), ";")
		for _, envPath := range envPaths {
			if strings.Contains(envPath, "curl") {
				curlFromEnv = envPath
				break
			}
			if strings.Contains(envPath, "cURL") {
				curlFromEnv = envPath
				break
			}
			if strings.Contains(envPath, "Curl") {
				curlFromEnv = envPath
				break
			}
			if strings.Contains(envPath, "CURL") {
				curlFromEnv = envPath
				break
			}
		}
		lookForCurl(curlFromEnv)
	}
	if !MyOptions.Curl.IsExist {
		lookForCurl(filepath.Join(os.Getenv("SystemRoot"), "System32"))
	}

	if !MyOptions.Curl.IsExist {
		log.Printf("Can't find '%s' anywhere\nDownload it at 'https://curl.se/' and put the content of '/bin' folder next to this program.\n", MyOptions.Curl.ProgramName)
	} else {
		err := printCurlInfo()
		if err != nil {
			return fmt.Errorf("can't read curl version: %v", err)
		}
	}

	return nil
}

func printCurlInfo() error {
	cmd := exec.Command(MyOptions.Curl.ExecutableFullPath, "-V")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("can't pipe cmd: %v", err)
	}
	defer out.Close()
	r := bufio.NewReader(out)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("can't call cmd: %v", err)
	}
	scan := bufio.NewScanner(r)
	log.Println("------------------")
	for scan.Scan() {
		log.Println(scan.Text())
	}
	log.Println("------------------")
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("can't properly close pipe: %v", err)
	}
	return nil
}

func lookForCurl(_folder string) {
	fullPath := filepath.Join(_folder, MyOptions.Curl.ExecutableName)
	if _, err := os.Stat(fullPath); err == nil {
		MyOptions.Curl.IsExist = true
		MyOptions.Curl.ExecutableFullPath = fullPath
		log.Printf("'%s' executable '%s' found at '%s'\n", MyOptions.Curl.ProgramName, MyOptions.Curl.ExecutableName, _folder)
	} else {
		log.Printf("Can't find '%s' executable '%s' at '%s'\n", MyOptions.Curl.ProgramName, MyOptions.Curl.ExecutableName, _folder)
	}
}

func readConfigStringArray(_optionStringArray *optionStringArray) error {
	c, err := openConfig()
	if err != nil {
		return fmt.Errorf("can't open config file: %v", err)
	}
	defer c.Close()
	scan := bufio.NewScanner(c)
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "[") && !utils.IsCommented(scan.Text(), "/") {
			param := strings.SplitN(scan.Text(), `=`, 2)
			if param[0] == _optionStringArray.nameInConfig {
				if param[1] != "" {
					_optionStringArray.isCustom = true
					_optionStringArray.Value = strings.Split(param[1], ";")
					log.Printf("Set option '%s': '%s'\n", _optionStringArray.nameInConfig, _optionStringArray.Value)
					return nil
				} else {
					log.Printf("Can't set option '%s': empty value; using defaults: '%s'\n", _optionStringArray.nameInConfig, _optionStringArray.Value)
					return nil
				}
			}
		}
	}
	log.Printf("Can't set option '%s': not found in config; using defaults: '%s'\n", _optionStringArray.nameInConfig, _optionStringArray.Value)
	return nil
}
