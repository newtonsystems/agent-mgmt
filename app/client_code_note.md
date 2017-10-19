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
