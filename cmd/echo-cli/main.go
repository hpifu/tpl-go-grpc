package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hpifu/go-kit/hflag"
	"github.com/hpifu/tpl-go-grpc/api"
	"google.golang.org/grpc"
)

var AppVersion = "unknown"

func main() {
	version := hflag.Bool("v", false, "print current version")
	address := hflag.String("h", "127.0.0.1:17060", "address")
	hflag.Parse()
	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	conn, err := grpc.Dial(*address, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("dial failed. err: [%v]\n", err)
		return
	}
	defer conn.Close()

	client := api.NewServiceClient(conn)

	res, err := client.Echo(context.Background(), &api.EchoReq{Rid: "1234567", Message: "hello world"})
	fmt.Println(res, err)
}
