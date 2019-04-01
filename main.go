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
	severityMap := map[string]int{
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
		{Text: "Status", Type: "string"},
		{Text: "IsSeen", Type: "int"},
		{Text: "Acknowledged", Type: "int"},
		{Text: "Created", Type: "time"},
		{Text: "Updated", Type: "time"},
		{Text: "TinyId", Type: "string"},
		{Text: "Owner", Type: "string"},
		{Text: "Cluster", Type: "string"},
		{Text: "HostName", Type: "string"},
		{Text: "Description", Type: "string"},
	}
	// create metrics
	OPSGENIEOpenAlerts, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlerts", 10)
	if err != nil {
		log.Fatalln(err)
	}
	OPSGENIEOpenAlertsCritical, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlertsCritical", 10)
	if err != nil {
		log.Fatalln(err)
	}
	OPSGENIEOpenAlertsWarning, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlertsWarning", 10)
	if err != nil {
		log.Fatalln(err)
	}
	OPSGENIEOpenAlertsOther, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlertsOther", 10)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		rowsOpenCount := map[string]float64{
			"critical": 0,
			"warning": 0,
			"other": 0,
			"total": 0,
		}
		rows, rowsCritical, rowsWarning := []grada.Row{}, []grada.Row{}, []grada.Row{}
		rowsAck, rowsAckCritical, rowsAckWarning  := []grada.Row{}, []grada.Row{}, []grada.Row{}
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
					alertType := Find(alert.Tags, "alert_type: ")
					severity := Find(alert.Tags, "severity: ")
					cluster := Find(alert.Tags, "cluster: ")
					hostname := Find(alert.Tags, "hostname: ")
					// create and append grafana row
					row := []grada.Row{
						{
							alert.Message,
							alertType,
							severityMap[severity],
							alert.Status,
							alert.IsSeen,
							alert.Acknowledged,
							alert.CreatedAt,
							alert.UpdatedAt,
							alert.TinyID,
							alert.Owner,
							cluster,
							hostname,
							alertDetail.Description,
						},
					}
					// Lame sorting	
					if alert.Acknowledged {
						switch severity {
						case "critical":
							rowsAckCritical = append(rowsAckCritical, row...)
						case "warning":
							rowsAckWarning = append(rowsAckWarning, row...)
						default:
							rowsAck = append(rowsAck, row...)
						}
					} else {
						rowsOpenCount["total"] ++
						switch severity {
						case "critical":
							rowsOpenCount["critical"] ++
							rowsCritical = append(rowsCritical, row...)
						case "warning":
							rowsOpenCount["warning"] ++
							rowsWarning = append(rowsWarning, row...)
						default:
							rowsOpenCount["other"] ++
							rows = append(rowsAck, row...)
						}
					}
				}
			}
		}
		OPSGENIEOpenAlerts.Add(rowsOpenCount["total"])
		OPSGENIEOpenAlertsCritical.Add(rowsOpenCount["critical"])
		OPSGENIEOpenAlertsWarning.Add(rowsOpenCount["warning"])
		OPSGENIEOpenAlertsOther.Add(rowsOpenCount["other"])
		rowsCritical = append(rowsCritical, rowsWarning...)
		rowsCritical = append(rowsCritical, rows...)
		rowsCritical = append(rowsCritical, rowsAckCritical...)
		rowsCritical = append(rowsCritical, rowsAckWarning...)
		rowsCritical = append(rowsCritical, rowsAck...)
		// update grafana table
		grada.Table = []grada.TableResponse{
			{
				Columns: columns,
				Rows:    rowsCritical,
				Type:    "table",
			},
		}
		time.Sleep(time.Duration(fetchInterval) * time.Second)
	}
}
