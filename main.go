package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	ProblemWithConfigFile bool
	serviceUrl            string
)

//	TfsRequest represents a request to get TFS information
type TfsRequest struct {
	TfsUrl     string `json:"TFSUrl"`
	ProjectUrl string `json:"TeamProjectUrl"`
	UserName   string `json:"TFSUserName"`
	Password   string `json:"TFSPassword"`
	StartDate  string `json:"StartDate"`
	EndDate    string `json:"EndDate"`
}

type ChangesetInfo struct {
	Id            int        `json:"ChangesetId"`
	Comments      string     `json:"Comments"`
	CommittedBy   string     `json:"CommittedBy"`
	CommittedDate string     `json:"CommittedDate"`
	WorkItems     []WorkItem `json:"WorkItems"`
}

type WorkItem struct {
	Title       string `json:"WorkItemTitle"`
	Id          int    `json:"WorkItemId"`
	CreatedBy   string `json:"WorkItemCreatedBy"`
	CreatedDate string `json:"WorkItemCreatedDate"`
}

func main() {
	//	Initialize the config system
	init_config()

	//	Spit our our configuration information:
	log.Printf("[INFO] Using serviceUrl: %v\n", viper.GetString("tfsrequest.serviceurl"))

	//	Our default set of items:
	retval := []ChangesetInfo{}

	//	Create our request:
	request := TfsRequest{
		TfsUrl:     viper.GetString("tfsrequest.tfsurl"),
		ProjectUrl: viper.GetString("tfsrequest.projecturl"),
		UserName:   viper.GetString("tfsrequest.user"),
		Password:   viper.GetString("tfsrequest.password"),
		StartDate:  viper.GetString("tfsrequest.startdate"),
		EndDate:    viper.GetString("tfsrequest.enddate")}

	//	Serialize our request to JSON:
	requestBytes := new(bytes.Buffer)
	if err := json.NewEncoder(requestBytes).Encode(&request); err != nil {
		log.Panicf("[FATAL] %v\n", err)
	}

	// Convert bytes to a reader.
	requestJSON := strings.NewReader(requestBytes.String())

	//	Post the JSON to the api url
	serviceUrl = viper.GetString("tfsrequest.serviceurl")
	log.Println("[INFO] Calling service...")
	res, err := http.Post(serviceUrl, "application/json", requestJSON)
	defer res.Body.Close()

	//	Decode the return object
	if err = json.NewDecoder(res.Body).Decode(&retval); err != nil {
		log.Panicf("[FATAL] %v\n", err)
	}

	//	Indicate we got something back:
	log.Printf("[INFO] Got %v items back.  Formatting using template...\n", len(retval))

	//	Create our map of relevant work items:
	reportwi := make(map[int]string)
	for _, c := range retval {
		for _, w := range c.WorkItems {
			reportwi[w.Id] = w.Title
		}
	}

	//	Create our template
	t := template.New("TFSInfo") // Create a template.
	t, err = t.Parse(`
{{range $key, $value := .}}
	<li>TFS {{$key}} - {{$value}}</li>
{{end}}`)

	// t, err = t.ParseFiles(viper.GetString("template-file")) // Parse template file.
	if err != nil {
		log.Panicf("[FATAL] %v\n", err)
	}

	//	Execute the template:
	buff := bytes.NewBufferString("")
	t.Execute(buff, reportwi)

	//	Save the data to the file
	log.Printf("[INFO] Saving to file: %v\n", viper.GetString("savetofile"))
	f, err := os.Create(viper.GetString("savetofile"))
	defer f.Close()
	if err != nil {
		log.Panicf("[FATAL] %v\n", err)
	}
	f.WriteString(buff.String())
	f.Sync()
}

func init_config() {
	log.Println("[INFO] Initializing configuration...")
	viper.AutomaticEnv() // read in environment variables that match

	//	Parse flags
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	//	Set our defaults
	viper.SetDefault("tfsrequest.serviceurl", "")
	viper.SetDefault("tfsrequest.tfsurl", "")
	viper.SetDefault("tfsrequest.projecturl", "")
	viper.SetDefault("tfsrequest.user", "")
	viper.SetDefault("tfsrequest.password", "")
	viper.SetDefault("tfsrequest.startdate", "")
	viper.SetDefault("tfsrequest.enddate", "")
	viper.SetDefault("savetofile", "changesets.html")

	viper.SetConfigName("tfsinfo2html") // name of config file (without extension)
	viper.AddConfigPath("$HOME")        // adding home directory as first search path
	viper.AddConfigPath(".")            // also look in the working directory

	// If a config file is found, read it in
	// otherwise, make note that there was a problem
	if err := viper.ReadInConfig(); err != nil {
		log.Panicf("[FATAL] There was a problem with your config file: %v", err)
	}
}
