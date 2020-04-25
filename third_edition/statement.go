package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type Invoices []*Invoice

type Invoice struct {
	Customer     string       `json:"customer"`
	Performances Performances `json:"performances"`
}

type Performances []*Performance

type Performance struct {
	PlayID   string `json:"playID"`
	Audience int64  `json:"audience"`
}

func ReadInvoices(r io.Reader) (*Invoices, error) {
	result := new(Invoices)
	dec := json.NewDecoder(r)
	if err := dec.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

type Plays struct {
	Plays map[string]Play
}

type Play struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func ReadPlays(data []byte) (map[string]*Play, error) {
	result := make(map[string]*Play)
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func Statement1(invoice *Invoice, plays map[string]*Play) (string, error) {
	var (
		result      string
		totalAmount int64
	)

	// inner func playFor
	playFor := func(aPerformance *Performance) *Play {
		return plays[aPerformance.PlayID]
	}

	// inner func amountFor
	amountFor := func(aPerformance *Performance) (int64, error) {
		result := int64(0)
		switch playFor(aPerformance).Type {
		case "tragedy":
			result = 40000
			if aPerformance.Audience > 30 {
				result += 1000 * (aPerformance.Audience - 30)
			}
			break
		case "comedy":
			result = 30000
			if aPerformance.Audience > 20 {
				result += 10000 + 500*(aPerformance.Audience-20)
			}
			result += 300 * aPerformance.Audience
			break
		default:
			return 0, fmt.Errorf("unknown type, type: %v", playFor(aPerformance).Type)
		}
		return result, nil
	}

	// inner func volumeCreditsFor
	volumeCreditsFor := func(aPerformance *Performance) int64 {
		result := int64(0)
		result += findMax(aPerformance.Audience-30, 0)
		if playFor(aPerformance).Type == "comedy" {
			result += int64(math.Floor(float64(aPerformance.Audience) / 5))
		}
		return result
	}

	// inner func totalVolumeCredits
	totalVolumeCredits := func() int64 {
		result := int64(0)
		for _, perf := range invoice.Performances {
			result += volumeCreditsFor(perf)
		}
		return result
	}

	// main process
	result = fmt.Sprintf("Statement for %v \n", invoice.Customer)
	for _, perf := range invoice.Performances {
		thisAmount, err := amountFor(perf)
		if err != nil {
			return "", err
		}
		// print line for this order
		result += fmt.Sprintf("  %s: %s (%v seats) \n", playFor(perf).Name, usd(thisAmount), perf.Audience)
		totalAmount += thisAmount
	}

	result += fmt.Sprintf("Amount owed is %v \n", usd(totalAmount))
	result += fmt.Sprintf("You earned %v credits\n", totalVolumeCredits())

	return result, nil
}

func findMax(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func usd(value int64) string {
	return fmt.Sprintf("$%v", fmt.Sprintf("%.2f", float64(value)/100))
}

func main() {
	jsonFile, err := os.Open("data/invoices.json")

	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)

	r := strings.NewReader(string(byteValue))
	invoices, err := ReadInvoices(r)
	if err != nil {
		log.Error("parse json failed")
		return
	}

	for _, invoice := range *invoices {
		log.Infof("customer: %v", invoice.Customer)
		for _, perf := range invoice.Performances {
			log.Infof(" palyID: %v, audiences: %v", perf.PlayID, perf.Audience)
		}
	}

	jsonFile2, err := os.Open("data/plays.json")
	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile2.Close()

	byteValue2, _ := ioutil.ReadAll(jsonFile2)

	plays, err := ReadPlays(byteValue2)
	if err != nil {
		log.WithError(err).Errorf("parse json failed")
		return
	}

	for playID, play := range plays {
		log.Infof("playID: %v, play_name: %v, play_type: %v", playID, play.Name, play.Type)
	}
	time.Sleep(2 * time.Second)

	invoice := (*invoices)[0]
	statement, err := Statement1(invoice, plays)
	if err != nil {
		log.WithError(err).Errorf("statement find err")
		return
	}
	fmt.Println(statement)
}
