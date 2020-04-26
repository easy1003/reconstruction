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

type NewPerformance struct {
	Play          *Play
	Audience      int64
	Amount        int64
	VolumeCredits int64
}

type statementData struct {
	Customer           string
	Performances       []*NewPerformance
	Play               Play
	TotalAmount        int64
	TotalVolumeCredits int64
}

func Statement(invoice *Invoice, plays map[string]*Play) (string, error) {
	statementData := new(statementData)
	statementData.Customer = invoice.Customer
	statementData.Performances = enrichPerformance(invoice.Performances, plays)
	statementData.TotalAmount = totalAmount(statementData)
	statementData.TotalVolumeCredits = totalVolumeCredits(statementData)
	return RenderPlainText(statementData)
}

func enrichPerformance(performances Performances, plays map[string]*Play) []*NewPerformance {
	result := make([]*NewPerformance, 0, len(performances))
	for _, perf := range performances {
		aNewPerformance := new(NewPerformance)
		aNewPerformance.Play = playFor(perf, plays)
		aNewPerformance.Audience = perf.Audience
		thisAmount, err := amountFor(aNewPerformance)
		if err != nil {
			log.WithError(err).Errorf("amoutFor find err")
			continue
		}
		aNewPerformance.Amount = thisAmount
		aNewPerformance.VolumeCredits = volumeCreditsFor(aNewPerformance)

		result = append(result, aNewPerformance)
	}
	return result

}

func playFor(aPerformance *Performance, plays map[string]*Play) *Play {
	return plays[aPerformance.PlayID]
}

func amountFor(aNewPerformance *NewPerformance) (int64, error) {
	result := int64(0)
	switch aNewPerformance.Play.Type {
	case "tragedy":
		result = 40000
		if aNewPerformance.Audience > 30 {
			result += 1000 * (aNewPerformance.Audience - 30)
		}
		break
	case "comedy":
		result = 30000
		if aNewPerformance.Audience > 20 {
			result += 10000 + 500*(aNewPerformance.Audience-20)
		}
		result += 300 * aNewPerformance.Audience
		break
	default:
		return 0, fmt.Errorf("unknown type, type: %v", aNewPerformance.Play.Type)
	}
	return result, nil
}

func totalAmount(data *statementData) int64 {
	result := int64(0)
	for _, perf := range data.Performances {
		result += perf.Amount
	}
	return result
}

func volumeCreditsFor(aNewPerformance *NewPerformance) int64 {
	result := int64(0)
	result += findMax(aNewPerformance.Audience-30, 0)
	if aNewPerformance.Play.Type == "comedy" {
		result += int64(math.Floor(float64(aNewPerformance.Audience) / 5))
	}
	return result
}

func totalVolumeCredits(data *statementData) int64 {
	result := int64(0)
	for _, perf := range data.Performances {
		result += perf.VolumeCredits
	}
	return result
}

func RenderPlainText(data *statementData) (string, error) {
	var (
		result string
	)
	// main process
	result = fmt.Sprintf("Statement for %v \n", data.Customer)
	for _, perf := range data.Performances {

		// print line for this order
		result += fmt.Sprintf("  %s: %s (%v seats) \n", perf.Play.Name, usd(perf.Amount), perf.Audience)
	}

	result += fmt.Sprintf("Amount owed is %v \n", usd(data.TotalAmount))
	result += fmt.Sprintf("You earned %v credits\n", data.TotalVolumeCredits)

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
	statement, err := Statement(invoice, plays)
	if err != nil {
		log.WithError(err).Errorf("statement find err")
		return
	}
	fmt.Println(statement)
}
