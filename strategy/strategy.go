package strategy

import (
	"bufio"
	"fmt"
	"goodcheckgogo/options"
	"goodcheckgogo/utils"
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type Strategy struct {
	Keys         []string
	KeysSorted   []string
	IsValid      bool
	Successes    int
	IsTested     bool
	HasSuccesses bool
}

type keySet struct {
	keys []string
}

var Protocol string = "unset"
var IPV int = -1
var Proxy string = "unset"
var ProtoFull string = "unset"
var keySets []keySet

func NewStrategy() Strategy {
	s := Strategy{
		Keys:         nil,
		KeysSorted:   nil,
		IsValid:      true,
		Successes:    -1,
		IsTested:     false,
		HasSuccesses: false,
	}
	return s
}

func newKeySet(_keys []string) keySet {
	k := keySet{
		keys: _keys,
	}
	return k
}

func ReadStrategies(file string) ([]Strategy, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("can't open a file '%s': %v", file, err)
	}
	defer f.Close()

	var s []Strategy

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "/") {
			line := scan.Text()
			log.Println("Reading line:", line)
			if strings.Contains(line, "#PROTO=") {
				err = parseProtocol(line)
				if err != nil {
					return nil, fmt.Errorf("can't parse a line with protocol settings '%s': %v", line, err)
				}
				continue
			}
			if strings.Contains(line, "#IPV=") {
				err = parseIPV(line)
				if err != nil {
					return nil, fmt.Errorf("can't parse a line with IP version settings '%s': %v", line, err)
				}
				continue
			}
			if strings.Contains(line, "#PROXY=") {
				err = parseProxy(line)
				if err != nil {
					return nil, fmt.Errorf("can't parse a line with proxy settings '%s': %v", line, err)
				}
				continue
			}
			if strings.Contains(line, "#KEY#") {
				err := parseKey(line)
				if err != nil {
					return nil, fmt.Errorf("can't parse a line with strategy keys '%s': %v", line, err)
				}
				continue
			}
			if strings.Contains(line, "#ENDGROUP#") {
				log.Println("Group ended, forming strategies from key sets...")
				if len(keySets) == 0 {
					return nil, fmt.Errorf("no key sets found, use '#KEY#' to set them")
				}
				k, err := formStrategies()
				if err != nil {
					return nil, fmt.Errorf("can't process key sets: %v", err)
				}
				s = append(s, k...)
				keySets = nil
				continue
			}
			return nil, fmt.Errorf("can't parse a line '%s': unexpected content", line)
		}
	}

	if Protocol == "unset" {
		return nil, fmt.Errorf("protocol is undefined; use '#PROTO=' to set it")
	}
	if IPV == -1 {
		IPV = 4
		log.Println("IP version is undefined, assuming IPv4")
	}
	if Proxy == "unset" {
		Proxy = "noproxy"
		log.Println("Proxy is undefined, assuming no-proxy")
	}
	if len(s) == 0 {
		return nil, fmt.Errorf("no strategies found")
	} else {
		log.Printf("Total strategies formed from the list: %d\n", len(s))
	}

	switch Protocol {
	case "TCP":
		switch IPV {
		case 4:
			ProtoFull = "tcp4"
		case 6:
			ProtoFull = "tcp6"
		default:
			return nil, fmt.Errorf("schrodinger's cat: 'IPV' value is out of bounds: '%d'", IPV)
		}
	case "UDP":
		switch IPV {
		case 4:
			ProtoFull = "udp4"
		case 6:
			ProtoFull = "udp6"
		default:
			return nil, fmt.Errorf("schrodinger's cat: 'IPV' value is out of bounds: '%d'", IPV)
		}
	default:
		return nil, fmt.Errorf("schrodinger's cat: 'Protocol' value is out of bounds: '%s'", Protocol)
	}

	return s, nil
}

func formStrategies() ([]Strategy, error) {

	var strats []Strategy

	total := 1
	for _, keySet := range keySets {
		total = total * len(keySet.keys)
	}

	for i := 0; i < total; i++ {
		strats = append(strats, NewStrategy())
	}

	previousSteps := 1
	for _, keySet := range keySets {
		var err error
		strats, previousSteps, err = keysStep(strats, keySet.keys, previousSteps, total)
		if err != nil {
			return nil, fmt.Errorf("can't form strategy line: %v", err)
		}
	}

	for i := 0; i < total; i++ {
		var processed []string
		for _, key := range strats[i].Keys {
			replacer := strings.NewReplacer(
				options.MyOptions.FakeSNI.Mask, options.MyOptions.FakeSNI.Value,
				options.MyOptions.FakeHexStreamTCP.Mask, options.MyOptions.FakeHexStreamTCP.Value,
				options.MyOptions.FakeHexStreamUDP.Mask, options.MyOptions.FakeHexStreamUDP.Value,
				options.MyOptions.FakeHexBytesTCP.Mask, options.MyOptions.FakeHexBytesTCP.Value,
				options.MyOptions.FakeHexBytesUDP.Mask, options.MyOptions.FakeHexBytesUDP.Value,
				options.MyOptions.PayloadTCP.Mask, options.MyOptions.PayloadTCP.Value,
				options.MyOptions.PayloadUDP.Mask, options.MyOptions.PayloadUDP.Value,
			)
			key = replacer.Replace(key)
			processed = append(processed, strings.Split(key, "&")...)
		}
		for j := 0; j < len(processed)-1; j++ {
			if processed[j] == "empty" {
				continue
			}
			for k := j + 1; k < len(processed); k++ {
				if processed[j] == processed[k] {
					processed[k] = "empty"
				}
			}
		}
		for j := len(processed) - 1; j >= 0; j-- {
			if processed[j] == "empty" {
				processed = utils.RemoveFromArrayString(processed, j)
			}
		}
		if processed == nil {
			strats[i].IsValid = false
			continue
		}
		strats[i].Keys = processed
	}
	for i := 0; i < total; i++ {
		strats[i].KeysSorted = append(strats[i].KeysSorted, strats[i].Keys...)
		sort.Strings(strats[i].KeysSorted)
	}
	for i := 0; i < total-1; i++ {
		if strats[i].IsValid {
			for j := i + 1; j < total; j++ {
				if strats[j].IsValid && reflect.DeepEqual(strats[i].KeysSorted, strats[j].KeysSorted) {
					strats[j].IsValid = false
				}
			}
		}
	}
	var stratsValid []Strategy
	for i := 0; i < total; i++ {
		if strats[i].IsValid {
			stratsValid = append(stratsValid, NewStrategy())
			stratsValid[len(stratsValid)-1].Keys = strats[i].Keys
			log.Printf("Formed strategy %d: %s\n", len(stratsValid), stratsValid[len(stratsValid)-1].Keys)
		}
	}

	return stratsValid, nil
}

