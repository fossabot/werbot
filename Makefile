SHELL:= /bin/bash
ROOT_PATH:=$(abspath $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST))))))
GO_PATH:=$(shell go env GOPATH)
CPU_ARCH:=$(shell go env GOARCH)
OS_NAME:=$(shell go env GOHOSTOS)
include $(ROOT_PATH)/.vscode/config/.env

DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
GIT_COMMIT=$(shell cd "${ROOT_PATH}" && git rev-parse --short HEAD)
GIT_DIRTY=$(shell cd "${ROOT_PATH}" && test -n "$(git status --porcelain)" && echo "+CHANGES" || true)
GIT_TAG=$(shell cd "${ROOT_PATH}" && git describe --tags --abbrev=0 2>/dev/null)

VERSION=$(subst v,,${GIT_TAG})

## Output related vars
ifdef TERM
	BOLD:=$(shell tput bold)
	RED:=$(shell tput setaf 1)
	GREEN:=$(shell tput setaf 2)
	YELLOW:=$(shell tput setaf 3)
	RESET:=$(shell tput sgr0)
endif

# $(msg) bla bla bla   instead of   @echo bla bla bla
msg = @echo

# $(call file_exists,file-name)
# Return non-null if a file exists.
file_exists = $(wildcard $1)

# $(call make_target_dir,directory-name-opt)
# Create a directory if it doesn't exist.
make_target_dir = $(if $(call file-exists,$(if $1,$1,$(dir $@))),,mkdir -p $(if $1,$1,$(dir $@)))

