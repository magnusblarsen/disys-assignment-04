package main

import (
	"bufio"
	"context"
	"fmt"
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
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + 5000

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

	for i := 0; i < 3; i++ {
		port := int32(5000) + int32(i)

		if port == ownPort {
			continue
		}

		var conn *grpc.ClientConn
		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := MutexService.NewMutexServiceClient(conn)
		p.clients[port] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "enter" {
			p.AttemptAccess()
		}
		if text == "e" {
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
			fmt.Println("something went wrong")
		}
		fmt.Printf("Got reply from id %v: %v\n", id, reply.Answer)
	}
	p.state = HELD
	fmt.Printf("Got Access")
}

func (p *peer) Exit() {
	p.state = RELEASED
	for _, request := range p.requestQueue {
		request <- true
	}
	p.requestQueue = make([]chan bool, 0)
}
