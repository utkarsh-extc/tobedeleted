package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/src/io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"time"
)

var c http.Client

type GetResponse struct {
	Centers []Centers `json:"centers"`
}
type Sessions struct {
	SessionID         string   `json:"session_id"`
	Date              string   `json:"date"`
	AvailableCapacity int      `json:"available_capacity"`
	MinAgeLimit       int      `json:"min_age_limit"`
	Vaccine           string   `json:"vaccine"`
	Slots             []string `json:"slots"`
}
type Centers struct {
	CenterID     int        `json:"center_id"`
	Name         string     `json:"name"`
	Address      string     `json:"address"`
	StateName    string     `json:"state_name"`
	DistrictName string     `json:"district_name"`
	BlockName    string     `json:"block_name"`
	Pincode      int        `json:"pincode"`
	Lat          int        `json:"lat"`
	Long         int        `json:"long"`
	From         string     `json:"from"`
	To           string     `json:"to"`
	FeeType      string     `json:"fee_type"`
	Sessions     []Sessions `json:"sessions"`
}

var pinCode = ""
var emailHost = "smtp.gmail.com"
var emailFrom = ""
var emailPassword = ""
var duration = 1 * time.Minute
var count int

func main() {
	flag.StringVar(&pinCode, "districtCode", "363", "pincode of your area\ne.g 363 for pune")
	flag.StringVar(&emailFrom, "emailId", emailFrom, "your_id@gmail.com")
	flag.StringVar(&emailPassword, "password", emailPassword, "Google App Password \ncreate new using https://support.google.com/accounts/answer/185833")
	flag.DurationVar(&duration, "interval", duration, "time interval as duration \ne.g. 30s,5m,1h")

	flag.Parse()

	if pinCode == "" || emailPassword == "" || emailFrom == "" {
		flag.Usage()
		return
	}

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	fmt.Println("looping started")

	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

loop:
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			date := fmt.Sprintf("%02d-%02d-%04d", now.Day(), now.Month(), now.Year())
			req, _ := http.NewRequest(http.MethodGet,
				"https://cdn-api.co-vin.in/api/v2/appointment/sessions/public/calendarByDistrict?district_id="+pinCode+"&date="+date, nil)

			req.Header.Set("cache-control", "no-cache")
			req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML,like Gecko) Chrome/90.0.4430.93 Safari/537.36")

			resp, err := c.Do(req)
			if err != nil {
				fmt.Errorf("%v\n", err)
				time.Sleep(30 * time.Second)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				fmt.Errorf("%v\n", resp.StatusCode)
				time.Sleep(30 * time.Second)
				continue
			}

			slots := new(GetResponse)

			func() {
				defer func() { _ = resp.Body.Close() }()
				all, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Errorf("%v:stop using this tool\n", resp.StatusCode)
				}

				err = json.Unmarshal(all, slots)
				if err != nil {
					fmt.Errorf("unable to unmarshal: %v", err)
				}
			}()
			var availableCenters []Centers

			found := false
			for _, c := range slots.Centers {
				for _, s := range c.Sessions {
					if s.AvailableCapacity > 0 && s.MinAgeLimit < 45 {
						availableCenters = append(availableCenters, c)
						found = true
					}
				}
			}
			if found {
				func() {
					blob, _ := json.MarshalIndent(availableCenters, "", "  ")
					fmt.Println(string(blob))

					emailPort := 587

					// Set up authentication information.
					auth := smtp.PlainAuth("", emailFrom, emailPassword, emailHost)

					// Connect to the server, authenticate, set the sender and recipient,
					// and send the email all in one step.
					to := []string{emailFrom}
					msg := []byte("To: " + emailFrom + "\r\n" +
						"Subject: Available Centers!\r\n" +
						"\r\n" +
						"List : \r\n" +
						string(blob) + "\r\n")
					err := smtp.SendMail(fmt.Sprintf("%s:%d", emailHost, emailPort), auth, emailFrom, to, msg)
					if err != nil {
						log.Fatal(err)
					}

					count++
				}()

				time.Sleep(10 * time.Minute)
				if count > 10 {
					break loop
				}
			} else {
				count = 0
			}
		}
	}
}