func keysStep(strategies []Strategy, keys []string, previousSteps int, totalStrategies int) ([]Strategy, int, error) {
	if len(keys) == 0 {
		return nil, previousSteps, fmt.Errorf("keys array length is zero")
	}
	if len(strategies) == 0 {
		return nil, previousSteps, fmt.Errorf("strategies array length is zero")
	}
	if previousSteps == 0 {
		return nil, previousSteps, fmt.Errorf("steps in cycle is zero")
	}
	if totalStrategies == 0 {
		return nil, previousSteps, fmt.Errorf("total strategies number is zero")
	}
	if len(strategies) != totalStrategies {
		return nil, previousSteps, fmt.Errorf("total strategies number isn'n equal to an actual array length")
	}

	currentSteps := previousSteps * len(keys)
	stepLength := totalStrategies / currentSteps

	for i := 0; i < currentSteps; i++ {
		n := i % len(keys)
		for j := 0; j < stepLength; j++ {
			strategies[j+i*stepLength].Keys = append(strategies[j+i*stepLength].Keys, keys[n])
		}
	}
	return strategies, currentSteps, nil
}

func parseKey(l string) error {
	s := strings.Split(l, "#")
	if len(s) < 2 {
		return fmt.Errorf("keys value is undefined")
	}
	if s[2] == "" {
		return fmt.Errorf("keys value is empty")
	}
	ss, err := parseKeysSet(s[2])
	if err != nil {
		return fmt.Errorf("can't parse keys from a line: %v", err)
	}
	keySets = append(keySets, newKeySet(ss))
	log.Println("Key set found:", ss)
	return nil
}

func parseKeysSet(l string) ([]string, error) {
	if l == "" {
		return nil, fmt.Errorf("nothing to parse: value is empty")
	}
	s := strings.Split(l, ";")
	var ss []string
	for _, value := range s {
		if value != "" {
			ss = append(ss, value)
		}
	}
	if len(ss) == 0 {
		return nil, fmt.Errorf("no keys found")
	}
	return ss, nil
}

func parseProxy(l string) error {
	if Proxy != "unset" {
		return fmt.Errorf("proxy value was already set")
	}
	v := strings.Split(l, "=")
	if len(v) == 0 {
		Proxy = "noproxy"
		log.Println("Proxy value is undefined, assuming no-proxy")
		return nil
	}
	switch v[1] {
	case "":
		Proxy = "noproxy"
		log.Println("Proxy value is empty, assuming no-proxy")
		return nil
	default:
		Proxy = v[1]
		log.Println("Setting proxy as:", Proxy)
		if !options.MyOptions.Curl.IsExist {
			return fmt.Errorf("proxy setting is found, but '%s' which is required for testing in proxy-mode aren't", options.MyOptions.Curl.ProgramName)
		}
	}
	return nil
}

func parseIPV(l string) error {
	if IPV != -1 {
		return fmt.Errorf("IP version was already set")
	}
	v := strings.Split(l, "=")
	if len(v) == 0 {
		IPV = 4
		log.Println("IP version value is undefined, assuming IPv4")
		return nil
	}
	if v[1] == "" {
		IPV = 4
		log.Println("IP version value is empty, assuming IPv4")
		return nil
	}
	i, err := strconv.Atoi(v[1])
	if err != nil {
		return fmt.Errorf("can't convert IP version value to integer: %v", err)
	}
	if i != 4 && i != 6 {
		return fmt.Errorf("incorrect IP version '%d': expected 4 or 6", i)
	}
	IPV = i
	log.Println("Found IP version:", IPV)
	return nil
}

func parseProtocol(l string) error {
	if Protocol != "unset" {
		return fmt.Errorf("protocol value was already set")
	}
	v := strings.Split(l, "=")
	if len(v) == 0 {
		return fmt.Errorf("protocol value is undefined")
	}
	switch v[1] {
	case "":
		return fmt.Errorf("protocol value can't be empty")
	case "TCP":
		Protocol = v[1]
		log.Println("Found protocol value:", Protocol)
	case "UDP":
		Protocol = v[1]
		log.Println("Found protocol value:", Protocol)
	default:
		return fmt.Errorf("protocol value '%s' is incorrect: expected TCP or UDP", v[1])
	}
	return nil
}