# $(call get_latest_release,golangci/golangci-lint)
# Latest version release of package.
get_latest_release = $(shell curl --silent "https://api.github.com/repos/$1/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

.DEFAULT_GOAL:=help

#############################################################################
.PHONY: help
help: 
	@grep --no-filename -E '^[a-zA-Z_/-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
#############################################################################



#############################################################################
.PHONY: gen_key_aes
gen_key_aes: ## Generating AES key
	@openssl enc -aes-128-cbc -k secret -P -md sha1 -pbkdf2
#############################################################################


#############################################################################
.PHONY: gen_key_server
gen_key_server: ## Generating ssh server key
	@if [ -f $(ROOT_PATH)/docker/core/server.key ]; then \
		rm -rf $(ROOT_PATH)/docker/core/server.key*; \
	fi
	@ssh-keygen -t rsa -b 4096 -f $(ROOT_PATH)/docker/core/server_key -N '' -C 'werbot@core'
	@rm -rf $(ROOT_PATH)/docker/core/server_key.pub
	@mv $(ROOT_PATH)/docker/core/server_key $(ROOT_PATH)/docker/core/server.key

	$(msg) "$(YELLOW)Server key generated$(RESET)"
#############################################################################


#############################################################################
# run: 
# make gen_protos - recreate all protofiles
# make gen_protos user - recreate protofile user from folder /internal/grpc/proto/
.PHONY: gen_protos
gen_protos: ## Generating protos files
	@if [ $(filter-out $@,$(MAKECMDGOALS)) ]; then\
		if [ -d ${ROOT_PATH}/internal/grpc/proto/$(filter-out $@,$(MAKECMDGOALS))/ ];then\
			$(call _gen_protos,${ROOT_PATH}/internal/grpc/proto/$(filter-out $@,$(MAKECMDGOALS))/);\
		else \
			echo "error";\
		fi \
	else \
		for entry in ${ROOT_PATH}/internal/grpc/proto/*/; do\
			$(call _gen_protos,$${entry});\
		done \
	fi

	$(msg) "$(YELLOW)Proto-files updated$(RESET)"

define _gen_protos
	rm -f ${1}*.pb.go;\
	rm -f ${1}*.ts;\
	rm -f ${1}*.js;\
	rm -rf ${1}google;\
	pushd ${1} >/dev/null 2>&1;\
	echo "${1}*.proto";\
	protoc --proto_path=. \
	  --proto_path=/usr/local/include/ \
		--proto_path=${ROOT_PATH}/internal/grpc/proto/ \
		--go_out=paths=source_relative:. \
		--go-grpc_out=paths=source_relative:. \
		$$(basename ${1}).proto;\
	protoc --proto_path=. \
	  --proto_path=/usr/local/include/ \
		--proto_path=${ROOT_PATH}/internal/grpc/proto/ \
		--gotag_out=paths=source_relative:. \
		$$(basename ${1}).proto;\
	protoc --proto_path=. \
	  --proto_path=/usr/local/include/ \
		--proto_path=${ROOT_PATH}/internal/grpc/proto/ \
		--plugin=protoc-gen-ts=${ROOT_PATH}/web/node_modules/@protobuf-ts/plugin/bin/protoc-gen-ts \
		--ts_out=. \
		--ts_opt use_proto_field_name,ts_nocheck,long_type_string,force_optimize_code_size,force_client_none \
		$$(basename ${1}).proto;\
		popd >/dev/null 2>&1
endef
#############################################################################


#############################################################################
.PHONY: upd_protos
upd_protos:
	$(eval PROTOS_LATEST=$(call get_latest_release,protocolbuffers/protobuf))
	$(eval GOTAG_LATEST=$(call get_latest_release,srikrsna/protoc-gen-gotag))
	@case $(OS_NAME) in \
		darwin*) \
			brew install protobuf protoc-gen-go protoc-gen-go-grpc;\
			;; \
		linux) \
			$(call make_target_dir,${ROOT_PATH}/.vscode/tmp);\
			wget "https://github.com/protocolbuffers/protobuf/releases/download/${PROTOS_LATEST}/protoc-$(subst v,,${PROTOS_LATEST})-linux-x86_64.zip" -4 -q -O ${ROOT_PATH}/.vscode/tmp/protoc.zip;\
			unzip ${ROOT_PATH}/.vscode/tmp/protoc.zip -d ${ROOT_PATH}/.vscode/tmp/protoc3;\
			sudo mv ${ROOT_PATH}/.vscode/tmp/protoc3/bin/* /usr/local/bin/;\
			sudo rm -rf /usr/local/include/google;\
			sudo mv ${ROOT_PATH}/.vscode/tmp/protoc3/include/* /usr/local/include/;\
			go install google.golang.org/protobuf/cmd/protoc-gen-go@latest;\
			go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest;\
			go install github.com/srikrsna/protoc-gen-gotag@latest;\
			sudo rm -rf /usr/local/include/tagger;\
			sudo cp -a ${GO_PATH}/pkg/mod/github.com/srikrsna/protoc-gen-gotag@${GOTAG_LATEST}/tagger /usr/local/include/tagger/;\
			sudo chown $(USER) /usr/local/bin/protoc;\
			sudo chown -R $(USER) /usr/local/include/google /usr/local/include/tagger;\
			rm -rf ${ROOT_PATH}/.vscode/tmp;\
			;; \
	esac

	$(msg) "$(YELLOW)Protos tools updated$(RESET)"
#############################################################################


#############################################################################
.PHONY: gen_key_grpc
gen_key_grpc: ## Generating TLS keys for gRPC
	@$(call make_target_dir,${ROOT_PATH}/.vscode/tmp) 
	@echo "$$_gen_grpc_key_conf" > ${ROOT_PATH}/.vscode/tmp/.temp-openssl-config
	@openssl genrsa 2048 > ${ROOT_PATH}/.vscode/tmp/private_key.pem
	@openssl req -nodes -new -x509 -sha256 -days 1825 -config ${ROOT_PATH}/.vscode/tmp/.temp-openssl-config \
			-extensions 'req_ext' \
			-key ${ROOT_PATH}/.vscode/tmp/private_key.pem \
			-out ${ROOT_PATH}/.vscode/tmp/public_key.pem
	@mv ${ROOT_PATH}/.vscode/tmp/private_key.pem ${ROOT_PATH}/docker/core/grpc_private.key
	@mv ${ROOT_PATH}/.vscode/tmp/public_key.pem ${ROOT_PATH}/docker/core/grpc_public.key
	@rm -rf ${ROOT_PATH}/.vscode/tmp

	$(msg) "$(YELLOW)TLS keys for gRPC generated$(RESET)"

export _gen_grpc_key_conf
define _gen_grpc_key_conf
[ req ]
prompt                 = no
req_extensions         = req_ext
distinguished_name     = req_distinguished_name

[ req_distinguished_name ]
countryName            = US
stateOrProvinceName    = Delaware
localityName           = Middletown
organizationName       = Werbot, Inc.
organizationalUnitName = werbot
commonName             = werbot.com

[ req_ext ]
subjectAltName         = @alt_names
 
[ alt_names ]
DNS.1                  = werbot.com
endef
#############################################################################


#############################################################################
.PHONY: upd_geolite
upd_geolite: ## Updating and install GeoLite database to the latest version
	@if [ -f $(ROOT_PATH)/docker/core/GeoLite2-Country.mmdb ]; then \
		rm -rf $(ROOT_PATH)/docker/core/GeoLite2-Country.mmdb; \
	fi
	@export $(shell sed 's/=.*//' $(ROOT_PATH)/.vscode/config/.env)
	@$(call make_target_dir,${ROOT_PATH}/.vscode/tmp)
	@wget "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=$(GEOLITE_LICENSE)&suffix=tar.gz" -4 -q -O $(ROOT_PATH)/.vscode/tmp/country.tar.gz
	@tar -zxf $(ROOT_PATH)/.vscode/tmp/country.tar.gz -C $(ROOT_PATH)/.vscode/tmp
	@cp $$(ls -d $(ROOT_PATH)/.vscode/tmp/*/ | head -n 1)*.mmdb $(ROOT_PATH)/docker/core/GeoLite2-Country.mmdb
	@rm -rf $(ROOT_PATH)/.vscode/tmp
	$(msg) "$(YELLOW)Base GeoLite2-Country updated$(RESET)"
#############################################################################


#############################################################################
.PHONY: upd_golang
upd_golang: ## Updating and install Go to the latest version
	$(eval GO_RELEASE=$(shell wget -qO- "https://golang.org/dl/" | grep -v -E 'go[0-9\.]+(beta|rc)' | grep -E -o 'go[0-9\.]+' | grep -E -o '[0-9]\.[0-9]+(\.[0-9]+)?' | sort -V | uniq | tail -1))
	$(eval GO_PATH="/usr/local/go")
	@if [ ! -d $(GO_PATH) ]; then \
		sudo mkdir $$GO_PATH; \
		echo -e "\nexport PATH=\$$PATH:$(GO_PATH)/bin\n" >> ~/.bashrc; \
		echo -e "\nexport PATH=\$$PATH:\$$HOME/go/bin\n" >> ~/.bashrc; \
		source ~/.bashrc; \
	fi
	@curl --silent https://dl.google.com/go/go$(GO_RELEASE).$(OS_NAME)-$(CPU_ARCH).tar.gz | sudo tar -vxz --strip-components 1 -C $(GO_PATH) >/dev/null 2>&1
	$(msg) "$(YELLOW)GO updated$(RESET)"
#############################################################################


#############################################################################
.PHONY: new_build
new_build: ## Building new version to git
	$(eval NEW_VERSION=$(shell read -p "Enter new release version (current version ${VERSION}): " enter ; echo $${enter}))
	@if [ ${NEW_VERSION} ]; then\
		#sed -i -e "s/\(Licensed Work:\s*Werbot\s\)v[0-9][0-9.]*/\\1v${NEW_VERSION}/" $(ROOT_PATH)/LICENSE;\
		sed -i -e "s/\(Change Date:\s*\)[-0-9]\+/\\1$(shell date +'%Y-%m-%d' -d '4 years')/" $(ROOT_PATH)/LICENSE;\
		git add .;\
		git commit -a -m "meta: Create release";\
		git tag v${NEW_VERSION};\
		git push origin main;\
		git push --tags origin main;\
	else \
		echo "$(RED)Cloudflare ip lists updated$(RESET)";\
	fi
#############################################################################


#############################################################################
.PHONY: prod_build
prod_build: ## Building project in bin folder
	$(eval NAME=$(filter-out $@,$(MAKECMDGOALS)))
	@if [ ${NAME} ]; then\
		if [ -d ${ROOT_PATH}/cmd/${NAME}/ ];then\
			make -s prod_build_go ${NAME}; \
		elif [ ${NAME} == "app" ]; then\
			make -s prod_build_app;\
		else \
			echo "error";\
		fi \
	else \
		for entry in ${ROOT_PATH}/cmd/*/; do\
			make -s prod_build_go $$(basename $${entry});\
		done; \
		make -s prod_build_app;\
	fi

.PHONY: prod_build_go
prod_build_go:
	$(eval NAME=$(filter-out $@,$(MAKECMDGOALS)))
	@echo "Build goose_db_version" ${NAME} ${VERSION};\
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X github.com/werbot/werbot/internal/version.version=${VERSION} -X github.com/werbot/werbot/internal/version.gitCommit=${GIT_COMMIT} -X github.com/werbot/werbot/internal/version.buildDate=${BUILD_DATE}" -o ${ROOT_PATH}/bin/${NAME} ${ROOT_PATH}/cmd/${NAME};\
	upx -1 -k bin/${NAME} >/dev/null 2>&1;\
	rm -rf bin/${NAME}.~

.PHONY: prod_build_app
prod_build_app:
	@echo "Build web app"
	@cd ${ROOT_PATH}/web && yarn build
#############################################################################


#############################################################################
.PHONY: prod_package
prod_package: ## Building a docker container
	$(eval NAME=$(filter-out $@,$(MAKECMDGOALS)))
	@if [ ${NAME} ]; then \
		if [ -d ${ROOT_PATH}/cmd/${NAME}/ ];then\
			make -s prod_package_go ${NAME}; \
		elif [ ${NAME} == "app" ]; then\
			make -s prod_package_app; \
		else \
			echo "error";\
		fi \
	else \
		for entry in ${ROOT_PATH}/cmd/*/; do\
			make -s prod_package_go $$(basename $${entry});\
		done; \
		make -s prod_package_app; \
	fi

.PHONY: prod_package_go
prod_package_go:
	$(eval NAME=$(filter-out $@,$(MAKECMDGOALS)))
	$(eval DESCRIPTION=$(shell cat ${ROOT_PATH}/cmd/${NAME}/.description))
	@echo "Package go container" ${NAME} ${VERSION}
	@cat ${ROOT_PATH}/build/docker/Dockerfile_go > ${ROOT_PATH}/bin/Dockerfile_${NAME}
	@sed -i -E "s/_NAME_/${NAME}/g" ${ROOT_PATH}/bin/Dockerfile_${NAME}
	@sed -i -E "s/_GIT_COMMIT_/${GIT_COMMIT}/g" ${ROOT_PATH}/bin/Dockerfile_${NAME}
	@sed -i -E "s/_VERSION_/${VERSION}/g" ${ROOT_PATH}/bin/Dockerfile_${NAME}
	@sed -i -E "s/_DESCRIPTION_/${DESCRIPTION}/g" ${ROOT_PATH}/bin/Dockerfile_${NAME}
	docker build -f ${ROOT_PATH}/bin/Dockerfile_${NAME} -t ghcr.io/werbot/${NAME}:latest .
	docker tag ghcr.io/werbot/${NAME}:latest ghcr.io/werbot/${NAME}:${VERSION}
	rm -rf ${ROOT_PATH}/bin/${NAME}/
	rm ${ROOT_PATH}/bin/Dockerfile_${NAME}
	docker image prune --filter "dangling=true" --force

.PHONY: prod_package_app
prod_package_app:
	$(eval DESCRIPTION=$(shell awk -F'"' '/"description": ".+"/{ print $$4; exit; }' ${ROOT_PATH}/web/package.json))
	@echo "Package web app container" ${VERSION}
	@cat ${ROOT_PATH}/build/docker/Dockerfile_web > ${ROOT_PATH}/bin/Dockerfile_web
	@sed -i -E "s/_GIT_COMMIT_/${GIT_COMMIT}/g" ${ROOT_PATH}/bin/Dockerfile_web
	@sed -i -E "s/_VERSION_/${VERSION}/g" ${ROOT_PATH}/bin/Dockerfile_web
	@sed -i -E "s/_DESCRIPTION_/${DESCRIPTION}/g" ${ROOT_PATH}/bin/Dockerfile_web
	@cp -a ${ROOT_PATH}/internal/grpc/proto ${ROOT_PATH}/web/
	docker build -f ${ROOT_PATH}/bin/Dockerfile_web -t ghcr.io/werbot/app:latest .
	docker tag ghcr.io/werbot/app:latest ghcr.io/werbot/app:${VERSION}
	rm -rf ${ROOT_PATH}/web/dist ${ROOT_PATH}/web/proto
	rm ${ROOT_PATH}/bin/Dockerfile_web
	docker image prune --filter "dangling=true" --force
#############################################################################


#############################################################################
.PHONY: prod_push
prod_push: ## Submitting the project to the docker registry
	$(eval NAME=$(filter-out $@,$(MAKECMDGOALS)))
#	@export $(shell sed 's/=.*//' $(ROOT_PATH)/.vscode/config/.env)
#	@echo $(GITHUB_TOKEN) | docker login ghcr.io -u USERNAME --password-stdin
	@if [ ${NAME} ]; then \
		if [ -d ${ROOT_PATH}/cmd/${NAME}/ ];then\
			make -s prod_push_go ${NAME};\
		elif [ ${NAME} == "app" ]; then\
			make -s prod_push_app; \
		else \
			echo "error";\
		fi \
	else \
		for entry in ${ROOT_PATH}/cmd/*/; do\
			make -s prod_push_go $$(basename $${entry});\
		done; \
		make -s prod_push_app; \
	fi

.PHONY: prod_push_go
prod_push_go:
	$(eval NAME=$(filter-out $@,$(MAKECMDGOALS)))
	echo "Push go package" ${NAME} ${VERSION}
	docker push ghcr.io/werbot/${NAME}:latest
	docker push ghcr.io/werbot/${NAME}:${VERSION}

.PHONY: prod_push_app
prod_push_app:
	echo "Push web package"
	docker push ghcr.io/werbot/app:latest
	docker push ghcr.io/werbot/app:${VERSION}
#############################################################################


#############################################################################
.PHONY: lint
lint: ## Cleaning garbage and inactive containers
#	@REVIVE_FORCE_COLOR=1 revive -formatter friendly ./...
	@golangci-lint run
#############################################################################


#############################################################################
.PHONY: upd_cdn_ip
upd_cdn_ip: 
## Cloudflare ip lists from https://www.cloudflare.com/en-gb/ips/
	@echo -n >${ROOT_PATH}/docker/haproxy/cloudflare-ips.txt

	@for i in $(shell curl -s https://www.cloudflare.com/ips-v4); do\
		echo $$i >>${ROOT_PATH}/docker/haproxy/cloudflare-ips.txt;\
	done

	@for i in $(shell curl -s https://www.cloudflare.com/ips-v6); do\
		echo $$i >>${ROOT_PATH}/docker/haproxy/cloudflare-ips.txt;\
	done

	$(msg) "$(YELLOW)Cloudflare ip lists updated$(RESET)"
#############################################################################


#############################################################################
# make srv_migration ent up && make srv_migration saas up && make srv_migration test up
# make srv_migration test down && make srv_migration saas down && make srv_migration ent down
.PHONY: srv_migration
srv_migration:
	$(eval MIGRATION_DIR=${ROOT_PATH}/migration)
	$(eval DB_POSTFIX="goose_db_version")
	$(eval ARG_TYPE = $(filter ent saas test,$(MAKECMDGOALS)))
	$(eval ARG_GOOSE = $(filter create up up1 down down1 redo status,$(MAKECMDGOALS)))
	@if [ $(ARG_TYPE) ]; then\
		MIGRATION_DIR=${ROOT_PATH}/migration;\
		DB_POSTFIX="goose_db_version";\
		if [ "$(ARG_TYPE)" == "saas" ];then\
			MIGRATION_DIR=${MIGRATION_DIR}/saas;\
			DB_POSTFIX=${DB_POSTFIX}"_saas";\
		elif [ "$(ARG_TYPE)" == "test" ]; then\
			MIGRATION_DIR=${ROOT_PATH}/migration/test;\
			DB_POSTFIX=${DB_POSTFIX}"_test";\
		fi;\
		if [ $(ARG_GOOSE) ]; then\
			source ${ROOT_PATH}/.vscode/config/.env.buffet;\
			GOOSE_CMD="goose -dir $$MIGRATION_DIR -table $$DB_POSTFIX postgres "$$PSQLSERVER_DSN"";\
			if [ $(ARG_GOOSE) == "create" ]; then $$GOOSE_CMD create migration_name sql && $$GOOSE_CMD fix; fi;\
			if [ $(ARG_GOOSE) == "up" ]; then $$GOOSE_CMD up; fi;\
			if [ $(ARG_GOOSE) == "up1" ]; then $$GOOSE_CMD up-by-one; fi;\
			if [ $(ARG_GOOSE) == "down" ]; then $$GOOSE_CMD reset; fi;\
			if [ $(ARG_GOOSE) == "down1" ]; then $$GOOSE_CMD down; fi;\
			if [ $(ARG_GOOSE) == "redo" ]; then $$GOOSE_CMD redo; fi;\
			if [ $(ARG_GOOSE) == "status" ]; then $$GOOSE_CMD status; fi;\
		else \
			echo "Parameters not passed";\
		fi; \
	else \
		echo "Parameters not passed";\
	fi
#############################################################################


#############################################################################
.PHONY: upd_env_files
upd_env_files: 
	@if [ $(filter-out $@,$(MAKECMDGOALS)) ]; then\
		if [ -d ${ROOT_PATH}/cmd/$(filter-out $@,$(MAKECMDGOALS))/ ];then\
			$(call _upd_env_files,${ROOT_PATH}/cmd/$(filter-out $@,$(MAKECMDGOALS))/);\
		else \
			echo "error";\
		fi \
	else \
		for entry in ${ROOT_PATH}/cmd/*/; do\
			$(call _upd_env_files,$${entry});\
		done \
	fi

define _upd_env_files
	NAME=$$(basename ${1});\
	PARAMETERS=();\
	HEADER=FALSE;\
	ENV_FILE="${ROOT_PATH}/.vscode/config/.env.$$NAME";\
	echo "Scan $$NAME $$VERSION parameters";\
	for file in ${ROOT_PATH}/cmd/$$NAME/*.go; do\
		test -f "$$file" || continue;\
		PARAMETERS+="$$(awk '{while (match($$0, /(config.[a-zA-Z]+\("([_A-Z]+)[, "]+(?|[a-zA-Z0-9_:.\/]+|)(?|"\)|\)))/, result)){print result[2] "=" result[3];$$0 = sub($$0, "")}}' $$file) ";\
	done;\
	for i in $$(printf "%s\n" $$PARAMETERS | sort -u); do\
		PARAMETER_NAME=$$(echo $$i | cut -d= -f 1);\
		PARAMETER_ARGUMENT=$$(echo $$i | cut -d= -f 2);\
		if [ ! $$(grep $$PARAMETER_NAME $$ENV_FILE) ]; then\
			if [ $$HEADER == FALSE ]; then\
				echo -e "\n\n\n# New parameters from project files:" >>$$ENV_FILE;\
				HEADER=TRUE;\
			fi;\
			echo -e "$$PARAMETER_NAME=$$PARAMETER_ARGUMENT" >> $$ENV_FILE;\
		fi;\
	done
endef
#############################################################################


#############################################################################
.PHONY: upd_install
upd_install: 
	@cp -a ${ROOT_PATH}/docker/grafana ${ROOT_PATH}/scripts/install/cfg/
	@cp -a ${ROOT_PATH}/docker/haproxy/*.txt ${ROOT_PATH}/scripts/install/cfg/haproxy/
	@cp -a ${ROOT_PATH}/docker/haproxy/config.cfg ${ROOT_PATH}/scripts/install/cfg/haproxy/
	@cp -a ${ROOT_PATH}/docker/loki ${ROOT_PATH}/scripts/install/cfg/
	@cp -a ${ROOT_PATH}/docker/prometheus ${ROOT_PATH}/scripts/install/cfg/
	@cp -a ${ROOT_PATH}/docker/promtail ${ROOT_PATH}/scripts/install/cfg/
	@cp -a ${ROOT_PATH}/docker/docker-compose.yaml ${ROOT_PATH}/scripts/install/cfg/
	$(msg) "$(YELLOW)Install configs updated$(RESET)"
#############################################################################


#############################################################################
.PHONY: clean
clean: ## Cleaning garbage and inactive containers
#	@(lsof -t -i :5172 | xargs kill) 2>/dev/null 	
	@for pid in $$(echo $$(ps ax | grep __debug_bin | grep -v grep | awk '{print $$1}')); do \
		printf "%.25s " "killing $$pid ..................................."; \
		kill $$pid; \
		echo "killed"; \
	done

	@if [ -d $(ROOT_PATH)/web/dist ]; then \
			rm -rf $(ROOT_PATH)/web/dist; \
	fi

	@rm -rf $(ROOT_PATH)/bin/*
	@docker system prune --all --volumes --force
	@docker image prune --filter "dangling=true" --force
#############################################################################


#############################################################################
%: ## A parameter
	@true
#############################################################################
