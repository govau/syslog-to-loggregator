package main

import (
	"flag"
	"fmt"
	"log"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/v1"
	"github.com/cloudfoundry/dropsonde"
	syslog "gopkg.in/mcuadros/go-syslog.v2"
)

func main() {
	log.Println("Start")

	var instanceIndex int
	var metronPort int
	var sendStartMessage bool
	var sourceName string
	var syslogPort int

	flag.IntVar(&instanceIndex, "instance-index", -1, "Instance number for this vm e.g. 0 - required")
	flag.IntVar(&metronPort, "metron-port", 3457, "Metron agent port")
	flag.BoolVar(&sendStartMessage, "send-start-message", false, "Send a message on application start to loggregator. Might be useful for debugging")
	flag.StringVar(&sourceName, "source-name", "", "Logging source name used for all logs sent to loggregator - required")
	flag.IntVar(&syslogPort, "syslog-port", -1, "Port to start the syslog server on - required")
	flag.Parse()

	if instanceIndex < 0 {
		log.Fatal("--instance-index is required and must be greater than 0.")
	}

	if sourceName == "" {
		log.Fatal("--source-name is required")
	}

	if syslogPort < 0 {
		log.Fatal("--syslog-port is required and must be greater than 0.")
	}

	// TODO is there a better type than this?
	// See https://docs.cloudfoundry.org/devguide/deploy-apps/streaming-logs.html#format
	SOURCE_TYPE := "RTR"

	metronAddress := fmt.Sprintf("127.0.0.1:%d", metronPort)
	syslogServerAddress := fmt.Sprintf("127.0.0.1:%d", syslogPort)
	log.Println("Metron address: " + metronAddress)
	log.Println("Syslog server address: " + syslogServerAddress)

	err := dropsonde.Initialize(metronAddress, sourceName)
	if err != nil {
		log.Fatal("Could not initialize dropsonde", err)
	}

	client, err := v1.NewClient()
	if err != nil {
		log.Fatal("Could not create loggregator client", err)
	}

	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP(syslogServerAddress)

	log.Println("Starting syslog server")
	err = server.Boot()
	if err != nil {
		log.Fatalf("Could not start syslog server", err)
	}

	if sendStartMessage {
		client.EmitLog("syslog_to_loggregator started",
			loggregator.WithAppInfo(sourceName, SOURCE_TYPE, fmt.Sprint(instanceIndex)),
			loggregator.WithStdout(),
		)
	}

	go func(channel syslog.LogPartsChannel) {
		var message string
		for logParts := range channel {
			// c, _ := json.Marshal(logParts)
			// fmt.Println("Logparts:", string(c))
			// example output:
			// Logparts: {"app_name":"haproxy","client":"127.0.0.1:39179","facility":16,"hostname":"90d47cf8-5e75-42b2-af2c-70e956e892d4","message":"127.0.0.1:44536 [16/Jan/2018:05:59:19.199] http http/\u003cNOSRV\u003e 0/-1/-1/-1/0 301 91 - - LR-- 1/1/0/0/0 0/0 \"HEAD /foo HTTP/1.1\"","msg_id":"-","priority":134,"proc_id":"74238","severity":6,"structured_data":"-","timestamp":"2018-01-16T05:59:19Z","tls_peer":"","version":1}

			if logParts["message"] != nil {
				message = logParts["message"].(string)
			} else if logParts["content"] != nil {
				message = logParts["content"].(string)
			} else {
				continue
			}
			client.EmitLog(
				message,
				loggregator.WithAppInfo(sourceName, SOURCE_TYPE, fmt.Sprint(instanceIndex)),
				loggregator.WithStdout(),
			)
		}

	}(channel)

	server.Wait()

}
