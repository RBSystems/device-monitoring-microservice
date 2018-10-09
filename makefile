# vars
ORG=$(shell echo $(CIRCLE_PROJECT_USERNAME))
BRANCH=$(shell echo $(CIRCLE_BRANCH))
NAME=$(shell echo $(CIRCLE_PROJECT_REPONAME))

ifeq ($(NAME),)
NAME := $(shell basename "$(PWD)")
endif

ifeq ($(ORG),)
ORG=byuoitav
endif

ifeq ($(BRANCH),)
BRANCH:= $(shell git rev-parse --abbrev-ref HEAD)
endif

# go
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
VENDOR=gvt fetch -branch $(BRANCH)

# angular
NPM=npm
NPM_INSTALL=$(NPM) install
NG_BUILD=ng build --prod --aot --build-optimizer 
NG1=dash

# aws
AWS_S3_ADD=aws s3 cp
S3_BUCKET=$(shell echo $(AWS_S3_SERVICES_BUCKET))

build: build-x86 build-arm build-web

build-x86:
	env GOOS=linux CGO_ENABLED=0 $(GOBUILD) -o $(NAME)-bin -v

build-arm:
	env GOOS=linux GOARCH=arm $(GOBUILD) -o $(NAME)-arm -v

build-web: $(NG1)
	cd $(NG1) && $(NPM_INSTALL) && $(NG_BUILD) --base-href="./$(NG1)/"
	mkdir files
	mv $(NG1)/dist files/$(NG1)-dist

test:
	$(GOTEST) -v -race $(go list ./... | grep -v /vendor/)

clean:
	$(GOCLEAN)
	rm -f $(NAME)-bin
	rm -f $(NAME)-arm
	rm -rf files/

run: $(NAME)-bin
	./$(NAME)-bin

deps:
	$(NPM_INSTALL) -g @angular/cli
ifneq "$(BRANCH)" "master"
	# put vendored packages in here
	# e.g. $(VENDOR) github.com/byuoitav/event-router-microservice
	$(VENDOR) github.com/byuoitav/authmiddleware
	$(VENDOR) github.com/byuoitav/touchpanel-ui-microservice
	$(VENDOR) github.com/byuoitav/common
	$(VENDOR) github.com/byuoitav/av-api
endif
	$(GOGET) -d -v

deploy: $(NAME)-arm $(NAME).service files/$(NG1)-dist
ifeq "$(BRANCH)" "master"
	$(eval BRANCH=development)
endif
	@echo adding files to $(S3_BUCKET)
	$(AWS_S3_ADD) $(NAME)-arm s3://$(S3_BUCKET)/$(BRANCH)/device-monitoring
	$(AWS_S3_ADD) $(NAME).service s3://$(S3_BUCKET)/$(BRANCH)/device-monitoring.service
	$(AWS_S3_ADD) files/ s3://$(S3_BUCKET)/$(BRANCH)/files/ --recursive
ifeq "$(BRANCH)" "development"
	$(eval BRANCH=master)
endif

### deps
$(NAME)-bin:
	$(MAKE) build-x86

$(NAME)-arm:
	$(MAKE) build-arm

$(NAME).service:


files/$(NG1)-dist:
	$(MAKE) build-web
