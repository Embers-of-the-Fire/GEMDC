package main

import (
	"GEMDC/market"
	"fmt"
	"log"
	"os"
	"sync"
)

func main() {
	w := &sync.WaitGroup{}
	w.Add(1)
	go func() {
		sereLogFile, err := os.OpenFile("log-sere.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		serenityLogger := log.New(sereLogFile, "", log.LstdFlags)
		serenityServer := market.Server{
			ServerName: "serenity",
			LogPath:    "log-sere.log",
			Logger:     serenityLogger,
		}
		c := make(chan int, 1)
		wg := &sync.WaitGroup{}
		wg.Add(1)
		serenityServer.Start(true, c, wg)
		for i := range c {
			fmt.Printf("Received From Server@Serenity - CODE: %d\n", i)
		}
		wg.Wait()
		w.Done()
	}()
	w.Add(1)
	go func() {
		tranqLogFile, err := os.OpenFile("log-trans.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		tranquilityLogger := log.New(tranqLogFile, "", log.LstdFlags)
		tranquilityServer := market.Server{
			ServerName: "tranquility",
			LogPath:    "log-trans.log",
			Logger:     tranquilityLogger,
		}
		c := make(chan int, 1)
		wg := &sync.WaitGroup{}
		wg.Add(1)
		tranquilityServer.Start(true, c, wg)
		for i := range c {
			fmt.Printf("Received From Server@Tranquility - CODE: %d\n", i)
		}
		wg.Wait()
		w.Done()
	}()
	w.Wait()
}
