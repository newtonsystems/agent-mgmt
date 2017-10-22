// Some notes on why the code is writtent the way it is


``` go


clientService := createTestClient(t, conn)


	resp, err := client.GetAvailableAgents(ctx, session, mongoDBDatabase) //	grpc.Header(&header),    // will retrieve header
	//grpc.Trailer(&trailer),  // will retrieve trailer)


func createTestClient(t *testing.T, conn *grpc.ClientConn) service.Service {
	var getAvailableAgentsEndpoint endpoint.Endpoint
	{
		getAvailableAgentsEndpoint = grpctransport.NewClient(
			conn, "grpc_types.AgentMgmt", "GetAvailableAgents",
			transport.EncodeGRPCGetAvailableAgentsRequest,
			transport.DecodeGRPCGetAvailableAgentsResponse,
			grpc_types.GetAvailableAgentsResponse{},
		).Endpoint()
	}
	var getAgentIDFromRefEndpoint endpoint.Endpoint
	{
		getAgentIDFromRefEndpoint = grpctransport.NewClient(
			conn, "grpc_types.AgentMgmt", "GetAgentIDFromRef",
			transport.EncodeGRPCGetAvailableAgentsRequest,
			transport.DecodeGRPCGetAvailableAgentsResponse,
			grpc_types.GetAgentIDFromRefResponse{},
		).Endpoint()
	}

	return amendpoint.Set{
		GetAvailableAgentsEndpoint: getAvailableAgentsEndpoint,
		GetAgentIDFromRefEndpoint:  getAgentIDFromRefEndpoint,
	}
}
```


Useful client code maybe

```go
func checkClient(t *testing.T, session models.Session, client grpc_types.AgentMgmtClient, source, golden string, description string) {
	_, err := ioutil.ReadFile(source)

	if err != nil {
		t.Error(err)
		return
	}

	var header, trailer metadata.MD
	ctx := context.Background()

	resp, err := client.GetAvailableAgents(
		ctx,
		&grpc_types.GetAvailableAgentsRequest{Limit: 10},
		grpc.Header(&header),   // will retrieve header
		grpc.Trailer(&trailer), // will retrieve trailer
	)

	//md, ok := metadata.FromIncomingContext(ctx)
	if resp != nil {
		logger.Log("resp", fmt.Sprintf("\n%#v", resp.AgentIds), "err", err, "header", fmt.Sprintf("\n%#v", header), "trailer", fmt.Sprintf("\n%#v", trailer))
	}
	if err != nil {
		logger.Log("msg", "err is not nil", "header", fmt.Sprintf("\n%#v", header), "trailer", fmt.Sprintf("\n%#v", trailer))
		if val, ok := trailer["errortype"]; ok {
			//do something here

			logger.Log("msg", "errortype found key")
			logger.Log("msg", val[0])
			errortype, err := strconv.Atoi(val[0])
			errortype2 := amerrors.ErrorType(errortype)
			if err != nil {
				logger.Log("msg", "failed conversitony")
			}
			switch errortype2 {
			case amerrors.ErrAgentIDNotFound:
				logger.Log("msg", "ErrAgentIDNotFoundError")

			}
		}
		s, ok := status.FromError(err)
		if !ok {
			logger.Log("msg", status.Errorf(codes.Internal, "client call: unknown %v", err))
		}
		switch s.Code() {
		case codes.Unknown:

			logger.Log("msg", status.Errorf(codes.Internal, "client call: invalid argyemet %v", s), "errortype", "hj")
		}
	}
}
```

useful create a server and a sprivifc gokit type client reequires endpoint as a service


