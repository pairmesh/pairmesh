# Generate the database query interfaces.
queryset:
	go run ./tools/qs/main.go -in ./portal/db/models/models.go -out ./portal/db/models/autogen_query.go

# Generate the protocol buffer files
proto:
	@cd message/protos; \
    protoc --go_out=. *.proto; \
    protoc --go-grpc_out=. *.proto
