package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ChimeraCoder/anaconda"
)

const lastUpdate = "https://public.rts.iebc.or.ke/jsons/round1/results/Kenya_Elections_Presidential/lastUpdate.json"
const presUpdate = "https://public.rts.iebc.or.ke/jsons/round1/results/Kenya_Elections_Presidential/1/info.json"
const resultsPage = "https://public.rts.iebc.or.ke/results/results.html"

var (
	twitterConsumerKey    = getenv("TWITTERCONSUMERKEY")
	twitterConsumerSecret = getenv("TWITTERCONSUMERSECRET")
	twitterAccessToken    = getenv("TWITTERACCESSTOKEN")
	twitterAccessSecret   = getenv("TWITTERACCESSSECRET")
)

func getenv(val string) string {
	s := os.Getenv(val)
	if s == "" {
		log.Fatalf("Missing env variable %s", val)
	}
	return s
}

var api *anaconda.TwitterApi

func main() {
	anaconda.SetConsumerKey(twitterConsumerKey)
	anaconda.SetConsumerSecret(twitterConsumerSecret)
	api = anaconda.NewTwitterApi(twitterAccessToken, twitterAccessSecret)

	for {
		if err := updateLastUpdate(); err != nil {
			log.Print(err)
		}
		time.Sleep(time.Minute * 2)
	}
}

type results struct {
	Progress struct {
		Processed int64
		Total     int64
	}
	Results struct {
		Parties []struct {
			ID      int64
			Acronym string
			Name    string
			Ord     int64
			Votes   struct {
				Presential    int64
				Absentee      int64
				International int64
				Special       int64
				total         int64
				Percent       float64
			}
		}
		Abstention int64
		Blank      int64
		Null       int64
		Census     int64
	}
	Timestamp     int64
	realTimestamp int64

	fetched string
}

func updateLastUpdate() error {
	resp, err := http.Get(presUpdate)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if lastFetched(dat) {
		return nil
	}

	updateLastFetch(dat)
	res := results{}
	err = json.Unmarshal(dat, &res)
	if err != nil {
		return err
	}
	res.realTimestamp, err = getRealTimestamp(res.Timestamp)
	if err != nil {
		return err
	}
	if len(res.Results.Parties) < 3 {
		return fmt.Errorf("invalid response: %s", dat)
	}
	tweet := pretty(res)
	fmt.Printf("\n%s", tweet)

	_, err = api.PostTweet(tweet, nil)
	if err != nil {
		return fmt.Errorf("failed to post tweet: %v", err)
	}
	log.Printf("Results update tweet posted")
	return nil
}
func getRealTimestamp(t1 int64) (int64, error) {
	d := fmt.Sprintf("%d", t1)
	if len(d) < 10 {
		return 0, fmt.Errorf("invalid original timestamp")
	}
	s := d[:10]
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return int64(i), nil
}

const lastFetchFile = "lastfetched.dat"

func lastFetched(new []byte) bool {
	lastF, err := ioutil.ReadFile(lastFetchFile)
	if err != nil {
		log.Printf("failed to read last fetch file: %v", err)
		return false
	}
	return bytes.Equal(new, lastF)
}

func updateLastFetch(new []byte) {
	err := ioutil.WriteFile(lastFetchFile, new, 0644)
	if err != nil {
		log.Printf("failed to write last fetch: %v", err)
	}

}

var shortnames = map[string]string{
	"UHURU KENYATTA":               "KENYATTA (JP)",
	"RAILA ODINGA":                 "ODINGA (ODM)",
	"JOSEPH WILLIAM NTHIGA NYAGAH": "Nyagah (IND)",
	"JOHN EKURU LONGOGGY AUKOT":    "Aukot (TAK)",
	"MOHAMED ABDUBA DIDA":          "Dida (ARK)",
	"JAPHETH  KAVINGA KALUYU":      "Kaluyu (IND)",
	"SHAKHALAGA KHWA JIRONGO":      "Jirongo (UDP)",
	"MICHAEL WAINAINA MWAURA":      "Mwaura (IND)",
}

func pretty(res results) string {
	peeps := res.Results.Parties[:2]
	var out string
	out += fmt.Sprintf("#KenyaDecides UPDATE\n\n")
	for _, p := range peeps {
		out += fmt.Sprintf("%s %d (%.2f%%)\n",
			shortnames[p.Name],
			p.Votes.Presential+p.Votes.Absentee+p.Votes.International+p.Votes.Special,
			p.Votes.Percent,
		)
	}
	out += fmt.Sprintf("\nValid votes: %d\n", res.Results.Blank)
	out += fmt.Sprintf("Stations: %d / %d\n", res.Progress.Processed, res.Progress.Total)
	return out
}
