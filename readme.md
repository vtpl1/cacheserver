### Create project
```bash
npm create vite@latest phoring -- --template react-ts
```
```bash
npm i @chakra-ui/react @emotion/react @emotion/styled framer-motion @reduxjs/toolkit react-redux d3
npm i --include=dev @types/d3
```
```bash
protoc -I ./interface \
   --go_out ./proto \
   --go_opt paths=source_relative \
   --go-grpc_out ./proto \
   --go-grpc_opt paths=source_relative \
   ./interface/hello_world.proto

protoc ./interface/api/v1/*.proto \
      --go_out=. \
      --go_opt=paths=source_relative \
      --go-grpc_out=. \
      --go-grpc_opt=paths=source_relative \
      --grpc-gateway_out . \
      --grpc-gateway_opt logtostderr=true \
      --grpc-gateway_opt paths=source_relative \
      --grpc-gateway_opt generate_unbound_methods=true
```

https://earthly.dev/blog/golang-grpc-gateway/