proto:
	rm -rf pb/*/*.go
	protoc -I ./modules --go_out=pb --go_opt paths=source_relative --go-grpc_out=pb \
	--go-grpc_opt paths=source_relative --grpc-gateway_out=pb \
	--grpc-gateway_opt paths=source_relative ./modules/*/*.proto

clean:
	rm -rf pb/*/*.go

run_server:
	go run server_grpc_gateway/main.go

run_client:
	go run client/main.go

