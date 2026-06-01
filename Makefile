.PHONY: css css-watch run build

# Build the Tailwind CSS bundle (web/static/css/output.css is generated, not committed).
css:
	npm run build:css

# Rebuild CSS on every template/input.css change. Run alongside `make run` while developing.
css-watch:
	npm run dev:css

# Build CSS, then run the Go server (styles are always fresh).
run: css
	go run ./cmd

# Build CSS, then compile the server binary.
build: css
	go build -o nex-gen-cms ./cmd
