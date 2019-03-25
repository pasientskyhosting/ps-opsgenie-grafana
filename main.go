package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"./grada"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
)

// Find substring in a slice
func Find(a []string, x string) string {
	for i, n := range a {
		matched, _ := regexp.MatchString(x, n)
		if matched {
			return strings.Replace(a[i], x, "", -1)
		}
	}
	return ""
}

func main() {
	// `OpsGenie fetch interval` from environment variable or default to 60s
	fetchInterval, _ := strconv.ParseInt(os.Getenv("OPSGENIE_FETCH_INTERVAL"), 0, 64)
	if fetchInterval == 0 {
		fetchInterval = 60
	}
	// `OpsGenie api key` from environment variable or die
	apiKey := os.Getenv("OPSGENIE_API_KEY")
	if apiKey == "" {
		log.Fatal("Environment variable OPSGENIE_API_KEY not set")
	}
	// map severity tag values to integers for sorting in grafana
	severity := map[string]int{
		"info":     100,
		"warning":  200,
		"critical": 300,
		"":         0,
	}
	// OpsGenie client
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(apiKey)
	alertCli, _ := cli.AlertV2()
	dash := grada.GetDashboard()
	columns := []grada.Column{
		{Text: "Message", Type: "string"},
		{Text: "AlertType", Type: "string"},
		{Text: "Severity", Type: "int"},
		// {Text: "Tags", Type: "[]string"},
		{Text: "Status", Type: "string"},
		{Text: "IsSeen", Type: "bool"},
		{Text: "Acknowledged", Type: "bool"},
		{Text: "Created", Type: "time"},
		{Text: "Updated", Type: "time"},
		{Text: "TinyId", Type: "string"},
		{Text: "Owner", Type: "string"},
		{Text: "Description", Type: "string"},
	}
	// create metric
	OPSGENIEmetric, err := dash.CreateMetricWithBufSize("OpsGenie", 1)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		OPSGENIEmetric.Add(1)
		rows := []grada.Row{}
		// get alert list
		response, err := alertCli.List(alertsv2.ListAlertRequest{
			Limit:                100,
			Offset:               0,
			Query:                "status=open",
			SearchIdentifierType: alertsv2.Name,
		})
		if err != nil {
			fmt.Println(err.Error())
			break
		} else {
			// get alert details
			for _, alert := range response.Alerts {
				response2, err2 := alertCli.Get(alertsv2.GetAlertRequest{
					Identifier: &alertsv2.Identifier{
						TinyID: alert.TinyID,
					},
				})

				if err2 != nil {
					fmt.Println(err2.Error())
					continue
				} else {
					alertDetail := response2.Alert
					// create and append grafana row
					row := []grada.Row{
						{
							alert.Message,
							Find(alert.Tags, "alert_type: "),         // Alert Type
							severity[Find(alert.Tags, "severity: ")], // Severity
							// alert.Tags,
							alert.Status,
							alert.IsSeen,
							alert.Acknowledged,
							alert.CreatedAt,
							alert.UpdatedAt,
							alert.TinyID,
							alert.Owner,
							alertDetail.Description,
						},
					}
					rows = append(rows, row...)
				}
			}
		}
		// update grafana table
		grada.Table = []grada.TableResponse{
			{
				Columns: columns,
				Rows:    rows,
				Type:    "table",
			},
		}
		time.Sleep(time.Duration(fetchInterval) * time.Second)
	}
}
