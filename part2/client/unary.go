package main
	
import(
	"context"
	"log"
	"time"
	
	pb "github.com/hjani-2003/Cloud_Computing_Project/tree/harman/part2/proto"
)

func callSayHello(client pb.GreetServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
	res, err := client.SayHello(ctx, &pb.NoParam{})
	if err != nil {
		log. Fatalf("could not greet: %v", err)
	}
	log.Printf("%s", res.Message)
	
}