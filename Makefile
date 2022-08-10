JS_PATH := dist/asma/main.js
ORIGINAL_WASM_FILENAME := asma_bg.wasm
ORIGINAL_WASM_PATH := dist/asma/$(ORIGINAL_WASM_FILENAME)
NEW_WASM_FILENAME := main.wasm
SLS := node_modules/.bin/serverless

build-asma::
	@mkdir -p dist/asma
	@wasm-pack build --target web asma

# Copy distribution files
	@cp asma/pkg/asma.js $(JS_PATH)
	@cp asma/pkg/$(ORIGINAL_WASM_FILENAME) dist/asma/$(NEW_WASM_FILENAME)

# Display build information
# @echo "JS size:" $(shell ls -lh $(JS_PATH) |awk '{print $$5}')
	@echo "VM size:" $(shell ls -lh dist/asma/$(ORIGINAL_WASM_FILENAME) |awk '{print $$5}')

# Replace URL
	@sed -i '' 's/$(ORIGINAL_WASM_FILENAME)/$(NEW_WASM_FILENAME)/g' $(JS_PATH)

# npm install -g esbuild
	@esbuild --minify --sourcemap $(JS_PATH) --allow-overwrite --outfile=$(JS_PATH)

# If we want to optimize the WASM binary
# @wasm-opt -Oz -o dist/asma/asma_bg_optimized.wasm dist/asma/asma_bg.wasm

build-proxy::
# @GOOS=linux CGO_ENABLED=0 go build -v -o dist/rproxy main.go
	@GOOS=linux CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" -o dist/rproxy main.go

proxy-optimize:
	which upx && upx -9 dist/rproxy || true

deploy-proxy:: build-proxy proxy-optimize
	AWS_PROFILE=fundamentei $(SLS) deploy -s production

# setup::
# 	@AWS_PROFILE=fundamentei \
# 		aws iam create-role \
# 			--role-name lambda-basic-execution \
# 			--assume-role-policy-document \
# 			file://aws/lambda-trust-policy.json
# 	@AWS_PROFILE=fundamentei \
# 		aws iam attach-role-policy \
# 			--role-name lambda-basic-execution \
# 			--policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
# 	@AWS_PROFILE=fundamentei aws iam get-role --role-name lambda-basic-execution
