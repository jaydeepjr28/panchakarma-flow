# PowerShell Protobuf Generation Script
$ErrorActionPreference = "Stop"

Write-Host "Compiling Protocol Buffers with protoc..."

protoc -I. `
       --go_out=. --go_opt=module=github.com/jaydeepjr28/panchakarma-flow `
       --go-grpc_out=. --go-grpc_opt=module=github.com/jaydeepjr28/panchakarma-flow `
       proto/clinic/v1/clinic_service.proto `
       proto/appointment/v1/appointment_service.proto `
       proto/identity/v1/identity_service.proto

Write-Host "Protobuf compilation successful."
