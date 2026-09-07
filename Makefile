# HAR-X — capture (Go) + analyzer (Python/Streamlit)

CAPTURE_DIR   := capture
ANALYZER_DIR  := analyzer
CAPTURES_OUT  := captures
BIN           := $(CAPTURE_DIR)/bin/harx-capture
VENV          := $(ANALYZER_DIR)/.venv
PY            := $(VENV)/bin/python
PIP           := $(VENV)/bin/pip
STREAMLIT     := $(VENV)/bin/streamlit

URL   ?=
TITLE ?=

.PHONY: help build build-capture build-analyzer \
        run run-capture \
        tidy fmt vet test clean clean-captures

help:
	@echo "HAR-X — available targets:"
	@echo "  make build            build everything (capture binary + analyzer venv)"
	@echo "  make build-capture    go build the capture tool"
	@echo "  make build-analyzer   create venv + pip install analyzer deps"
	@echo "  make run-capture      run capture (URL=... TITLE=... optional)"
	@echo "  make run              run the Streamlit analyzer"
	@echo "  make tidy             go mod tidy for the capture module"
	@echo "  make fmt              gofmt the capture module"
	@echo "  make vet              go vet the capture module"
	@echo "  make clean            remove built binaries and the analyzer venv"
	@echo "  make clean-captures   delete everything in captures/ (careful!)"

build: build-capture build-analyzer

build-capture: $(CAPTURES_OUT)
	cd $(CAPTURE_DIR) && go build -o bin/harx-capture ./cmd/harx-capture

build-analyzer: $(VENV)/bin/activate

$(VENV)/bin/activate:
	python3 -m venv $(VENV)
	$(PIP) install --upgrade pip -q
	$(PIP) install -q -r $(ANALYZER_DIR)/requirements.txt

$(CAPTURES_OUT):
	mkdir -p $(CAPTURES_OUT)

run-capture: build-capture
	./$(BIN) -out $(CAPTURES_OUT) \
		$(if $(URL),-url $(URL)) \
		$(if $(TITLE),-title $(TITLE))

run: build-analyzer
	cd $(ANALYZER_DIR) && ../$(VENV)/bin/streamlit run app.py

tidy:
	cd $(CAPTURE_DIR) && go mod tidy

fmt:
	cd $(CAPTURE_DIR) && gofmt -l -w .

vet:
	cd $(CAPTURE_DIR) && go vet ./...

test:
	cd $(CAPTURE_DIR) && go test ./...

clean:
	rm -f $(BIN)
	rm -rf $(VENV)

clean-captures:
	rm -f $(CAPTURES_OUT)/*.har