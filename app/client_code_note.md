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
