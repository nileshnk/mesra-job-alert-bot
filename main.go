package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	colly "github.com/gocolly/colly/v2"
	dotenv "github.com/joho/godotenv"
)
var LastPostDate string
var BitMesraTPHost string

type AppConfig struct {
	LastPostDate string `json:"last_post_date"`
	HostName string `json:"host_name"`
	IntervalSeconds string `json:"interval_seconds"`
} 
var Config AppConfig 

func main() {
	 EnvErr := dotenv.Load(".env")
	 if EnvErr != nil {
		 log.Fatal(EnvErr)
	 }
	 ConfigData, ConfigErr := os.ReadFile("./config.json")
	 if ConfigErr != nil {
		log.Fatal(ConfigErr)
	 }

	 
	 parseErr :=  json.Unmarshal(ConfigData, &Config)
	 if parseErr != nil {
		 log.Fatal(parseErr)
	 }

	 LastPostDate  = Config.LastPostDate
	 BitMesraTPHost  = Config.HostName
	
	port :=  os.Getenv("GO_ADDR")
	if port == "" {
		port = "5050"
	}
	
	// TimeInterval  := Config.IntervalSeconds //os.Getenv("INTERVAL_SECONDS")
	// if TimeInterval == "" {
	// 	TimeInterval = "10"
	// }
	// IntervalSeconds, IntervalErr := time.ParseDuration(TimeInterval)
	// if IntervalErr != nil {
	// 	log.Fatal(IntervalErr)
	// }
	// fmt.Println(IntervalSeconds)
	interval :=  15 * time.Second
	ticker := time.Tick(interval)
	// Your task to be performed at each interval
	fmt.Println("Starting the Monitor service...")
	for {
		select {
		case <-ticker:
			// Your task goes here
			fmt.Println("Performing task at interval...")
			Init()
		}
	}
}


func Init() {

		userIdentity :=  os.Getenv("USER_ID")
		userPassword := os.Getenv("USER_PASSWORD") 
		payload := strings.NewReader(fmt.Sprintf("identity=%s&password=%s&submit=Login",userIdentity, userPassword))
		RequestUrl := Config.HostName + "auth/login.html";

		createReq, createReqErr := http.NewRequest(http.MethodPost, RequestUrl, payload)
		if createReqErr != nil {
			log.Fatal(createReqErr)
		}

		createReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:78.0) Gecko/20100101 Firefox/78.0")
		createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		client := &http.Client{}
		resp, respErr := client.Do(createReq)

		if respErr != nil {
			log.Fatal(respErr)
		}

		defer resp.Body.Close()

		fmt.Println(resp.Header)
		UserCookie := resp.Cookies()[0]


		// new request to get the data 
		scrapeIndexUrl :=  Config.HostName + "index.html"
		HomePageReq, HomePageReqErr := http.NewRequest("GET", scrapeIndexUrl, nil)
		if HomePageReqErr != nil {
			log.Fatal(HomePageReqErr)
		}

		HomePageReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:78.0) Gecko/20100101 Firefox/78.0")
		HomePageReq.Header.Set("Cookie", UserCookie.String())//fmt.Sprintf());

		HomePageRes, HomePageResErr := client.Do(HomePageReq)

		if HomePageResErr != nil {
			log.Fatal(HomePageResErr)
		}

		defer HomePageRes.Body.Close()

		SiteScraper(scrapeIndexUrl, UserCookie.Value);
}

func TelegramBot(bot *tgbotapi.BotAPI, message string){
	
	// log.Printf("Authorized on account %s", bot.Self.UserName)
	CHAT_ID, _ := strconv.ParseInt(os.Getenv("TELEGRAM_CHAT_ID"), 10, 64)
	var JobAlertChannelChatID int64 = CHAT_ID
	msg := tgbotapi.NewMessage(JobAlertChannelChatID, message)
	_, err := bot.Send(msg)
	if err != nil {
		log.Panic(err)
	}

	// fmt.Println(msgStatus)
}	

type CompanyData struct {
	Company string `json:"company"`
	Deadline time.Time `json:"deadline"`
	PostedOn time.Time `json:"posted_on"`
	UpdatesLink string `json:"updates_link"`
	DetailsLink string `json:"details_link"`
}

func SiteScraper(url string, cookie_session_id string){
	if LastPostDate == "" {
		LastPostDate = time.Now().AddDate(0,0,-1).Format(time.DateOnly)
	}
	
	ParsedQueryDate, _ := time.Parse(time.DateOnly, LastPostDate)

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_API_KEY"))
		if err != nil {
			log.Panic(err)
		}
	bot.Debug = true
	c := colly.NewCollector()
	var allPosts []CompanyData
	c.OnHTML("#job-listings tbody", func(e *colly.HTMLElement) {

		e.DOM.Children().Map(func(i int, s *goquery.Selection) string {

			singlePost := CompanyData{}
			s.Find("td").Each(func(i int, st *goquery.Selection) {

				switch i {
				case 0:
					singlePost.Company = st.Text()
					break
				case 1:
					data, _ := time.Parse("02/01/2006",  st.Text())
					singlePost.Deadline = data
					break
				case 2:
					data, _ := time.Parse("02/01/2006",  st.Text())
					singlePost.PostedOn = data
					break
				case 3:
					st.Find("a").Each(func(i int, lastNode *goquery.Selection) {
						data, _ := lastNode.Attr("href")
						if i==0 {
							singlePost.UpdatesLink =  data
						}
						if i==1 {
							singlePost.DetailsLink = data
						}
					})
					break
				}
		})
			
			if singlePost.PostedOn.After(ParsedQueryDate) {
				messageStr  := createPost(singlePost)
				TelegramBot(bot, messageStr)
			} 

			allPosts = append(allPosts, singlePost)
			return s.Text()+","
		})
		fmt.Println("No New Posts")
		fmt.Println(allPosts[0].Company)
		LastPostDate = allPosts[0].PostedOn.Format(time.DateOnly)
		os.WriteFile("./config.json", []byte(fmt.Sprintf(`{"last_post_date": "%s", "host_name": "%s", "interval_seconds": "%s"}`, LastPostDate, BitMesraTPHost, "15")), 0644)
		fmt.Println("Last Post Date",LastPostDate)
		// fmt.Println(allPosts)
	})
	
	// rawDate := "21/07/2023"
	// lastDate := "20/07/2023"

	// date, _ := time.Parse("02/01/2006", rawDate)
	// date2, _ := time.Parse("02/01/2006", lastDate)
	// if date2. After(date) {
	// 	fmt.Println("True")
	// }
	// fmt.Println(date)
	c.SetCookies( Config.HostName + "index.html", []*http.Cookie{
		{
			Name: "ci_sessions",
			Value: cookie_session_id,
		},
	})
	c.Visit(url)

}

func createPost(post CompanyData) string {
	message := fmt.Sprintf("** New Job Posted! **\n\nCompany: %s\n\nDeadline: %s\n\nPosted On: %s\n\nUpdates Link: %s\n\nDetails Link: %s\n", post.Company, post.Deadline.Format(time.DateOnly), post.PostedOn.Format(time.DateOnly), BitMesraTPHost + post.UpdatesLink, BitMesraTPHost + post.DetailsLink)
	return message
}