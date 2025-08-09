package checklist

import (
	"bufio"
	"fmt"
	"goodcheckgogo/utils"
	"log"
	"os"
	"strings"
)

type Website struct {
	Address                         string
	IP                              string
	IsResolved                      bool
	HasSuccesses                    bool
	MostSuccessfulStrategyNum       int
	MostSuccessfulStrategySuccesses int
	LastResponseCode                int
}

func NewWebsite(addr string) Website {
	w := Website{
		Address:                         addr,
		IP:                              "unknown",
		IsResolved:                      false,
		HasSuccesses:                    false,
		MostSuccessfulStrategyNum:       -1,
		MostSuccessfulStrategySuccesses: -1,
		LastResponseCode:                -1,
	}
	return w
}

func ReadChecklist(file string) ([]Website, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("can't open a file '%s': %v", file, err)
	}
	defer f.Close()

	var w []Website

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		if !utils.IsCommented(scan.Text(), "/") {
			addr := cleanURL(scan.Text())
			log.Println("URL to check:", addr)
			w = append(w, NewWebsite(addr))
		}
	}
	return w, nil
}

func cleanURL(url string) string {
	withReplaces := utils.InsensitiveReplace(url, "http://", "")
	withReplaces = utils.InsensitiveReplace(withReplaces, "https://", "")

	withReplaces = strings.Split(withReplaces, `/`)[0]

	withReplaces = "https://" + withReplaces

	return withReplaces
}

var (
	//uzpkfa50vqlgb61wrmhc72xsnid83ytoje94-
	//0123456789abcdefghijklmnopqrstuvwxyz-
	//clusterDecodeArrayA = [...]string{"u", "z", "p", "k", "f", "a", "5", "0", "v", "q", "l", "g", "b", "6",
	//	"1", "w", "r", "m", "h", "c", "7", "2", "x", "s", "n", "i", "d", "8", "3", "y", "t", "o", "j", "e", "9", "4", "-"}
	//clusterDecodeArrayB = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d",
	//	"e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "-"}
	//elsz6dkry5cjqx4bipw3ahov29gnu08fmt1
	//uzpkfa50vqlgb61wrmhc72xsnid83ytoje94
	clusterDecodeArrayA = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e",
		"f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "-"}
	clusterDecodeArrayB = [...]string{"7", "e", "l", "s", "z", "6", "d", "k", "r", "y", "5", "c", "j", "q", "x",
		"4", "b", "i", "p", "w", "3", "a", "h", "o", "v", "2", "9", "g", "n", "u", "0", "8", "f", "m", "t", "1", "-"}
)

func ConvertClusterToURL(codename string) string {
	decodedCodename := ""
	for _, letter := range codename {
		l := string(letter)
		for i := 0; i < len(clusterDecodeArrayA); i++ {
			if l == clusterDecodeArrayA[i] {
				decodedCodename = decodedCodename + clusterDecodeArrayB[i]
				break
			}
		}
	}
	decodedCodename = "https://rr1---sn-" + decodedCodename + ".googlevideo.com"
	return decodedCodename
}
