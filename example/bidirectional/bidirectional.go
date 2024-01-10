package main

import (
	"fmt"
	"log"
	"time"

	"github.com/joe-at-startupmedia/posix_mq"
)

const maxSendTickNum = 10

var (
	mqSend *posix_mq.MessageQueue
	mqResp *posix_mq.MessageQueue
)

func main() {
	resp_c := make(chan int)
	go responder(resp_c)
	//wait for the responder to create the posix_mq files
	time.Sleep(1 * time.Second)
	send_c := make(chan int)
	go sender(send_c)
	<-resp_c
	<-send_c
}

func responder(c chan int) {
	if err := openQueues(); err != nil {
		c <- 1
	}
	defer func() {
		fmt.Println("Responder: finished")
		c <- 0
	}()

	count := 0
	for {
		count++
		msg, _, err := mqSend.Receive()
		if err != nil {
			fmt.Printf("Responder: error handling message: %s\n", err)
			continue
		}

		fmt.Printf("Responder: got new message from sender: %s\n", msg)

		mqResp.Send([]byte(fmt.Sprintf("Farewell, World : %d\n", count)), 0)
		if err != nil {
			fmt.Printf("Responder: errorsending responde: %s\n", err)
			continue
		}

		fmt.Println("Responder: sent a response")

		if count >= maxSendTickNum {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func sender(c chan int) {
	if err := openQueues(); err != nil {
		c <- 1
	}
	defer func() {
		closeQueue(mqSend)
		closeQueue(mqResp)
		fmt.Println("Sender: finished and unlinked")
		c <- 0
	}()

	count := 0
	for {
		count++
		err := mqSend.Send([]byte(fmt.Sprintf("Hello, World : %d\n", count)), 0)
		if err != nil {
			fmt.Printf("Sender: error sending message: %s\n", err)
			continue
		}

		fmt.Println("Sender: sent a new message")

		msg, _, err := mqResp.TimedReceive(time.Now().Local().Add(time.Second * time.Duration(1)))

		if err != nil {
			fmt.Printf("Sender: error receiving message: %s\n", err)
			continue
		}

		fmt.Printf("Sender: got a response: %s\n", msg)

		if count >= maxSendTickNum {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func openQueues() error {
	mqs, err := openQueue("send")
	if err != nil {
		return err
	}
	mqr, err := openQueue("resp")
	if err != nil {
		return err
	}
	mqSend = mqs
	mqResp = mqr
	return nil
}

func openQueue(postfix string) (*posix_mq.MessageQueue, error) {
	oflag := posix_mq.O_RDWR | posix_mq.O_CREAT
	posixMQFile := "posix_mq_example_" + postfix
	return posix_mq.NewMessageQueue("/"+posixMQFile, oflag, 0666, nil)
}

func closeQueue(mq *posix_mq.MessageQueue) {
	err := mq.Unlink()
	if err != nil {
		log.Println(err)
	}
}