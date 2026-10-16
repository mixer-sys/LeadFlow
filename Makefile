.PHONY: proto

proto:
	rm -rf api/gen
	protoc \
		--go_out=. \
		--go_opt=module=leadflow \
		--go-grpc_out=. \
		--go-grpc_opt=module=leadflow \
		api/proto/leadflow/v1/lead.proto