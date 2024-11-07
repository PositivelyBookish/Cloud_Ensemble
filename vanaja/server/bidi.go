package main

import (
	"io"
	"log"

	pb "Project/vanaja/proto"
)

func (s *server) AnalyzePatterns(stream pb.AgricultureService_AnalyzePatternsServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("Got request with crop type : %v", req.CropType)
		log.Printf("Got request with humidity : %v", req.Humidity)
		log.Printf("Got request with Id : %v", req.Id)
		log.Printf("Got request with soil moisture : %v", req.SoilMoisture)
		log.Printf("Got request with temperature : %v", req.Temperature)
		log.Printf("Got request with yield amount : %v", req.YieldAmount)
		res := &pb.AnalysisResult{
			Id:             req.Id + 1,
			Message:        "Hello " + req.CropType,
			PredictedYield: 112,
		}
		if err := stream.Send(res); err != nil {
			return err
		}
	}
}