```go
// createTestServer
func createTestServer(t *testing.T) {
	// Initialise mongo connection
	moSession := tests.CreateTestMongoConnection(*debug)
	defer moSession.Refresh()
	defer moSession.Close()

	// Create Service &  Endpoints (no logger, tracer, metrics etc)
	var (
		service   = service.NewService(nil, nil, nil, nil, nil)
		endpoints = amendpoint.NewEndpoint(service, nil, nil, nil, moSession, "test")
	)

	// gRPC server
	go func() {
		ln, err := net.Listen("tcp", hostPort)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		srv := transport.GRPCServer(endpoints, nil, nil)
		s := grpc.NewServer()
		grpc_types.RegisterAgentMgmtServer(s, srv)
		//defer s.GracefulStop()

		s.Serve(ln)
	}()

}

func createTestClient(t *testing.T, conn *grpc.ClientConn) service.Service {
	var getAvailableAgentsEndpoint endpoint.Endpoint
	{
		getAvailableAgentsEndpoint = grpctransport.NewClient(
			conn, "grpc_types.AgentMgmt", "GetAvailableAgents",
			transport.EncodeGRPCGetAvailableAgentsRequest,
			transport.DecodeGRPCGetAvailableAgentsResponse,
			grpc_types.GetAvailableAgentsResponse{},
		).Endpoint()
	}
	var getAgentIDFromRefEndpoint endpoint.Endpoint
	{
		getAgentIDFromRefEndpoint = grpctransport.NewClient(
			conn, "grpc_types.AgentMgmt", "GetAgentIDFromRef",
			transport.EncodeGRPCGetAvailableAgentsRequest,
			transport.DecodeGRPCGetAvailableAgentsResponse,
			grpc_types.GetAgentIDFromRefResponse{},
		).Endpoint()
	}

	return amendpoint.Set{
		GetAvailableAgentsEndpoint: getAvailableAgentsEndpoint,
		GetAgentIDFromRefEndpoint:  getAgentIDFromRefEndpoint,
	}
}


// in endpoints
// GetAvailableAgentsRequest implements the service interface, so Set may be used as a
// service. This is primarily useful in the context of a client library.
func (s Set) GetAvailableAgents(ctx context.Context, session models.Session, db string) ([]string, error) {
	resp, err := s.GetAvailableAgentsEndpoint(ctx, GetAvailableAgentsRequest{Limit: 10})
	if err != nil {
		var empty []string
		return empty, err
	}
	response := resp.(GetAvailableAgentsResponse)
	return response.AgentIds, err
}

// GetAgentIDFromRef implements the service interface, so Set may be used as a
// service. This is primarily useful in the context of a client library.
func (s Set) GetAgentIDFromRef(sess models.Session, db string, refID string) (int32, error) {
	resp, _ := s.GetAgentIDFromRefEndpoint(nil, GetAgentIDFromRefRequest{RefId: refID})
	response := resp.(GetAgentIDFromRefResponse)
	return response.AgentId, response.Err
}

// HeartBeat implements the service interface, so Set may be used as a
// service. This is primarily useful in the context of a client library.
func (s Set) HeartBeat(session models.Session, db string, agent models.Agent) (grpc_types.HeartBeatResponse_HeartBeatStatus, error) {
	resp, _ := s.HeartBeatEndpoint(nil, HeartBeatRequest{Agent: agent})
	response := resp.(HeartBeatResponse)
	return response.Status, response.Message
}


// -- grpc.go

// -- GetAvailableAgents() Client functions

// EncodeGRPCGetAvailableAgentsRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain sum request to a gRPC sum request. Primarily useful in a client.
func EncodeGRPCGetAvailableAgentsRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(endpoint.GetAvailableAgentsRequest)
	return &grpc_types.GetAvailableAgentsRequest{Limit: int32(req.Limit)}, nil
}

// DecodeGRPCGetAvailableAgentsResponse is a transport/grpc.DecodeResponseFunc that converts a
// gRPC sum reply to a user-domain sum response. Primarily useful in a client.
func DecodeGRPCGetAvailableAgentsResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*grpc_types.GetAvailableAgentsResponse)
	return endpoint.GetAvailableAgentsResponse{AgentIds: reply.AgentIds}, nil //grpc.Errorf(codes.InvalidArgument, "Ouch!") //, Err: str2err(reply.Err)}, nil
}

```
