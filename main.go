package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"pasientskyhosting/ps-opsgenie-grafana/grada"

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
	opsGenieAlertQuery := os.Getenv("OPSGENIE_ALERT_QUERY")
	if opsGenieAlertQuery == "" {
		opsGenieAlertQuery = "status:open"
	}
	// map severity tag values to integers for sorting in grafana
	priorityMap := map[alertsv2.Priority]int{
		alertsv2.P1: 1,
		alertsv2.P2: 2,
		alertsv2.P3: 3,
		alertsv2.P4: 4,
		alertsv2.P5: 5,
	}
	// OpsGenie client
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(apiKey)
	alertCli, _ := cli.AlertV2()
	dash := grada.GetDashboard()
	columns := []grada.Column{
		{Text: "Message", Type: "string"},
		{Text: "AlertType", Type: "string"},
		{Text: "Priority", Type: "int"},
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
	OPSGENIEOpenAlertsP1, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlertsP1", 10)
	if err != nil {
		log.Fatalln(err)
	}
	OPSGENIEOpenAlertsP2, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlertsP2", 10)
	if err != nil {
		log.Fatalln(err)
	}
	OPSGENIEOpenAlertsP3, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlertsP3", 10)
	if err != nil {
		log.Fatalln(err)
	}
	OPSGENIEOpenAlertsP4, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlertsP4", 10)
	if err != nil {
		log.Fatalln(err)
	}
	OPSGENIEOpenAlertsP5, err := dash.CreateMetricWithBufSize("OpsGenieOpenAlertsP5", 10)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		// alert counter by priority
		rowsOpenCount := map[alertsv2.Priority]float64{
			alertsv2.P1: 0,
			alertsv2.P2: 0,
			alertsv2.P3: 0,
			alertsv2.P4: 0,
			alertsv2.P5: 0,
		}
		// alert counter total
		var rowsOpenTotal float64
		// split acked and unacked alerts
		rows, rowsAck := []grada.Row{}, []grada.Row{}
		// get alert list from OpsGenie
		response, err := alertCli.List(alertsv2.ListAlertRequest{
			Limit:                100,
			Offset:               0,
			Query:                opsGenieAlertQuery,
			SearchIdentifierType: alertsv2.Name,
		})
		if err != nil {
			fmt.Println(err.Error())
			break
		} else {
			// get alert details - the Description field lives here
			for _, alert := range response.Alerts {
				response2, err2 := alertCli.Get(alertsv2.GetAlertRequest{
					Identifier: &alertsv2.Identifier{
						ID: alert.ID,
					},
				})
				if err2 != nil {
					fmt.Println(err2.Error())
					continue
				} else {
					// alert Details
					alertDetail := response2.Alert
					// pull out PS custom tags
					alertType := Find(alert.Tags, "alert_type: ")
					cluster := Find(alert.Tags, "cluster: ")
					hostname := Find(alert.Tags, "hostname: ")
					// create and append grafana row
					row := []grada.Row{
						{
							alert.Message,
							alertType,
							priorityMap[alert.Priority],
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
					// acked and unacked in their own slice
					if alert.Acknowledged {
						rowsAck = append(rowsAck, row...)
					} else {
						// total unacked
						rowsOpenTotal++
						// count by priority
						rowsOpenCount[alert.Priority]++
						rows = append(rows, row...)
					}

				}
			}
		}
		// add values to our metrics
		OPSGENIEOpenAlerts.Add(rowsOpenTotal)
		OPSGENIEOpenAlertsP1.Add(rowsOpenCount[alertsv2.P1])
		OPSGENIEOpenAlertsP2.Add(rowsOpenCount[alertsv2.P2])
		OPSGENIEOpenAlertsP3.Add(rowsOpenCount[alertsv2.P3])
		OPSGENIEOpenAlertsP4.Add(rowsOpenCount[alertsv2.P4])
		OPSGENIEOpenAlertsP5.Add(rowsOpenCount[alertsv2.P5])
		// sort unacked on priority
		sort.Slice(rows, func(i, j int) bool {
			return rows[i][2].(int) < rows[j][2].(int)
		})
		// sort acked on priority
		sort.Slice(rowsAck, func(i, j int) bool {
			return rowsAck[i][2].(int) < rowsAck[j][2].(int)
		})
		// put acked on the bottom
		rows = append(rows, rowsAck...)
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
