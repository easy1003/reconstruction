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
		result        string
		totalAmout    int64
		volumeCredits int64
	)
	playFor := func(aPerformance *Performance) *Play {
		return plays[aPerformance.PlayID]
	}

	result = fmt.Sprintf("Statement for %v \n", invoice.Customer)
	format := func(value float64) string {
		return fmt.Sprintf("$%v", fmt.Sprintf("%.2f", value))
	}

	for _, perf := range invoice.Performances {
		play := playFor(perf)
		thisAmount ,err := amountFor(perf, play)
		if err != nil {
			return "", err
		}
		// add volume credits
		volumeCredits += findMax(perf.Audience-30, 0)
		// add extra credit for every ten comedy attendees
		if play.Type == "comedy" {
			volumeCredits += int64(math.Floor(float64(perf.Audience) / 5))
		}

		// print line for this order
		result += fmt.Sprintf("  %s: %s (%v seats) \n", play.Name, format(float64(thisAmount)/100), perf.Audience)
		totalAmout += thisAmount
	}

	result += fmt.Sprintf("Amount owed is %v \n", format(float64(totalAmout)/100))
	result += fmt.Sprintf("You earned %v credits\n", volumeCredits)

	return result, nil
}

func amountFor(perf *Performance, play *Play) (int64, error)  {
	result := int64(0)
	switch play.Type {
	case "tragedy":
		result = 40000
		if perf.Audience > 30 {
			result += 1000 * (perf.Audience - 30)
		}
		break
	case "comedy":
		result = 30000
		if perf.Audience > 20 {
			result += 10000 + 500*(perf.Audience-20)
		}
		result += 300 * perf.Audience
		break
	default:
		return 0, fmt.Errorf("unknown type, type: %v", play.Type)
	}
	return result, nil
}


func findMax(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
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
