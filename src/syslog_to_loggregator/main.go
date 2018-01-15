package main

import (
	"fmt"
	"log"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/v1"
	"github.com/cloudfoundry/dropsonde"
	syslog "gopkg.in/mcuadros/go-syslog.v2"
)

func main() {
	fmt.Println("Start")

	// TODO is there a better type than this?
	// See https://docs.cloudfoundry.org/devguide/deploy-apps/streaming-logs.html#format
	SOURCE_TYPE := "RTR"

	// TODO this should be the host vm router instance number
	SOURCE_INSTANCE := "0"

	APP_ID := "haproxy"

	// TODO externalise these?
	dropsonde.Initialize("127.0.0.1:3457", APP_ID)

	log.Println("Creating loggregator client")
	client, err := v1.NewClient()
	if err != nil {
		log.Fatal("Could not create loggregator client", err)
	}

	fmt.Println("Sending test message to loggregator")
	client.EmitLog("syslog_to_loggregator startup test message",
		loggregator.WithAppInfo(APP_ID, SOURCE_TYPE, SOURCE_INSTANCE),
		loggregator.WithStdout(),
	)

	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	// default syslog port is 514, but this syslog is just for haproxy
	//TODO externalise this
	server.ListenUDP("0.0.0.0:515")
	// TODO is it better to use a unix socket instead of udp if possible?
	// https://github.com/mcuadros/go-syslog/blob/master/server.go#L102
	fmt.Println("Starting syslog server")
	err = server.Boot()
	if err != nil {
		log.Fatalf("Could not start syslog server", err)
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

			appId := "" // TODO is this value just always the APP_ID we initialize dropsonde with??
			if logParts["app_name"] != nil {
				appId = logParts["app_name"].(string)
			}
			client.EmitLog(
				message,
				loggregator.WithAppInfo(appId, SOURCE_TYPE, SOURCE_INSTANCE),
				loggregator.WithStdout(),
			)
		}

	}(channel)

	server.Wait()

}
