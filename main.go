package main

import (
	"fmt"
	"net/http"
	"io/ioutil"	
	"os"
	"encoding/json"
	"regexp"
	"strconv"
	"time"
	"sync"
)



//Thanks https://mholt.github.io/json-to-go/
type rechatChunk struct {
	Data []struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Command     string `json:"command"`
			Room        string `json:"room"`
			Timestamp   int64  `json:"timestamp"`
			VideoOffset int    `json:"video-offset"`
			Deleted     bool   `json:"deleted"`
			Message     string `json:"message"`
			From        string `json:"from"`
			Tags        struct {
				Badges      string      `json:"badges"`
				Color       string      `json:"color"`
				DisplayName string      `json:"display-name"`
				Emotes      interface{} `json:"emotes"`
				ID          string      `json:"id"`
				Mod         bool        `json:"mod"`
				RoomID      string      `json:"room-id"`
				SentTs      string      `json:"sent-ts"`
				Subscriber  bool        `json:"subscriber"`
				TmiSentTs   string      `json:"tmi-sent-ts"`
				Turbo       bool        `json:"turbo"`
				UserID      string      `json:"user-id"`
				UserType    interface{} `json:"user-type"`
			} `json:"tags"`
			Color string `json:"color"`
		} `json:"attributes"`
		Links struct {
			Self string `json:"self"`
		} `json:"links"`
	} `json:"data"`
	Meta struct {
		Next interface{} `json:"next"`
	} `json:"meta"`
}

const chunkDuration int = 30 //Every rechat chunk is 30 seconds long 

/*
Gets the rechat chunk; parses the json; puts the messages from the chunk into the text string
*/
func getChatChunk(url string, text *string, wg *sync.WaitGroup) {
	var textSum string = ""
	
	response, _ := http.Get(url)
	bytes, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	res := rechatChunk{}
	json.Unmarshal([]byte(string(bytes)), &res)
	
	for i, _ := range(res.Data) {
		msg := res.Data[i].Attributes.Message
		if msg == "" { //if msg is empty the person sending it was banned or timed out recently after
			msg = "<message deleted>"
		}
		tm := time.Unix(res.Data[i].Attributes.Timestamp/1000,0) //If you care about ns accuracy use time.Unix(0, res.Data[i].Attributes.Timestamp)
		sender := res.Data[i].Attributes.From
		textSum += fmt.Sprintf("%v \t %25v  %v \n", tm, sender, msg)
	}
	*text = textSum
	defer wg.Done()
}

func main() {
	_, err := strconv.Atoi(os.Args[1])
	if err != nil {
	    fmt.Printf("Please only pass the vod id (https://www.twitch.tv/videos/{vod id})\n")
	    os.Exit(1)
	}
	rechatURL := "https://rechat.twitch.tv/rechat-messages?video_id=v" + string(os.Args[1]) + "&start="
	response, _ := http.Get(rechatURL + "0")
	bytes, _ := ioutil.ReadAll(response.Body)
	responseString := string(bytes)
	response.Body.Close()
	
	re := regexp.MustCompile("[0-9]+")
	responseNumbers := re.FindAllString(responseString, -1)

	startOffset, _ := strconv.Atoi(responseNumbers[2])
	endOffset, _ := strconv.Atoi(responseNumbers[3])
	
	var chunkCount int = ((endOffset - startOffset) / chunkDuration)

	var chat []string = make([]string, chunkCount)


	var wg sync.WaitGroup
	wg.Add(chunkCount)
	
	for i := 0; i < chunkCount; i++ {
		go getChatChunk(rechatURL + strconv.Itoa(startOffset + (i * chunkDuration)), &chat[i], &wg)
		fmt.Printf("Downloading part %4v of %4v \n", i, chunkCount-1)
	}
	wg.Wait()

	fileName := "rechat_" + string(os.Args[1]) + ".txt"
	file, err := os.Create(fileName)
	if err != nil {
	    fmt.Printf("An error occurred while creating the file: %v \n", err)
	    os.Exit(1)
	}

	fmt.Printf("Combining parts...\n")
	for i := 0; i < chunkCount; i++ {
		file.WriteString(chat[i])
	}
	fmt.Printf("Done, chat downloaded as %v \n", "rechat_" + string(os.Args[1]) + ".txt")
}