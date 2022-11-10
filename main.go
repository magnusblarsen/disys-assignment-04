package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	MutexService "github.com/magnusblarsen/disys-assignment-04/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	WANTED   = 0
	RELEASED = 1
	HELD     = 2
)

func main() {
	file, err := os.OpenFile("ourLog", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer file.Close()

	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)


	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:      ownPort,
		clients: make(map[int32]MutexService.MutexServiceClient),
		ctx:     ctx,
		state:   RELEASED,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	grpcServer := grpc.NewServer()
	MutexService.RegisterMutexServiceServer(grpcServer, p)

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()


    	
    portfile, err := os.Open("port-addresses.txt")
    defer file.Close()
    if err != nil {
        log.Printf("could not read text file with ports: %v", err)
    }
    fileScanner := bufio.NewScanner(portfile)


	for fileScanner.Scan() {
		port, _ := strconv.ParseInt(fileScanner.Text(), 10, 32)
        port32 := int32(port)

		if port32 == ownPort {
			continue
		}

		var conn *grpc.ClientConn
		log.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %v", err)
		}
		defer conn.Close()
		c := MutexService.NewMutexServiceClient(conn)
		p.clients[port32] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if p.state == RELEASED && text == "enter" {
            log.Printf("Process with id %v is trying to get access\n", p.id)
			p.AttemptAccess()
		}
		if p.state == HELD && text == "exit" {
            log.Printf("Process with id %v is trying to exit\n", p.id)
			p.Exit()
		}
	}
}

type peer struct {
	MutexService.UnimplementedMutexServiceServer
	id           int32
	clients      map[int32]MutexService.MutexServiceClient
	ctx          context.Context
	state        int
	requestQueue []chan bool
}

func (p *peer) RequestAccess(ctx context.Context, req *MutexService.Request) (*MutexService.Reply, error) {
	if p.state == HELD || (p.state == WANTED && p.id < req.Id) {
		requestChan := make(chan bool)
		p.requestQueue = append(p.requestQueue, requestChan)
		<-requestChan
	}
	return &MutexService.Reply{Answer: true}, nil
}

func (p *peer) AttemptAccess() {
	p.state = WANTED

	request := &MutexService.Request{Id: p.id}
	for id, client := range p.clients {
		reply, err := client.RequestAccess(p.ctx, request)
		if err != nil {
            log.Printf("something went wrong: %v", err)
		}
		log.Printf("Got reply from id %v: %v\n", id, reply.Answer)
	}
	p.state = HELD
    log.Printf("process with id: %v now has access to the critical section", p.id)
}

func (p *peer) Exit() {
	p.state = RELEASED
	for _, request := range p.requestQueue {
		request <- true
	}
	p.requestQueue = make([]chan bool, 0)
}
