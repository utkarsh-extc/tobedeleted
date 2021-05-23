package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var c http.Client

var pinCode = ""
var pinCodes []string
var pinCodeIndex int
var emailHost = "smtp.gmail.com"
var emailFrom = ""
var emailRecipients = []string{""}
var emailPassword = ""

// api limit 100 req/5min per ip
var duration = 5 * time.Minute / 95
var count int
var mobile string
var secret = "U2FsdGVkX18qwZAGasLkIRs7giixSNa0qHKofrof7HAZ+creL7yka6fv6Jfp/ViSnyIVtCQpLRjapsF8JYBAVw==" // from inspect
var txnID string
var token string
var beneficiaries []Beneficiaries
var aptID string
var booked bool
var minAge int
var vaccineName string
var secondDose bool

func main() {
	list := ""
	flag.StringVar(&pinCode, "pincode", "411027", "list of interested pincode")
	flag.StringVar(&emailFrom, "emailId", emailFrom, "your_id@gmail.com")
	flag.StringVar(&list, "to", emailFrom, "comma separated email list")
	flag.StringVar(&emailPassword, "password", emailPassword, "Google App Password \ncreate new using https://support.google.com/accounts/answer/185833")
	flag.DurationVar(&duration, "interval", duration, "time interval as duration \ne.g. 30s,5m,1h")
	flag.StringVar(&mobile, "mobileNumber", "", "10 digit mobile number")
	flag.IntVar(&minAge, "minage", 18, "min age group 18 or 45")
	flag.StringVar(&vaccineName, "vaccine", "COVISHIELD", "perferred vaccine name\ne.g COVISHIELD or COVAXIN")
	flag.BoolVar(&secondDose, "secondDose", false, "set true to search slot for second dose")

	flag.Parse()

	emailRecipients = strings.Split(list, ",")
	sendEmail := len(emailRecipients) > 0 && emailFrom != "" && emailPassword != ""

	if !(strings.Contains(vaccineName, "COVAXIN") || strings.Contains(vaccineName, "COVISHIELD")) {
		flag.Usage()
		return
	}

	pinCodes = strings.Split(pinCode, ",")

	if len(pinCodes) < 1 || mobile == "" || len(mobile) == 11 {
		flag.Usage()
		return
	}

	fmt.Println("looping started")

	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

createOtp:
	// create otp
	otpReq, _ := http.NewRequest(http.MethodPost, "https://cdn-api.co-vin.in/api/v2/auth/generateMobileOTP",
		bytes.NewBufferString(fmt.Sprintf("{\"secret\":\"%s\",\"mobile\":%s}", secret, mobile)))
	otpReq.Header.Set("cache-control", "no-cache")
	otpReq.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML,like Gecko) Chrome/90.0.4430.93 Safari/537.36")

	otpResp, err := c.Do(otpReq)
	if err != nil {
		fmt.Println(fmt.Errorf("create otp:%v", err))
		time.Sleep(30 * time.Second)
		goto createOtp
	}
	func(resp *http.Response) {
		defer func() { _ = resp.Body.Close() }()
		all, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}

		tID := make(map[string]string)
		err = json.Unmarshal(all, &tID)
		if err != nil {
			fmt.Println(fmt.Errorf("unable to unmarshal otp resp: %v", err))
			time.Sleep(30 * time.Second)
			return
		}

		txnID, _ = tID["txnId"]

	}(otpResp)

	if txnID == "" {
		goto createOtp
	}

