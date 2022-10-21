package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"github.com/aiomonitors/godiscord"
	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

var (
	cpm       uint
	result    []string
	cfg       Config
	checks    uint
	valid     uint
	errors    uint
	start     time.Time
	proxyList []string
	Client = http.Client{
		Timeout: 5 * time.Second,
	}
)

type Config struct {
	Main struct {
		Workers int `yaml:"workers"`
		Startid int `yaml:"startid"`
		Stopid  int `yaml:"stopid"`
	} `yaml:"main"`
	Webhook struct {
		Webhook string `yaml:"webhook"`
	} `yaml:"webhook"`
}

type GroupInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsLocked    bool   `json:"isLocked"`
	Owner       struct {
		BuildersClubMembershipType string `json:"buildersClubMembershipType"`
		UserID                     int    `json:"userId"`
		Username                   string `json:"username"`
		DisplayName                string `json:"displayName"`
	} `json:"owner"`

	MemberCount        int  `json:"memberCount"`
	IsBuildersClubOnly bool `json:"isBuildersClubOnly"`
	PublicEntryAllowed bool `json:"publicEntryAllowed"`
}

func groupscrape(id int) {
RESTART:
	groupreq, err := http.NewRequest("GET", fmt.Sprintf("https://groups.roblox.com/v1/groups/%d", id), nil)
	if err != nil {
		fmt.Println(err)
		errors++
	}

	proxyURL, err := url.Parse(fmt.Sprintf("http://%s", proxyList[rand.Intn(len(proxyList))]))
	if err != nil {
		fmt.Println(err)
		errors++
	}

	http.DefaultTransport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	GroupResponse, err := Client.Do(groupreq)
	if err != nil {
		errors++
		goto RESTART
	}
	defer GroupResponse.Body.Close()

	var groupinfo *GroupInfo
	JSONParseError := json.NewDecoder(GroupResponse.Body).Decode(&groupinfo)
	if JSONParseError != nil {
		errors++
		fmt.Println(JSONParseError)
	}

	if groupinfo.Owner.DisplayName == "" {
		if groupinfo.IsLocked != true {
			if groupinfo.PublicEntryAllowed == true {
				valid++
				c := color.New(color.FgHiGreen).Add(color.Underline).Add(color.Bold)
				c.Printf("Group Name: %d | Group ID: %d | Members: %d | https://www.roblox.com/groups/%d\n", groupinfo.Name, groupinfo.ID, groupinfo.MemberCount, groupinfo.ID)
				resultfile, err := os.OpenFile("Config/results.txt", os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Println(err)
				}
				resultfile.WriteString(fmt.Sprintf("Group Name: %d | Group ID: %d | Members: %d | %d\n", groupinfo.Name, groupinfo.ID, groupinfo.MemberCount, groupinfo.ID))
				resultfile.Close()

				discordwebhook(groupinfo)
			}
		}
	}

	checks++
}

func discordwebhook(groupinfo *GroupInfo) {
	http.DefaultTransport = &http.Transport{}
	embed := godiscord.NewEmbed("RoFind found a unclaimed group!", "", fmt.Sprintf("https://www.roblox.com/groups/%d", groupinfo.ID))
	embed.SetAuthor("RoFind for CLI", "", "https://i.imgur.com/B7BPUl3.png")
    embed.SetColor("#2ECC71")
	embed.AddField("Name", strings.Title(groupinfo.Name), false)
	embed.AddField("ID", fmt.Sprintf("%d", groupinfo.ID), true)
	embed.AddField("Members", fmt.Sprintf("%d", groupinfo.MemberCount), true)
	embed.AddField("Description", groupinfo.Description, false)
DISCORDRESTART:
	discorderr := embed.SendToWebhook(cfg.Webhook.Webhook)
	if discorderr != nil {
		fmt.Println(discorderr)
		goto DISCORDRESTART
	}
}

func cpmcounter() {
	for {
		oldchecked := checks
		time.Sleep(1 * time.Second)
		newchecked := checks
		cpm = (newchecked - oldchecked) * 60
	}
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for io := range a {
		a[io] = min + io
	}
	return a
}

func main() {
    tempGreen := color.New(color.FgHiGreen).SprintFunc()
    tempBlue := color.New(color.FgHiBlue).SprintFunc()
	fmt.Printf("Made by %s - Edited by %s\n", tempGreen("Bixmox#2482"), tempBlue("Dev#8888"))
    color.White("")
    color.Yellow("⚠ Warning ⚠")
    color.White("The webhook might not send the message when a group is found,\nkeep track of the console.\n")
    color.White("")
    
	cf, err := os.Open("Config/config.yml")
	if err != nil {
		fmt.Println(err)
	}
	defer cf.Close()

	decoder := yaml.NewDecoder(cf)
	err = decoder.Decode(&cfg)
	if err != nil {
		fmt.Println(err)
	}
	IDRange := makeRange(int(cfg.Main.Startid), int(cfg.Main.Stopid))
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(IDRange), func(i, o int) { IDRange[i], IDRange[o] = IDRange[o], IDRange[i] })

	go cpmcounter()
	start = time.Now()

	proxyFile, err := os.Open("Config/proxies.txt")
	if err != nil {
		fmt.Println(err)
	}

	scanner := bufio.NewScanner(proxyFile)

	for scanner.Scan() {
		proxyList = append(proxyList, scanner.Text())
	}

	startTime := time.Now()
	wg := &sync.WaitGroup{}
	workChannel := make(chan int)
	for i := 0; i <= cfg.Main.Workers; i++ {
		wg.Add(1)
		go worker(wg, workChannel)
	}
	for _, a := range IDRange {
		workChannel <- a
	}
	close(workChannel)
	wg.Wait()
	fmt.Println("Took ", time.Since(startTime))
}

func worker(wg *sync.WaitGroup, jobs <-chan int) {
	for j := range jobs {
		groupscrape(j)
	}
	wg.Done()
}
