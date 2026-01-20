# Makefile (Windows / Service-only)

APP_NAME = syncase.exe
BIN_DIR = bin
GO = go

.PHONY: build install start stop uninstall clean

build:
	@echo Building Syncase Silent Service...
	@if not exist "$(BIN_DIR)" mkdir "$(BIN_DIR)"
	$(GO) build -o $(BIN_DIR)/$(APP_NAME)

install:
	@echo Installing Windows Service...
	@$(BIN_DIR)/$(APP_NAME) install

start:
	@echo Starting Windows Service...
	@$(BIN_DIR)/$(APP_NAME) start

stop:
	@echo Stopping Windows Service...
	@$(BIN_DIR)/$(APP_NAME) stop

uninstall:
	@echo Uninstalling Windows Service...
	@$(BIN_DIR)/$(APP_NAME) uninstall

clean:
	@echo Cleaning build artifacts...
	@if exist "$(BIN_DIR)\$(APP_NAME)" del /Q "$(BIN_DIR)\$(APP_NAME)"