readOtp:
	// read otp
	fmt.Println("enter received opt on number", mobile)
	fmt.Println("press 0 and enter to retry")

	otp := ""
	_, _ = fmt.Fscanf(os.Stdin, "%s", &otp)
	if otp == "0" {
		goto createOtp
	}
	if len(otp) < 6 {
		goto readOtp
	}

	h := sha256.New()
	h.Write([]byte(otp))

	//create token
	body := fmt.Sprintf("{\"otp\":\"%x\",\"txnId\":\"%s\"}", h.Sum(nil), txnID)
	tokReq, _ := http.NewRequest(http.MethodPost, "https://cdn-api.co-vin.in/api/v2/auth/validateMobileOtp",
		bytes.NewBufferString(body))
	tokReq.Header.Set("pragma", "no-cache")
	tokReq.Header.Set("content-type", "application/json")
	tokReq.Header.Set("content-length", string(rune(len(body))))
	tokReq.Header.Set("origin", "https://selfregistration.cowin.gov.in")
	tokReq.Header.Set("referer", "https://selfregistration.cowin.gov.in/")
	tokReq.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML,like Gecko) Chrome/90.0.4430.93 Safari/537.36")

	tokResp, err := c.Do(tokReq)
	if err != nil {
		fmt.Println(fmt.Errorf("validate otp:%v", err))
		goto createOtp
	}
	func(resp *http.Response) {
		defer func() { _ = resp.Body.Close() }()
		all, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}

		tok := make(map[string]string)
		err = json.Unmarshal(all, &tok)
		if err != nil {
			fmt.Println(fmt.Errorf("unable to unmarshal token: %v", err))
			return
		}

		token, _ = tok["token"]

		_ = ioutil.WriteFile("token.txt", []byte(token), 0644)

	}(tokResp)

	if token == "" {
		goto createOtp
	}

	// create timer on token expiry
	jwtToken, _ := jwt.Parse(token, nil)
	exp, _ := jwtToken.Claims.(jwt.MapClaims)["exp"].(float64)
	if exp == 0 {
		goto createOtp
	}

	expChan := time.After(time.Until(time.Unix(int64(exp), 0)))
	fmt.Println("token will expired at", time.Unix(int64(exp), 0).String())
	ticker := time.NewTicker(duration)

	func() {
		// get beneficiaries
		benReq, _ := http.NewRequest(http.MethodGet,
			"https://cdn-api.co-vin.in/api/v2/appointment/beneficiaries", nil)
		benReq.Header.Set("authorization", "Bearer "+token)
		benReq.Header.Set("cache-control", "no-cache")
		benReq.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML,like Gecko) Chrome/90.0.4430.93 Safari/537.36")

		benResp, err := c.Do(benReq)
		if err != nil {
			fmt.Println(fmt.Errorf("get beneficiary:%v", err))
			return
		}
		func(resp *http.Response) {
			defer func() { _ = resp.Body.Close() }()
			all, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			ben := new(GetBeneficiariesResponse)
			err = json.Unmarshal(all, ben)
			if err != nil {
				fmt.Println(fmt.Errorf("unable to unmarshal beneficiary: %v", err))
				return
			}

			beneficiaries = append(beneficiaries, ben.Beneficiaries...)

		}(benResp)
	}()

	if len(beneficiaries) < 1 {
		fmt.Println("please add beneficiaries")
		os.Exit(0)
	}

loop:
	for {
		select {
		case <-expChan:
			fmt.Println("token expired")
			txnID = ""
			token = ""
			_ = os.Remove("token.txt")
			ticker.Stop()
			goto createOtp

		case <-ticker.C:
			now := time.Now()
			date := fmt.Sprintf("%02d-%02d-%04d", now.Day(), now.Month(), now.Year())
			var availableCenters []Centers
			found := false

			pc := pinCodes[pinCodeIndex]
			pinCodeIndex++
			if pinCodeIndex >= len(pinCodes) {
				pinCodeIndex = 0
			}

			req, _ := http.NewRequest(http.MethodGet,
				"https://cdn-api.co-vin.in/api/v2/appointment/sessions/calendarByPin?pincode="+pc+"&date="+date, nil)
			req.Header.Set("authorization", "Bearer "+token)
			req.Header.Set("cache-control", "no-cache")
			req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML,like Gecko) Chrome/90.0.4430.93 Safari/537.36")

			slotResp, err := c.Do(req)
			if err != nil {
				fmt.Println(fmt.Errorf("get slots:%v", err))
				continue
			}

			if slotResp.StatusCode != http.StatusOK {
				fmt.Println(fmt.Errorf("get slots:%v", slotResp.StatusCode))
				continue
			}

			slots := new(GetResponse)

			func(resp *http.Response) {
				defer func() { _ = resp.Body.Close() }()
				all, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Println(fmt.Errorf("%v:stop using this tool", resp.StatusCode))
				}

				err = json.Unmarshal(all, slots)
				if err != nil {
					fmt.Println(fmt.Errorf("unable to unmarshal slot: %v", err))
				}
			}(slotResp)

			for _, c := range slots.Centers {
				freeSessions := make([]Sessions, len(c.Sessions))
				for _, s := range c.Sessions {
					if s.AvailableCapacity > 0 && s.MinAgeLimit == minAge {
						if secondDose && s.AvailableCapacityDose2 > 0 {
							freeSessions = append(freeSessions, s)
							found = true
						} else if secondDose && s.AvailableCapacityDose1 > 0 {
							freeSessions = append(freeSessions, s)
							found = true
						}
					}
				}
				if len(freeSessions) > 0 {
					c.Sessions = freeSessions
					availableCenters = append(availableCenters, c)
				}
			}

			if found {
				blob, _ := json.MarshalIndent(availableCenters, "", "  ")
				fmt.Println(string(blob))
				// schedule appointment
				for _, center := range availableCenters {
					schInput := ScheduleRequestInput{
						Dose: func() int {
							if !secondDose {
								return 1
							}
							return 2
						}(),
						SessionID: center.Sessions[0].SessionID,
						Slot:      center.Sessions[0].Slots[0],
						Beneficiaries: func() (ben []string) {
							for _, b := range beneficiaries {
								ben = append(ben,
									b.BeneficiaryReferenceID)
							}
							return
						}(),
					}

					blob, _ := json.Marshal(schInput)

					// will not work because of CAPTCHA
					// POST https://cdn-api.co-vin.in/api/v2/appointment/schedule
					schReq, _ := http.NewRequest(http.MethodPost,
						"https://cdn-api.co-vin.in/api/v2/appointment/schedule", bytes.NewBuffer(blob))
					schReq.Header.Set("authorization", "Bearer "+token)
					schReq.Header.Set("cache-control", "no-cache")
					schReq.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML,like Gecko) Chrome/90.0.4430.93 Safari/537.36")

					schResp, err := c.Do(schReq)
					if err != nil {
						fmt.Println(fmt.Errorf("book slot:%v", err))
						return
					}
					if schResp.StatusCode != http.StatusOK {
						continue
					}

					func(resp *http.Response) {
						defer func() { _ = resp.Body.Close() }()
						all, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							return
						}

						apt := make(map[string]string)
						err = json.Unmarshal(all, &apt)
						if err != nil {
							fmt.Println(fmt.Errorf("unable to unmarshal schedule resp: %v", err))
							return
						}

						aptID, _ = apt["appointment_id"]
						if aptID != "" {
							fmt.Println("booked apt id ", aptID)
						}
						booked = true
					}(schResp)
					break
				}
				blob, _ = json.MarshalIndent(availableCenters, "", "  ")

				if sendEmail {
					emailPort := 587

					// Set up authentication information.
					auth := smtp.PlainAuth("", emailFrom, emailPassword, emailHost)

					// Connect to the server, authenticate, set the sender and recipient,
					// and send the email all in one step.
					to := emailRecipients
					msg := []byte("To: " + list + "\r\n" +
						"From: Golang looper\r\n" +
						"Subject: Available Centers!\r\n" +
						"\r\n" +
						"Appointment ID: " + aptID + "\r\n" +
						"\r\n" +
						"List : \r\n" +
						string(blob) + "\r\n")
					err = smtp.SendMail(fmt.Sprintf("%s:%d", emailHost, emailPort), auth, emailFrom, to, msg)
					if err != nil {
						log.Fatal(err)
					}
				}

				count++

				time.Sleep(5 * time.Minute)
				if count > 10 || booked {
					break loop
				}
			} else {
				count = 0
			}
		}
	}
}
